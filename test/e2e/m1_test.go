// Package e2e 的 M1 用例：连接向导（状态分流 → rootDSE 探测 → 空/存量分支 →
// 档案加密落盘 → 热生效）、OU 增删改、删除保护与级联、移动预览、关键字搜索、
// CSRF 防护（R1.3~R1.5 / R2.1~R2.5 / R11.1~R11.2）。
package e2e

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"newldap/internal/config"
)

// newWizardEnv 构造"未配置"环境（无 env / 无配置文件），并用 t.Chdir 把
// data/config.yaml 的写入重定向到临时目录，避免污染仓库。
func newWizardEnv(t *testing.T) *env {
	t.Helper()
	cfg := &config.Config{}
	cfg.Server.SessionTTL = time.Minute
	cfg.Server.AuditFile = auditFile
	cfg.Profile.BaseDN = ""

	e := newEnvWithCfg(t, cfg)
	t.Chdir(t.TempDir())
	return e
}

func TestSetupWizardFlow(t *testing.T) {
	config.SetSecretKeySource("e2e-test-key")
	t.Cleanup(func() { config.SetSecretKeySource("") })

	e := newWizardEnv(t)

	// 1) 未配置 → 向导入口（R1.5 入口侧分流）
	st := e.do(t, "GET", "/api/v1/setup/status", nil, 200)
	if st["configured"] != false {
		t.Fatalf("初始应未配置: %v", st)
	}

	// 2) 探测坏地址 → ok=false 且是中文白话（R1.4 的错误路径）
	bad := e.do(t, "POST", "/api/v1/probe", map[string]any{"url": "ldap://127.0.0.1:1"}, 200)
	if bad["ok"] != false || bad["error"] == "" {
		t.Fatalf("坏地址探测 = %v", bad)
	}

	// 3) 探测替身：rootDSE + bind 测试 + base 状态（存量目录分支）
	pr := e.do(t, "POST", "/api/v1/probe", map[string]any{
		"url": e.addr, "bindDN": "cn=admin," + root, "bindPassword": "admin123", "baseDN": root,
	}, 200)
	if pr["ok"] != true || pr["bind"] != "ok" || pr["paging"] != true {
		t.Fatalf("探测结果异常: %v", pr)
	}
	base, _ := pr["base"].(map[string]any)
	if base == nil || base["exists"] != true || base["hasChildren"] != true {
		t.Fatalf("存量目录 base 状态 = %v", base)
	}

	// 3b) 错误密码 → fail 且防枚举文案
	pr2 := e.do(t, "POST", "/api/v1/probe", map[string]any{
		"url": e.addr, "bindDN": "cn=admin," + root, "bindPassword": "错误密码",
	}, 200)
	if pr2["bind"] != "fail" || !strings.Contains(fmt.Sprint(pr2["bindError"]), "账号或密码错误") {
		t.Fatalf("错误密码探测 = %v", pr2)
	}

	// 4) 保存档案（R1.3 + R11.2 密码加密落盘）
	saved := e.do(t, "POST", "/api/v1/setup/profile", map[string]any{
		"name": "测试连接", "url": e.addr, "bindDN": "cn=admin," + root,
		"bindPassword": "admin123", "baseDN": root, "rememberPassword": true,
	}, 200)
	if saved["saved"] != true || saved["plaintextSaved"] != false {
		t.Fatalf("保存结果 = %v", saved)
	}
	b, _ := os.ReadFile(filepath.Join("data", "config.yaml"))
	if strings.Contains(string(b), "admin123") {
		t.Fatal("配置文件泄漏明文密码！")
	}
	if !strings.Contains(string(b), "enc:v1:") {
		t.Fatal("配置文件未使用加密格式")
	}

	// 5) 已配置 → status 变化 + 向导端点对匿名关闭
	st2 := e.do(t, "GET", "/api/v1/setup/status", nil, 200)
	if st2["configured"] != true {
		t.Fatalf("保存后应已配置: %v", st2)
	}
	e.do(t, "POST", "/api/v1/probe", map[string]any{"url": e.addr}, 403)

	// 6) 热生效：登录用刚保存的档案成功建立会话
	e.do(t, "POST", "/api/v1/auth/login", map[string]string{"bindDN": "cn=admin," + root, "password": "admin123"}, 200)
	me := e.do(t, "GET", "/api/v1/me", nil, 200)
	if me["baseDN"] != root {
		t.Fatalf("me.baseDN = %v（档案未热生效？）", me["baseDN"])
	}
}

// TestProbeEmptyDirectory 覆盖 R1.5 空目录分支：base 条目不存在 → exists=false。
func TestProbeEmptyDirectory(t *testing.T) {
	e := newWizardEnv(t)
	pr := e.do(t, "POST", "/api/v1/probe", map[string]any{
		"url": e.addr, "bindDN": "cn=admin," + root, "bindPassword": "admin123",
		"baseDN": "ou=不存在的分支," + root,
	}, 200)
	base, _ := pr["base"].(map[string]any)
	if base == nil || base["exists"] != false {
		t.Fatalf("空分支探测 = %v", base)
	}
}

func TestOUCrudMoveAndDelete(t *testing.T) {
	e := newEnv(t)
	e.do(t, "POST", "/api/v1/auth/login", map[string]string{"bindDN": "cn=admin," + root, "password": "admin123"}, 200)

	// 新建 OU（R2.3）
	e.do(t, "POST", "/api/v1/entry", map[string]any{
		"dn": "ou=m1dept," + root, "objectClass": []string{"organizationalUnit"},
		"attrs": map[string][]string{"description": {"M1 测试部门"}},
	}, 201)

	// 树计数（R2.1）：m1dept=0；tech ≥4
	tree := e.do(t, "GET", "/api/v1/tree?base="+root, nil, 200)
	kids, _ := tree["children"].([]any)
	for _, k := range kids {
		n, _ := k.(map[string]any)
		if n["dn"] == "ou=m1dept,"+root && n["childCount"] != float64(0) {
			t.Errorf("m1dept childCount = %v", n["childCount"])
		}
		if n["dn"] == "ou=tech,"+root {
			if cc, _ := n["childCount"].(float64); cc < 4 {
				t.Errorf("tech childCount = %v，应 ≥4", n["childCount"])
			}
		}
	}

	// 编辑描述（属性级修改，R2.3）
	entry := e.do(t, "GET", "/api/v1/entry?dn=ou=m1dept,"+root, nil, 200)
	e.do(t, "PUT", "/api/v1/entry", map[string]any{
		"dn": "ou=m1dept," + root, "expectedCSN": entry["entryCSN"],
		"changes": []map[string]any{{"op": "replace", "attr": "description", "vals": []string{"改过的描述"}}},
	}, 200)
	entry2 := e.do(t, "GET", "/api/v1/entry?dn=ou=m1dept,"+root, nil, 200)
	attrs, _ := entry2["attrs"].(map[string]any)
	if d, _ := attrs["description"].([]any); len(d) == 0 || d[0] != "改过的描述" {
		t.Errorf("描述修改未生效: %v", attrs["description"])
	}

	// 移动预览 + 移动（R2.4）
	pv := e.do(t, "POST", "/api/v1/entry/move-preview", map[string]any{
		"dn": "ou=m1dept," + root, "newParent": "ou=product," + root,
	}, 200)
	if pv["newDN"] != "ou=m1dept,ou=product,"+root {
		t.Fatalf("移动预览 newDN = %v", pv["newDN"])
	}
	moved := e.do(t, "POST", "/api/v1/entry/move", map[string]any{
		"dn": "ou=m1dept," + root, "newParent": "ou=product," + root,
	}, 200)
	if moved["dn"] != "ou=m1dept,ou=product,"+root {
		t.Fatalf("移动结果 = %v", moved["dn"])
	}

	// 改名（R2.3 重命名）
	e.do(t, "POST", "/api/v1/entry/move", map[string]any{
		"dn": "ou=m1dept,ou=product," + root, "newRDN": "ou=m1dept2",
	}, 200)
	e.do(t, "GET", "/api/v1/entry?dn=ou=m1dept2,ou=product,"+root, nil, 200)

	// 删除保护（R2.5）：tech 非空 → 409；预览列子孙；级联后清空
	prev := e.do(t, "GET", "/api/v1/entry/delete-preview?dn=ou=tech,"+root, nil, 200)
	if prev["subtree"] != float64(4) || prev["empty"] != false {
		t.Fatalf("删除预览 = %v", prev)
	}
	e.do(t, "POST", "/api/v1/entry/delete", map[string]any{"dn": "ou=tech," + root, "cascade": false}, 409)
	e.do(t, "POST", "/api/v1/entry/delete", map[string]any{"dn": "ou=tech," + root, "cascade": true}, 200)
	e.do(t, "GET", "/api/v1/entry?dn=ou=tech,"+root, nil, 404)

	// 叶子删除
	e.do(t, "POST", "/api/v1/entry/delete", map[string]any{"dn": "ou=m1dept2,ou=product," + root, "cascade": false}, 200)

	// 关键字搜索（R2.2）：搜 hanmei 命中 1 条；窗口参数生效
	// （tech 部已在上面级联删除，剩余工号 E1011/E1013 恰好 2 条 e10 命中）
	res := e.do(t, "GET", "/api/v1/search?q=hanmei&attr=cn&attr=uid", nil, 200)
	if res["total"] != float64(1) {
		t.Fatalf("关键字搜索 total = %v", res["total"])
	}
	res2 := e.do(t, "GET", "/api/v1/search?q=e10&attr=uid&limit=2", nil, 200)
	if res2["total"] != float64(2) {
		t.Errorf("工号搜索 total = %v，应为 2", res2["total"])
	}
	entries, _ := res2["entries"].([]any)
	if len(entries) > 2 {
		t.Errorf("limit=2 窗口未生效，返回 %d 条", len(entries))
	}

	// 审计：新增 create/delete 事件，无密码泄漏
	auditHas(t, `"op":"create"`, `"op":"delete"`, `"op":"move"`)
}

// TestCSRFGuard 覆盖 R11.1：写方法 Origin 与 Host 不一致 → 403；一致/缺失 → 放行。
func TestCSRFGuard(t *testing.T) {
	e := newEnv(t)
	u, _ := url.Parse(e.ts.URL)

	// 恶意 Origin → 403（在到达业务逻辑前被拦）
	req := map[string]string{"bindDN": "cn=admin," + root, "password": "admin123"}
	e.doWithOrigin(t, "POST", "/api/v1/auth/login", req, "http://evil.example", 403)
	// 同源 Origin → 通过 CSRF 层（凭据正确则 200）
	e.doWithOrigin(t, "POST", "/api/v1/auth/login", req, u.String(), 200)
	// 无 Origin（curl 等）→ 放行
	e.do(t, "POST", "/api/v1/auth/login", req, 200)
}

func auditHas(t *testing.T, wants ...string) {
	t.Helper()
	b, err := os.ReadFile(auditFile)
	if err != nil {
		t.Fatal(err)
	}
	s := string(b)
	for _, w := range wants {
		if !strings.Contains(s, w) {
			t.Errorf("审计日志缺少 %s", w)
		}
	}
	if strings.Contains(s, "admin123") {
		t.Error("审计日志泄漏密码！")
	}
}

// TestSearchNeverLeaksPasswords 安全回归：搜索在任何参数姿势下都不得返回
// userPassword（默认属性集 / 显式请求敏感属性 / 服务端多回属性三条防线）。
func TestSearchNeverLeaksPasswords(t *testing.T) {
	e := newEnv(t)
	e.do(t, "POST", "/api/v1/auth/login", map[string]string{"bindDN": "cn=admin," + root, "password": "admin123"}, 200)

	for name, path := range map[string]string{
		"默认属性集":    "/api/v1/search?filter=(objectClass=inetOrgPerson)",
		"显式请求敏感属性": "/api/v1/search?filter=(objectClass=inetOrgPerson)&attr=userPassword&attr=cn",
		"通配符属性":    "/api/v1/search?filter=(objectClass=inetOrgPerson)&attr=*&attr=cn",
	} {
		res := e.do(t, "GET", path, nil, 200)
		entries, _ := res["entries"].([]any)
		if len(entries) == 0 {
			t.Fatalf("[%s] 无结果，测试无效", name)
		}
		for _, it := range entries {
			m, _ := it.(map[string]any)
			attrs, _ := m["attrs"].(map[string]any)
			for k := range attrs {
				if strings.EqualFold(k, "userPassword") {
					t.Errorf("[%s] 响应泄漏了 userPassword（dn=%v）", name, m["dn"])
				}
			}
		}
	}
}

// TestEntryDetailNeverLeaksPasswords 安全回归：条目详情（GET /api/v1/entry）
// 读取全量属性，同样不得把 userPassword 带回浏览器。
func TestEntryDetailNeverLeaksPasswords(t *testing.T) {
	e := newEnv(t)
	e.do(t, "POST", "/api/v1/auth/login", map[string]string{"bindDN": "cn=admin," + root, "password": "admin123"}, 200)

	for name, dn := range map[string]string{
		"人员条目": "uid=zhangwei,ou=tech," + root,
		"管理员条目": "cn=admin," + root,
	} {
		res := e.do(t, "GET", "/api/v1/entry?dn="+url.QueryEscape(dn), nil, 200)
		attrs, _ := res["attrs"].(map[string]any)
		if len(attrs) == 0 {
			t.Fatalf("[%s] 无属性，测试无效（响应 %v）", name, res)
		}
		for k := range attrs {
			if strings.EqualFold(k, "userPassword") {
				t.Errorf("[%s] GET /api/v1/entry 泄漏了 userPassword（dn=%s）", name, dn)
			}
		}
	}
}
