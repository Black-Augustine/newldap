// M2 验收 e2e：人员领域（创建/详情反查组/重置密码/禁用启用/调动）、
// 用户组领域（场景化建组/gid 分配/成员维护）、schema 表单模型、LDIF 粘贴建条目。
package e2e

import (
	"fmt"
	"os"
	"strings"
	"testing"
)

func TestPeopleLifecycle(t *testing.T) {
	e := newEnv(t)
	e.do(t, "POST", "/api/v1/auth/login", map[string]string{"bindDN": "cn=admin," + root, "password": "admin123"}, 200)

	// 1. schema 表单模型（R5.2）
	fm := e.do(t, "GET", "/api/v1/schema/form?class=inetOrgPerson", nil, 200)
	fields, _ := fm["fields"].([]any)
	if len(fields) < 5 {
		t.Fatalf("表单字段过少: %v", fields)
	}
	var cnField map[string]any
	for _, f := range fields {
		m := f.(map[string]any)
		if m["name"] == "cn" {
			cnField = m
		}
	}
	if cnField == nil || cnField["required"] != true || cnField["label"] != "姓名 / 名称" {
		t.Errorf("cn 字段 = %+v", cnField)
	}

	// 2. 新建人员（R3.1）
	created := e.do(t, "POST", "/api/v1/people", map[string]any{
		"cn": "许倩", "uid": "xuqian", "ou": "ou=tech," + root,
		"mail": "xuqian@example.cn", "employeeNumber": "E1021", "title": "算法工程师",
	}, 201)
	dn, _ := created["dn"].(string)
	initial, _ := created["initialPassword"].(string)
	if dn == "" || len(initial) != 12 {
		t.Fatalf("创建返回 = %+v", created)
	}
	// 初始密码可用于 bind（错误初始密码登录失败 + 正确的可通过）
	e.do(t, "POST", "/api/v1/auth/login", map[string]string{"bindDN": dn, "password": "错误的"}, 401)
	e.do(t, "POST", "/api/v1/auth/login", map[string]string{"bindDN": dn, "password": initial}, 200)
	// 回到管理员会话
	e.do(t, "POST", "/api/v1/auth/login", map[string]string{"bindDN": "cn=admin," + root, "password": "admin123"}, 200)

	// 重复 uid 拦截
	e.do(t, "POST", "/api/v1/people", map[string]any{"cn": "许", "uid": "xuqian", "ou": "ou=tech," + root}, 400)

	// 3. 详情：属性 + 所属组反查（R4.2 反向视图）
	det := e.do(t, "GET", "/api/v1/people?dn="+dn, nil, 200)
	attrs, _ := det["attrs"].(map[string]any)
	if attrs["mail"] == nil || attrs["userPassword"] != nil {
		t.Errorf("详情属性异常: mail=%v userPassword=%v", attrs["mail"], attrs["userPassword"])
	}
	zd := e.do(t, "GET", "/api/v1/people?dn=uid=zhangwei,ou=tech,"+root, nil, 200)
	groups, _ := zd["groups"].([]any)
	if len(groups) == 0 {
		t.Fatal("张伟的所属组反查为空（应含 vpn-access）")
	}

	// 4. 重置密码（R3.3）
	rst := e.do(t, "POST", "/api/v1/people/password", map[string]string{"dn": dn}, 200)
	if pw, _ := rst["password"].(string); len(pw) != 12 {
		t.Errorf("随机重置 = %q", pw)
	}

	// 5. 禁用 → 本人登录被拒；启用 → 新密码恢复（R3.4）
	e.do(t, "POST", "/api/v1/people/disable", map[string]string{"dn": dn}, 200)
	e.do(t, "POST", "/api/v1/auth/login", map[string]string{"bindDN": dn, "password": pw2(rst)}, 401)
	e.do(t, "POST", "/api/v1/people/enable", map[string]string{"dn": dn}, 200)
	e.do(t, "POST", "/api/v1/auth/login", map[string]string{"bindDN": dn, "password": pw2(rst)}, 200)
	e.do(t, "POST", "/api/v1/auth/login", map[string]string{"bindDN": "cn=admin," + root, "password": "admin123"}, 200)

	// 6. 调动部门（R3.5）
	moved := e.do(t, "POST", "/api/v1/people/dept", map[string]string{
		"dn": dn, "newParent": "ou=product," + root,
	}, 200)
	if moved["dn"] != "uid=xuqian,ou=product,"+root {
		t.Errorf("调动结果 = %v", moved["dn"])
	}

	// 7. 审计含人员事件且无密码值
	auditHas(t, `"op":"create-person"`, `"op":"reset-password"`, `"op":"disable"`, `"op":"enable"`)
	b := readAudit(t)
	if strings.Contains(b, initial) {
		t.Error("审计泄漏初始密码！")
	}
}

func pw2(m map[string]any) string { s, _ := m["password"].(string); return s }

func TestGroupsLifecycle(t *testing.T) {
	e := newEnv(t)
	e.do(t, "POST", "/api/v1/auth/login", map[string]string{"bindDN": "cn=admin," + root, "password": "admin123"}, 200)

	// 权限组：无成员被拒（schema MUST member）
	e.do(t, "POST", "/api/v1/groups", map[string]any{"name": "空组", "scenario": "权限组"}, 400)

	// 建权限组（带首名成员）
	g := e.do(t, "POST", "/api/v1/groups", map[string]any{
		"name": "代码管理员", "scenario": "权限组", "description": "Git 仓库管理权限",
		"members": []string{"uid=zhangwei,ou=tech," + root},
	}, 201)
	if g["dn"] != "cn=代码管理员,ou=groups,"+root {
		t.Fatalf("dn = %v", g["dn"])
	}

	// 建登录组（自动 gid）
	e.do(t, "POST", "/api/v1/groups", map[string]any{
		"name": "开发机登录", "scenario": "登录组", "memberUids": []string{"zhangwei"},
	}, 201)

	// 列表
	list := e.do(t, "GET", "/api/v1/groups", nil, 200)
	groups, _ := list["groups"].([]any)
	if len(groups) < 3 { // 种子 vpn-access + 新建两个
		t.Fatalf("组数量 = %d", len(groups))
	}

	// 成员增删（属性级）
	e.do(t, "POST", "/api/v1/groups/members", map[string]any{
		"dn":      "cn=vpn-access,ou=groups," + root,
		"changes": map[string]any{"add": []string{"uid=lina,ou=tech," + root}},
	}, 200)
	e.do(t, "POST", "/api/v1/groups/members", map[string]any{
		"dn":      "cn=vpn-access,ou=groups," + root,
		"changes": map[string]any{"remove": []string{"uid=lina,ou=tech," + root}},
	}, 200)

	// 人员详情反查新组
	zd := e.do(t, "GET", "/api/v1/people?dn=uid=zhangwei,ou=tech,"+root, nil, 200)
	zg, _ := zd["groups"].([]any)
	hasNew := false
	for _, g := range zg {
		if strings.Contains(fmt.Sprint(g), "代码管理员") {
			hasNew = true
		}
	}
	if !hasNew {
		t.Errorf("张伟反查不到新建组: %v", zg)
	}
}

func TestLDIFCreate(t *testing.T) {
	e := newEnv(t)
	e.do(t, "POST", "/api/v1/auth/login", map[string]string{"bindDN": "cn=admin," + root, "password": "admin123"}, 200)

	ldif := "dn: ou=数据中心," + root + "\nobjectClass: top\nobjectClass: organizationalUnit\nou: 数据中心\ndescription: LDIF 粘贴创建"
	res := e.do(t, "POST", "/api/v1/entry/ldif", map[string]string{"ldif": ldif}, 201)
	if res["dn"] != "ou=数据中心,"+root {
		t.Errorf("dn = %v", res["dn"])
	}
	// 可读回
	ent := e.do(t, "GET", "/api/v1/entry?dn=ou=数据中心,"+root, nil, 200)
	a, _ := ent["attrs"].(map[string]any)
	if a["description"] == nil {
		t.Error("LDIF 建的条目属性缺失")
	}
	// 坏 LDIF → 400
	e.do(t, "POST", "/api/v1/entry/ldif", map[string]string{"ldif": "没有dn行"}, 400)
}

func readAudit(t *testing.T) string {
	t.Helper()
	b, err := os.ReadFile(auditFile)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}
