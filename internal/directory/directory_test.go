package directory

import (
	"errors"
	"strings"
	"testing"

	ldap "github.com/go-ldap/ldap/v3"

	"newldap/internal/ldapclient"
	"newldap/internal/ldaptest"
	"newldap/internal/schema"
)

const root = "dc=example,dc=cn"

func newService(t *testing.T) (*Service, *ldaptest.Server) {
	t.Helper()
	srv, err := ldaptest.Start(true)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(srv.Stop)
	c, err := ldapclient.Dial(ldapclient.Options{URL: srv.Addr()})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(c.Close)
	if err := c.Bind("cn=admin,"+root, "admin123"); err != nil {
		t.Fatal(err)
	}
	return New(c), srv
}

// TestTreeAndEntry 覆盖 M0 冒烟链路的核心读路径：
// 登录（bind）→ 树浏览（一层子节点）→ 读取一条条目。
func TestTreeAndEntry(t *testing.T) {
	s, _ := newService(t)

	kids, err := s.Children(root)
	if err != nil {
		t.Fatal(err)
	}
	var ous []string
	for _, n := range kids {
		if n.Name == "tech" || n.Name == "product" || n.Name == "groups" {
			ous = append(ous, n.Name)
		}
	}
	if len(ous) != 3 {
		t.Errorf("根下应含 tech/product/groups，实际: %+v", kids)
	}
	// tech 有子部门（M0 种子里没有 → HasChildren=false；补一个再验）
	if err := s.C.AddEntry("ou=platform,ou=tech,"+root, map[string][]string{
		"objectClass": {"top", "organizationalUnit"}, "ou": {"platform"},
	}); err != nil {
		t.Fatal(err)
	}
	kids, _ = s.Children(root)
	for _, n := range kids {
		if n.Name == "tech" && !n.HasChildren {
			t.Error("tech 应标记 hasChildren")
		}
	}

	// 读取条目
	dn := "uid=zhangwei,ou=tech," + root
	d, err := s.Get(dn)
	if err != nil {
		t.Fatal(err)
	}
	if d.CSN == "" {
		t.Error("条目详情缺 entryCSN")
	}
	if d.Attrs["mail"][0] != "zhangwei@example.cn" {
		t.Errorf("mail = %v", d.Attrs["mail"])
	}
}

// TestUpdateConflictDetection 是 spike④ 的验收测试：
// entryCSN 冲突检测（编辑期间被他人修改 → 拒绝并返回 ErrConflict）。
func TestUpdateConflictDetection(t *testing.T) {
	s, _ := newService(t)
	dn := "uid=zhangwei,ou=tech," + root

	// 管理员 A 打开编辑页（读到 csn1）
	d, err := s.Get(dn)
	if err != nil {
		t.Fatal(err)
	}
	csnA := d.CSN

	// 期间 LAM/管理员 B 直接改了这条（另开连接模拟）
	s2s, err := ldaptest.Start(false)
	if err != nil {
		t.Fatal(err)
	}
	defer s2s.Stop()
	_ = s2s // 同一进程内 store 不共享；改用本连接直接修改模拟 B
	if err := s.C.ModifyAttributes(dn, []ldap.Change{
		{Operation: ldap.ReplaceAttribute, Modification: ldap.PartialAttribute{Type: "title", Vals: []string{"被他人先改了"}}},
	}); err != nil {
		t.Fatal(err)
	}

	// A 提交：携带过期 csn1 → 必须被拦截
	_, err = s.Update(dn, csnA, []Change{{Op: "replace", Attr: "mobile", Vals: []string{"139-0000-0000"}}})
	if !errors.Is(err, ErrConflict) {
		t.Fatalf("过期 CSN 提交应返回 ErrConflict，得到: %v", err)
	}

	// 刷新后用新 CSN 提交 → 成功，且 title（B 的修改）未被覆盖
	d2, _ := s.Get(dn)
	newCSN, err := s.Update(dn, d2.CSN, []Change{{Op: "replace", Attr: "mobile", Vals: []string{"139-0000-0000"}}})
	if err != nil {
		t.Fatal(err)
	}
	if newCSN == "" || newCSN == d2.CSN {
		t.Errorf("更新后应返回新 CSN: %q → %q", d2.CSN, newCSN)
	}
	d3, _ := s.Get(dn)
	if d3.Attrs["title"][0] != "被他人先改了" {
		t.Error("B 的 title 修改被覆盖了（违反 D3 非破坏纪律）")
	}
	if d3.Attrs["mobile"][0] != "139-0000-0000" {
		t.Errorf("mobile 更新未生效: %v", d3.Attrs["mobile"])
	}
}

// TestMove 覆盖 modrdn 经领域层的包装（R2.4/R3.5 调动部门）。
func TestMove(t *testing.T) {
	s, _ := newService(t)
	dn := "uid=lina,ou=tech," + root

	newDN, err := s.Move(MoveRequest{DN: dn, NewParent: "ou=product," + root})
	if err != nil {
		t.Fatal(err)
	}
	if newDN != "uid=lina,ou=product,"+root {
		t.Errorf("newDN = %q", newDN)
	}
	if _, err := s.Get(newDN); err != nil {
		t.Fatalf("移动后读取新 DN 失败: %v", err)
	}

	// 改名
	renamed, err := s.Move(MoveRequest{DN: newDN, NewRDN: "uid=lina.li"})
	if err != nil {
		t.Fatal(err)
	}
	if renamed != "uid=lina.li,ou=product,"+root {
		t.Errorf("renamed = %q", renamed)
	}
}

// TestSchemaFromDirectory 验证 subschema 读取 + 解析器组合（spike② 的服务端路径）。
func TestSchemaFromDirectory(t *testing.T) {
	s, _ := newService(t)
	raw, err := s.C.Subschema()
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("subschema 原始条数: objectClasses=%d attributeTypes=%d", len(raw["objectClasses"]), len(raw["attributeTypes"]))
	sm, err := schema.Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	if sm.ObjectClass("inetOrgPerson") == nil {
		t.Error("schema 中找不到 inetOrgPerson")
	}
	if sm.AttributeType("entryCSN") == nil || !sm.AttributeType("entryCSN").SingleValue {
		t.Error("entryCSN 应存在且为单值")
	}
}

// ---------- M1：创建 / 删除保护 / 级联 / 预览 / 计数 / 关键字过滤 ----------

// TestCreateAndCounts 覆盖 R2.3 新建 OU 与 R2.1 子节点计数。
func TestCreateAndCounts(t *testing.T) {
	s, _ := newService(t)

	if err := s.Create(CreateRequest{
		DN:      "ou=newdept," + root,
		Classes: []string{"top", "organizationalUnit"},
		Attrs:   map[string][]string{"description": {"新建的部门"}},
	}); err != nil {
		t.Fatal(err)
	}
	d, err := s.Get("ou=newdept," + root)
	if err != nil {
		t.Fatalf("新建后读取失败: %v", err)
	}
	if d.Attrs["description"][0] != "新建的部门" {
		t.Errorf("description = %v", d.Attrs["description"])
	}

	kids, err := s.Children(root)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, n := range kids {
		if n.Name == "newdept" {
			found = true
			if n.ChildCount != 0 || n.HasChildren {
				t.Errorf("newdept childCount = %d，应为 0", n.ChildCount)
			}
		}
		if n.Name == "tech" && n.ChildCount < 4 { // 4 个人员 + 可能的子 OU
			t.Errorf("tech childCount = %d，应 ≥4", n.ChildCount)
		}
	}
	if !found {
		t.Fatal("树中未见 newdept")
	}

	// RDN 属性缺失时自动补齐（ou 值与 DN 一致性校验）
	if err := s.Create(CreateRequest{
		DN:      "ou=noattr," + root,
		Classes: []string{"organizationalUnit"},
	}); err != nil {
		t.Fatal(err)
	}
	if d, _ := s.Get("ou=noattr," + root); d == nil || d.Attrs["ou"][0] != "noattr" {
		t.Error("Create 应自动补 RDN 属性值")
	}

	// 非法请求
	if err := s.Create(CreateRequest{Classes: []string{"organizationalUnit"}}); err == nil {
		t.Error("缺 dn 应报错")
	}
	if err := s.Create(CreateRequest{DN: "ou=x," + root}); err == nil {
		t.Error("缺 objectClass 应报错")
	}
}

// TestDeleteProtection 覆盖 R2.5：非空删除被拦、预览清单、级联删除自底向上。
func TestDeleteProtection(t *testing.T) {
	s, _ := newService(t)
	branch := "ou=tech," + root // 含 4 人

	// 空分支：ou=market 有 1 人 hanmei → 非空
	if err := s.Delete("ou=market,"+root, false); !errors.Is(err, ErrNotEmpty) {
		t.Fatalf("非空删除应返回 ErrNotEmpty，得到: %v", err)
	}

	pv, err := s.DeletePreview("ou=market," + root)
	if err != nil {
		t.Fatal(err)
	}
	if pv.Empty || pv.Subtree != 1 || pv.Children != 1 {
		t.Errorf("market 预览 = %+v，应 subtree=1 children=1", pv)
	}
	if len(pv.Sample) != 1 || pv.Sample[0] != "uid=hanmei,ou=market,"+root {
		t.Errorf("market 预览样例 = %v", pv.Sample)
	}

	// tech 分支：4 人
	pv2, _ := s.DeletePreview(branch)
	if pv2.Subtree != 4 {
		t.Errorf("tech 预览 subtree = %d，应为 4（样例 %v）", pv2.Subtree, pv2.Sample)
	}

	// 级联删除 → 全部消失
	if err := s.Delete(branch, true); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Get("uid=zhangwei,ou=tech," + root); !errors.Is(err, ErrNotFound) {
		t.Errorf("级联后子条目仍存在: %v", err)
	}
	if _, err := s.Get(branch); !errors.Is(err, ErrNotFound) {
		t.Errorf("级联后分支本身仍存在: %v", err)
	}

	// 叶子条目直接删
	if err := s.Create(CreateRequest{DN: "ou=leaf," + root, Classes: []string{"organizationalUnit"}}); err != nil {
		t.Fatal(err)
	}
	if err := s.Delete("ou=leaf,"+root, false); err != nil {
		t.Fatalf("叶子删除不应报错: %v", err)
	}
}

// TestMovePreview 覆盖 R2.4：移动前预览新 DN 与影响面，父不存在报 ErrNotFound。
func TestMovePreview(t *testing.T) {
	s, _ := newService(t)
	dn := "uid=lina,ou=tech," + root

	pv, err := s.MovePreview(MoveRequest{DN: dn, NewParent: "ou=product," + root})
	if err != nil {
		t.Fatal(err)
	}
	if pv.NewDN != "uid=lina,ou=product,"+root || pv.Rename || pv.Moved != 0 {
		t.Errorf("预览 = %+v", pv)
	}

	// 移动 OU：影响面应统计子孙数（tech 4 人）
	pv2, err := s.MovePreview(MoveRequest{DN: "ou=tech," + root, NewParent: "ou=product," + root})
	if err != nil {
		t.Fatal(err)
	}
	if pv2.NewDN != "ou=tech,ou=product,"+root || pv2.Moved != 4 {
		t.Errorf("OU 移动预览 = %+v，Moved 应为 4", pv2)
	}

	// 改名 + 移动组合
	pv3, err := s.MovePreview(MoveRequest{DN: dn, NewRDN: "uid=lina2", NewParent: "ou=market," + root})
	if err != nil {
		t.Fatal(err)
	}
	if !pv3.Rename || pv3.NewDN != "uid=lina2,ou=market,"+root {
		t.Errorf("改名预览 = %+v", pv3)
	}

	if _, err := s.MovePreview(MoveRequest{DN: dn, NewParent: "ou=ghost," + root}); !errors.Is(err, ErrNotFound) {
		t.Errorf("父不存在应返回 ErrNotFound，得到: %v", err)
	}
}

// TestKeywordFilter 覆盖 R2.2 关键字 → 过滤器构造（含转义防注入）。
func TestKeywordFilter(t *testing.T) {
	if f := KeywordFilter(""); f != "(objectClass=*)" {
		t.Errorf("空关键字 = %q", f)
	}
	f := KeywordFilter("zhang")
	for _, attr := range []string{"cn", "uid", "sn", "mail", "employeeNumber", "ou", "description"} {
		if !strings.Contains(f, attr+"=*zhang*") {
			t.Errorf("过滤器缺 %s 子句: %q", attr, f)
		}
	}
	// 括号被转义为 \28 \29，防止过滤表达式注入（go-ldap 的十六进制转义形式）
	esc := KeywordFilter("a)b(c")
	if strings.Contains(esc, "a)b") || strings.Contains(esc, "b(c") {
		t.Errorf("括号未转义: %q", esc)
	}
	if !strings.Contains(esc, `a\29b\28c`) {
		t.Errorf("括号应转义为 \\28/\\29: %q", esc)
	}
	// 非 ASCII 同样安全转义（十六进制形式），不影响语义
	cn := KeywordFilter("张伟")
	if !strings.Contains(cn, "cn=*") || !strings.Contains(cn, "uid=*") || !strings.Contains(cn, ")(uid=") {
		t.Errorf("中文关键字过滤器结构异常: %q", cn)
	}
}

func TestParseLDIFEntry(t *testing.T) {
	ldif := `# 一条完整的条目
dn: ou=数据中心,dc=example,dc=cn
objectClass: top
objectClass: organizationalUnit
ou: 数据中心
description: 很长的描述
 续行内容
employeeNumber: E1
employeeNumber: E2`
	req, err := ParseLDIFEntry(ldif)
	if err != nil {
		t.Fatal(err)
	}
	if req.DN != "ou=数据中心,dc=example,dc=cn" {
		t.Errorf("dn = %q", req.DN)
	}
	if len(req.Classes) != 2 {
		t.Errorf("objectClass = %v", req.Classes)
	}
	if req.Attrs["description"][0] != "很长的描述续行内容" {
		t.Errorf("续行未拼接: %q", req.Attrs["description"][0])
	}
	if len(req.Attrs["employeeNumber"]) != 2 {
		t.Errorf("多值属性 = %v", req.Attrs["employeeNumber"])
	}

	for _, bad := range []string{
		"cn=x",                       // 无 dn
		"dn: a=b\nchangetype: delete", // 非新建
		"dn: a=b\nmail:: YmFzZTY0",   // base64
		"续行在开头",
	} {
		if _, err := ParseLDIFEntry(bad); err == nil {
			t.Errorf("恶意输入 %q 应报错", bad)
		}
	}
}
