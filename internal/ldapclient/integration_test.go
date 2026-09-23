// 真实 OpenLDAP 集成测试（M0 spike① 的服务端验证）。
//
// 本地无 OpenLDAP/Docker 时自动跳过；CI 中由 service 容器提供：
//
//	LDAP_IT_URL=ldap://localhost:1389
//	LDAP_IT_BIND_DN=cn=admin,dc=example,dc=cn
//	LDAP_IT_PASSWORD=adminpass
//	LDAP_IT_BASE_DN=dc=example,dc=cn
//
// 运行：go test ./internal/ldapclient/ -run Integration -v
package ldapclient_test

import (
	"fmt"
	"os"
	"strings"
	"testing"

	ldap "github.com/go-ldap/ldap/v3"

	"newldap/internal/ldapclient"
)

func integrationEnv() (url, dn, pw, base string, ok bool) {
	url = os.Getenv("LDAP_IT_URL")
	dn = os.Getenv("LDAP_IT_BIND_DN")
	pw = os.Getenv("LDAP_IT_PASSWORD")
	base = os.Getenv("LDAP_IT_BASE_DN")
	return url, dn, pw, base, url != ""
}

func TestIntegrationLiveOpenLDAP(t *testing.T) {
	url, bindDN, pw, base, ok := integrationEnv()
	if !ok {
		t.Skip("未设置 LDAP_IT_URL，跳过真实 OpenLDAP 集成测试（CI 中自动运行）")
	}
	c, err := ldapclient.Dial(ldapclient.Options{URL: url})
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	if err := c.Bind(bindDN, pw); err != nil {
		t.Fatalf("管理员 bind: %v", err)
	}

	// rootDSE 探测
	dse, err := c.RootDSE()
	if err != nil {
		t.Fatal(err)
	}
	if len(dse.NamingContexts) == 0 {
		t.Fatal("真实服务应返回 namingContexts")
	}
	t.Logf("服务端: namingContexts=%v subschema=%s", dse.NamingContexts, dse.SubschemasSubentry)

	// 在 base 下建临时子树
	suffix := fmt.Sprintf("it-%d", os.Getpid())
	ou := "ou=" + suffix + "," + base
	person := "uid=ituser," + ou
	t.Cleanup(func() {
		_ = c.DeleteEntry(person)
		_ = c.DeleteEntry(ou)
	})
	if err := c.AddEntry(ou, map[string][]string{
		"objectClass": {"top", "organizationalUnit"}, "ou": {suffix},
	}); err != nil {
		t.Fatalf("真实服务新建 OU: %v", err)
	}
	if err := c.AddEntry(person, map[string][]string{
		"objectClass":  {"top", "person", "organizationalPerson", "inetOrgPerson"},
		"uid":          {"ituser"},
		"cn":           {"集成测试用户"},
		"sn":           {"用户"},
		"userPassword": {"It12345!"},
	}); err != nil {
		t.Fatalf("真实服务新建人员: %v", err)
	}

	// 真实 entryCSN 冲突检测路径
	csn1, err := c.CurrentCSN(person)
	if err != nil {
		t.Fatalf("读取真实 entryCSN: %v", err)
	}
	if csn1 == "" {
		t.Fatal("真实服务未返回 entryCSN（需要 OpenLDAP）")
	}
	if err := c.ModifyAttributes(person, []ldap.Change{
		{Operation: ldap.ReplaceAttribute, Modification: ldap.PartialAttribute{Type: "title", Vals: []string{"第一次修改"}}},
	}); err != nil {
		t.Fatal(err)
	}
	csn2, _ := c.CurrentCSN(person)
	if csn1 == csn2 {
		t.Errorf("真实服务修改后 entryCSN 未变化: %s", csn2)
	}

	// 分页：页大小 1 也要收全
	entries, err := c.PagedSearch(ou, ldap.ScopeWholeSubtree, "(objectClass=*)", nil, 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 2 {
		t.Errorf("分页(1/页)结果 = %d，应为 2", len(entries))
	}

	// 真实 modrdn
	if err := c.ModRDN(person, "uid=ituser2", ""); err != nil {
		t.Fatalf("真实服务 modrdn: %v", err)
	}
	person = "uid=ituser2," + ou
	e, err := c.ReadEntry(person, []string{"cn"})
	if err != nil || e.GetAttributeValue("cn") != "集成测试用户" {
		t.Fatalf("modrdn 后读取: %v", err)
	}

	// 真实 subschema 可解析
	sub, err := c.Subschema()
	if err != nil {
		t.Fatal(err)
	}
	if len(sub["objectClasses"]) < 50 || !strings.Contains(strings.Join(sub["objectClasses"], "\n"), "inetOrgPerson") {
		t.Errorf("真实 subschema 异常：objectClasses=%d 条", len(sub["objectClasses"]))
	}
}
