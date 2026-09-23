// R8.6 并存回归（M4）：以"直连 LDAP 的独立客户端"扮演 LAM，
// 与本工具 API 交替读写同一目录，断言双向可见、冲突拦截、互不破坏。
// 真实 OpenLDAP + LAM 的 GUI 级回归在 CI 中以等价的 ldapmodify/ldapsearch
// 语义覆盖（LAM 底层即标准 LDAP 操作；本套用例锁定的正是这些线协议不变量）。
package e2e

import (
	"strconv"
	"testing"

	ldap "github.com/go-ldap/ldap/v3"

	"newldap/internal/ldapclient"
)

// lamClient 是"LAM 侧"：绕过本工具的一切 API，直接标准 LDAP 操作。
type lamClient struct{ c *ldapclient.Client }

func newLam(t *testing.T, e *env) *lamClient {
	t.Helper()
	c, err := ldapclient.Dial(ldapclient.Options{URL: e.addr})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(c.Close)
	if err := c.Bind("cn=admin,"+root, "admin123"); err != nil {
		t.Fatal(err)
	}
	return &lamClient{c}
}

func TestLamCoexistence(t *testing.T) {
	e := newEnv(t)
	e.do(t, "POST", "/api/v1/auth/login", map[string]string{"bindDN": "cn=admin," + root, "password": "admin123"}, 200)
	lam := newLam(t, e)

	// ── 1. LAM 建人 → 本工具可见（R8.1 标准操作互见）
	lamDN := "uid=lamuser,ou=product," + root
	if err := lam.c.AddEntry(lamDN, map[string][]string{
		"objectClass":  {"top", "person", "organizationalPerson", "inetOrgPerson"},
		"uid":          {"lamuser"},
		"cn":           {"LAM 建的用户"},
		"sn":           {"用户"},
		"mail":         {"lamuser@example.cn"},
		"title":        {"LAM 侧录入的职务"},
		"userPassword": {"LamPass123!"},
	}); err != nil {
		t.Fatal(err)
	}
	res := e.do(t, "GET", "/api/v1/search?q=lamuser", nil, 200)
	if res["total"] != float64(1) {
		t.Fatalf("LAM 建的人本工具不可见: %v", res["total"])
	}

	// ── 2. 本工具改人 → LAM 侧读到的就是新值；且 LAM 写的其他字段原样（R8.2）
	if _, err := e.putChange(t, lamDN, "mobile", "139-0000-0001"); err != nil {
		t.Fatal(err)
	}
	le, err := lam.c.ReadEntry(lamDN, nil)
	if err != nil {
		t.Fatal(err)
	}
	if le.GetAttributeValue("mobile") != "139-0000-0001" {
		t.Error("本工具的修改 LAM 侧不可见")
	}
	if le.GetAttributeValue("title") != "LAM 侧录入的职务" {
		t.Error("本工具的修改覆盖了 LAM 写入的字段（违反非破坏纪律）")
	}

	// ── 3. LAM 先改 → 本工具带旧 CSN 提交 → 409 拦截；刷新后提交成功（R8.5）
	det := e.do(t, "GET", "/api/v1/entry?dn="+lamDN, nil, 200)
	stale := det["entryCSN"].(string)
	if err := lam.c.ModifyAttributes(lamDN, []ldap.Change{
		{Operation: ldap.ReplaceAttribute, Modification: ldap.PartialAttribute{Type: "title", Vals: []string{"LAM 又改了"}}},
	}); err != nil {
		t.Fatal(err)
	}
	e.do(t, "PUT", "/api/v1/entry", map[string]any{
		"dn": lamDN, "expectedCSN": stale,
		"changes": []map[string]any{{"op": "replace", "attr": "mail", "vals": []string{"new@example.cn"}}},
	}, 409)
	// 刷新重交
	det2 := e.do(t, "GET", "/api/v1/entry?dn="+lamDN, nil, 200)
	e.do(t, "PUT", "/api/v1/entry", map[string]any{
		"dn": lamDN, "expectedCSN": det2["entryCSN"],
		"changes": []map[string]any{{"op": "replace", "attr": "mail", "vals": []string{"new@example.cn"}}},
	}, 200)
	le2, _ := lam.c.ReadEntry(lamDN, nil)
	if le2.GetAttributeValue("title") != "LAM 又改了" || le2.GetAttributeValue("mail") != "new@example.cn" {
		t.Error("冲突后的合并提交未保住双方修改")
	}

	// ── 4. LAM 移动（modrdn）→ 本工具树/搜索跟到新位置（R8.1）
	newDN := "uid=lamuser,ou=market," + root
	if err := lam.c.ModRDN(lamDN, "uid=lamuser", "ou=market,"+root); err != nil {
		t.Fatal(err)
	}
	res2 := e.do(t, "GET", "/api/v1/search?q=lamuser", nil, 200)
	entries, _ := res2["entries"].([]any)
	if len(entries) != 1 {
		t.Fatalf("LAM 移动后本工具搜不到: %v", res2["total"])
	}
	if entries[0].(map[string]any)["dn"] != newDN {
		t.Errorf("本工具看到的 DN 未跟随移动: %v", entries[0])
	}

	// ── 5. 本工具删除保护对 LAM 建的数据同样生效（R8.4 行为一致）
	e.do(t, "POST", "/api/v1/entry/delete", map[string]any{"dn": newDN, "cascade": false}, 200) // 叶子直接删
	if _, err := lam.c.ReadEntry(newDN, []string{"1.1"}); err == nil {
		t.Error("本工具删除后 LAM 侧仍能读到")
	}

	// ── 6. 双方各自建/删交错后目录终态一致
	if err := lam.c.AddEntry("ou=lamdept,"+root, map[string][]string{
		"objectClass": {"top", "organizationalUnit"}, "ou": {"lamdept"},
	}); err != nil {
		t.Fatal(err)
	}
	e.do(t, "POST", "/api/v1/entry", map[string]any{
		"dn": "ou=ourdept," + root, "objectClass": []string{"organizationalUnit"},
		"attrs": map[string][]string{"ou": {"ourdept"}},
	}, 201)
	tree := e.do(t, "GET", "/api/v1/tree?base="+root, nil, 200)
	treeStr := fmtJSON(tree)
	for _, want := range []string{"lamdept", "ourdept"} {
		if !contains(treeStr, want) {
			t.Errorf("目录终态缺少 %s（双方写入应同时可见）", want)
		}
	}
	// 清理
	lam.c.DeleteEntry("ou=lamdept," + root) //nolint:errcheck
	e.do(t, "POST", "/api/v1/entry/delete", map[string]any{"dn": "ou=ourdept," + root, "cascade": false}, 200)
}

// putChange 用本工具 API 做一次属性级修改。
func (e *env) putChange(t *testing.T, dn, attr, val string) (string, error) {
	t.Helper()
	det := e.do(t, "GET", "/api/v1/entry?dn="+dn, nil, 200)
	csn, _ := det["entryCSN"].(string)
	e.do(t, "PUT", "/api/v1/entry", map[string]any{
		"dn": dn, "expectedCSN": csn,
		"changes": []map[string]any{{"op": "replace", "attr": attr, "vals": []string{val}}},
	}, 200)
	return csn, nil
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(s) > 0 && indexOfStr(s, sub) >= 0)
}

func indexOfStr(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}

// TestSelfServicePassword 自助改密 API（R9）：成功流程、防枚举文案、限流 429、审计。
func TestSelfServicePassword(t *testing.T) {
	e := newEnv(t)

	// 成功：旧密码验证 → 新密码生效
	e.do(t, "POST", "/api/v1/selfservice/password", map[string]string{
		"account": "zhangwei", "oldPassword": "Passw0rd!", "newPassword": "SelfNew#66q",
	}, 200)
	dn := "uid=zhangwei,ou=tech," + root
	e.do(t, "POST", "/api/v1/auth/login", map[string]string{"bindDN": dn, "password": "Passw0rd!"}, 401)
	e.do(t, "POST", "/api/v1/auth/login", map[string]string{"bindDN": dn, "password": "SelfNew#66q"}, 200)

	// 弱密码 → 400 带友好文案
	res := e.do(t, "POST", "/api/v1/selfservice/password", map[string]string{
		"account": "zhangwei", "oldPassword": "SelfNew#66q", "newPassword": "short",
	}, 400)
	if msg, _ := res["error"].(string); !contains(msg, "8 位") {
		t.Errorf("弱密码文案 = %q", msg)
	}

	// 错误旧密码 → 401；与不存在账号的文案一致（防枚举）
	r1 := e.do(t, "POST", "/api/v1/selfservice/password", map[string]string{
		"account": "zhangwei", "oldPassword": "错的", "newPassword": "Abcd1234!x",
	}, 401)
	r2 := e.do(t, "POST", "/api/v1/selfservice/password", map[string]string{
		"account": "ghost_user", "oldPassword": "错的", "newPassword": "Abcd1234!x",
	}, 401)
	if r1["error"] != r2["error"] {
		t.Errorf("防枚举文案不一致: %q vs %q", r1["error"], r2["error"])
	}

	// 连续失败触发账号限流 → 429（上限 5 次/10 分钟）
	for i := 0; i < 5; i++ {
		e.do(t, "POST", "/api/v1/selfservice/password", map[string]string{
			"account": "wangqiang", "oldPassword": "错" + strconv.Itoa(i), "newPassword": "Abcd1234!x",
		}, 401)
	}
	locked := e.do(t, "POST", "/api/v1/selfservice/password", map[string]string{
		"account": "wangqiang", "oldPassword": "Passw0rd!", "newPassword": "Abcd1234!x",
	}, 429)
	if msg, _ := locked["error"].(string); !contains(msg, "稍后") && !contains(msg, "过多") {
		t.Errorf("限流文案 = %q", msg)
	}

	// 审计：改密事件（成功+失败），无任何密码值
	auditHas(t, `"op":"self-password-change"`, `"result":"fail"`)
	b := readAudit(t)
	if contains(b, "SelfNew#66q") || contains(b, "Abcd1234!x") {
		t.Error("审计泄漏了密码值！")
	}
}
