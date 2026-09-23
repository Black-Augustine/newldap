package ldapclient

import (
	"strings"
	"testing"

	ldap "github.com/go-ldap/ldap/v3"

	"newldap/internal/ldaptest"
)

const root = "dc=example,dc=cn"

// TestWireProtocolAgainstDouble 是 spike① 的验收测试：
// 真实 go-ldap 客户端 × 进程内替身服务，验证 BER 线协议兼容性——
// 连接、bind（成功/失败）、rootDSE、分页搜索、条目增改删、modrdn。
func TestWireProtocolAgainstDouble(t *testing.T) {
	srv, err := ldaptest.Start(true)
	if err != nil {
		t.Fatal(err)
	}
	defer srv.Stop()

	c, err := Dial(Options{URL: srv.Addr()})
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()

	// bind 成功（admin）
	if err := c.Bind("cn=admin,"+root, "admin123"); err != nil {
		t.Fatalf("admin bind: %v", err)
	}
	// bind 失败必须报 invalidCredentials
	bad := func() {
		c2, err := Dial(Options{URL: srv.Addr()})
		if err != nil {
			t.Fatal(err)
		}
		defer c2.Close()
		err = c2.Bind("uid=zhangwei,ou=tech,"+root, "错误密码")
		if err == nil || !strings.Contains(err.Error(), "49") && !strings.Contains(err.Error(), "invalid") && !strings.Contains(err.Error(), "账号或密码") {
			t.Fatalf("期望 invalidCredentials，得到: %v", err)
		}
	}
	bad()

	// rootDSE
	dse, err := c.RootDSE()
	if err != nil {
		t.Fatal(err)
	}
	if len(dse.NamingContexts) == 0 || dse.NamingContexts[0] != root {
		t.Errorf("namingContexts = %v", dse.NamingContexts)
	}

	// 分页搜索：页大小 2，共 7 条（root+admin+5ou+6人+1组 = 14 条，subtree）
	entries, err := c.PagedSearch(root, ldap.ScopeWholeSubtree, "(objectClass=*)", nil, 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 14 {
		t.Errorf("分页搜索结果 = %d 条，应为 14（页大小 2 会走多页循环）", len(entries))
	}

	// 过滤搜索：(uid=zhangwei)
	found, err := c.PagedSearch(root, ldap.ScopeWholeSubtree, "(uid=zhangwei)", []string{"cn", "mail", "entryCSN"}, 100)
	if err != nil || len(found) != 1 {
		t.Fatalf("uid 搜索: %v, %d 条", err, len(found))
	}
	if found[0].GetAttributeValue("cn") != "张伟" {
		t.Errorf("cn = %q", found[0].GetAttributeValue("cn"))
	}

	// 新建条目
	err = c.AddEntry("ou=数据智能部,"+root, map[string][]string{
		"objectClass": {"top", "organizationalUnit"},
		"ou":          {"数据智能部"},
		"description": {"M0 spike 新建"},
	})
	if err != nil {
		t.Fatal(err)
	}

	// 属性级修改 + entryCSN 变化
	dn := "uid=xuqian,ou=数据智能部," + root
	if err := c.AddEntry(dn, map[string][]string{
		"objectClass":  {"top", "person", "organizationalPerson", "inetOrgPerson"},
		"uid":          {"xuqian"},
		"cn":           {"许倩"},
		"sn":           {"倩"},
		"userPassword": {"Init1234!"},
	}); err != nil {
		t.Fatal(err)
	}
	csn1, err := c.CurrentCSN(dn)
	if err != nil {
		t.Fatal(err)
	}
	if csn1 == "" {
		t.Fatal("替身未返回 entryCSN")
	}
	if err := c.ModifyAttributes(dn, []ldap.Change{
		{Operation: ldap.ReplaceAttribute, Modification: ldap.PartialAttribute{Type: "title", Vals: []string{"算法工程师"}}},
	}); err != nil {
		t.Fatal(err)
	}
	csn2, _ := c.CurrentCSN(dn)
	if csn1 == csn2 {
		t.Errorf("修改后 entryCSN 未变化: %s", csn2)
	}
	e, _ := c.ReadEntry(dn, nil)
	if e.GetAttributeValue("title") != "算法工程师" {
		t.Errorf("title 修改未生效")
	}

	// modrdn：改名 uid=xuqian → uid=xu.qian
	if err := c.ModRDN(dn, "uid=xu.qian", ""); err != nil {
		t.Fatalf("modrdn 改名: %v", err)
	}
	if e := srv.Entry("uid=xu.qian,ou=数据智能部," + root); e == nil {
		t.Fatal("modrdn 改名后找不到新 DN")
	}
	// modrdn：移动到 ou=tech
	if err := c.ModRDN("uid=xu.qian,ou=数据智能部,"+root, "uid=xu.qian", "ou=tech,"+root); err != nil {
		t.Fatalf("modrdn 移动: %v", err)
	}
	if e := srv.Entry("uid=xu.qian,ou=tech," + root); e == nil {
		t.Fatal("modrdn 移动后找不到目标 DN")
	}

	// 删除（子树约束：先删人再删部门）
	if err := c.DeleteEntry("uid=xu.qian,ou=tech," + root); err != nil {
		t.Fatal(err)
	}
	if err := c.DeleteEntry("ou=数据智能部," + root); err != nil {
		t.Fatal(err)
	}

	// subschema 可读且能喂给解析器（集成 schema 包由 directory 侧覆盖）
	sub, err := c.Subschema()
	if err != nil {
		t.Fatal(err)
	}
	if len(sub["objectClasses"]) == 0 || len(sub["attributeTypes"]) == 0 {
		t.Errorf("subschema 内容为空: %+v", sub)
	}
}
