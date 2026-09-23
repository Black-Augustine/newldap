package ldapserver

import (
	"encoding/hex"
	"testing"

	ldap "newldap/internal/goldapmessage"
)

func wireEntry(t *testing.T, build func(*ldap.SearchResultEntry)) (string, int) {
	t.Helper()
	e := NewSearchResultEntry("ou=demo,dc=example,dc=cn")
	build(&e)
	m := ldap.NewLDAPMessageWithProtocolOp(e)
	m.SetMessageID(3)
	b, err := m.Write()
	if err != nil {
		return "", -1
	}
	bs := b.Bytes()
	return hex.EncodeToString(bs), len(bs)
}

func TestSearchEntryWireSizes(t *testing.T) {
	h1, n1 := wireEntry(t, func(e *ldap.SearchResultEntry) {
		e.AddAttribute(ldap.AttributeDescription("objectClass"),
			ldap.AttributeValue("top"), ldap.AttributeValue("organizationalUnit"))
		e.AddAttribute(ldap.AttributeDescription("ou"), ldap.AttributeValue("demo"))
	})
	t.Logf("双值: n=%d %s", n1, h1)
	if n1 < 0 {
		t.Fatal("双值条目 Write 失败（size/write 不一致）")
	}

	h2, n2 := wireEntry(t, func(e *ldap.SearchResultEntry) {
		e.AddAttribute(ldap.AttributeDescription("objectClass"),
			ldap.AttributeValue("top"), ldap.AttributeValue("person"),
			ldap.AttributeValue("organizationalPerson"), ldap.AttributeValue("inetOrgPerson"))
		e.AddAttribute(ldap.AttributeDescription("cn"), ldap.AttributeValue("张伟"))
		e.AddAttribute(ldap.AttributeDescription("mail"), ldap.AttributeValue("zhangwei@example.cn"))
	})
	t.Logf("人员: n=%d %s", n2, h2)
	if n2 < 0 {
		t.Fatal("人员条目 Write 失败")
	}

	h3, n3 := wireEntry(t, func(e *ldap.SearchResultEntry) {})
	t.Logf("空属性: n=%d %s", n3, h3)
	if n3 < 0 {
		t.Fatal("空属性条目 Write 失败")
	}
}

func TestAttributeBisect(t *testing.T) {
	mk := func(vals ...string) (string, int) {
		return wireEntry(t, func(e *ldap.SearchResultEntry) {
			for _, v := range vals {
				e.AddAttribute(ldap.AttributeDescription("cn"), ldap.AttributeValue(v))
			}
		})
	}
	for _, c := range []struct {
		name string
		vals []string
	}{
		{"1 英文", []string{"abc"}},
		{"2 英文", []string{"abc", "def"}},
		{"3 英文", []string{"abc", "def", "ghi"}},
		{"4 英文", []string{"abc", "def", "ghi", "jkl"}},
		{"1 中文", []string{"张伟"}},
		{"2 中文", []string{"张伟", "李娜"}},
	} {
		h, n := mk(c.vals...)
		if n < 0 {
			t.Errorf("[%s] Write 失败（size/write 不一致）", c.name)
		} else {
			t.Logf("[%s] OK n=%d %s", c.name, n, h)
		}
	}
}

func TestComboBisect(t *testing.T) {
	oc4 := func(e *ldap.SearchResultEntry) {
		e.AddAttribute(ldap.AttributeDescription("objectClass"),
			ldap.AttributeValue("top"), ldap.AttributeValue("person"),
			ldap.AttributeValue("organizationalPerson"), ldap.AttributeValue("inetOrgPerson"))
	}
	cases := []struct {
		name string
		build func(*ldap.SearchResultEntry)
	}{
		{"仅 4 值 objectClass", oc4},
		{"oc4+cn 中文", func(e *ldap.SearchResultEntry) { oc4(e); e.AddAttribute(ldap.AttributeDescription("cn"), ldap.AttributeValue("张伟")) }},
		{"oc4+mail 19 字符", func(e *ldap.SearchResultEntry) { oc4(e); e.AddAttribute(ldap.AttributeDescription("mail"), ldap.AttributeValue("zhangwei@example.cn")) }},
		{"cn+mail 两属性", func(e *ldap.SearchResultEntry) {
			e.AddAttribute(ldap.AttributeDescription("cn"), ldap.AttributeValue("张伟"))
			e.AddAttribute(ldap.AttributeDescription("mail"), ldap.AttributeValue("zhangwei@example.cn"))
		}},
		{"oc4+cn+mail（原失败组合）", func(e *ldap.SearchResultEntry) {
			oc4(e)
			e.AddAttribute(ldap.AttributeDescription("cn"), ldap.AttributeValue("张伟"))
			e.AddAttribute(ldap.AttributeDescription("mail"), ldap.AttributeValue("zhangwei@example.cn"))
		}},
	}
	for _, c := range cases {
		h, n := wireEntry(t, c.build)
		if n < 0 {
			t.Errorf("[%s] Write 失败", c.name)
		} else {
			t.Logf("[%s] OK n=%d %s", c.name, n, h)
		}
	}
}
