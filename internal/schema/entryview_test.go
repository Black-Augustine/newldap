package schema

import "testing"

func entryFixture(t *testing.T) *Schema {
	t.Helper()
	return formFixture(t)
}

func TestEntryViewPerson(t *testing.T) {
	s := entryFixture(t)
	attrs := map[string][]string{
		"objectClass":     {"top", "person", "organizationalPerson", "inetOrgPerson"},
		"cn":              {"张伟"},
		"sn":              {"伟"},
		"uid":             {"zhangwei"},
		"mail":            {"zhangwei@example.cn", "zw@example.cn"},
		"title":           {"平台研发工程师"},
		"entryCSN":        {"20260921..."},
		"createTimestamp": {"20260921000000Z"},
		"notInSchema":     {"随便的值"},
	}
	ev := s.EntryView(attrs)

	if ev.Structural != "inetOrgPerson" {
		t.Errorf("结构类 = %q", ev.Structural)
	}
	if len(ev.ClassesChain) != 3 {
		t.Errorf("继承链 = %v", ev.ClassesChain)
	}
	groupOf := func(n string) string {
		for _, a := range ev.Attrs {
			if a.Name == n {
				return a.Group
			}
		}
		return ""
	}
	for name, want := range map[string]string{
		"cn": "must", "sn": "must", "uid": "may", "mail": "may", "title": "may",
		"entryCSN": "operational", "createTimestamp": "operational", "notInSchema": "unknown",
	} {
		if g := groupOf(name); g != want {
			t.Errorf("%s 分组 = %q，应为 %q", name, g, want)
		}
	}
	// 双语标签存在且含属性名
	for _, a := range ev.Attrs {
		if a.Name == "cn" && a.Label != "姓名 / 名称" {
			t.Errorf("cn 标签 = %q", a.Label)
		}
	}
	// 操作属性只读
	for _, a := range ev.Attrs {
		if a.Group == "operational" && !a.ReadOnly {
			t.Errorf("%s 应为只读", a.Name)
		}
	}
	// 可添加属性：类链 MAY 未占用（如 mobile/employeeNumber），不含已占用 cn
	hasMobile, hasCN := false, false
	for _, b := range ev.AvailableAttrs {
		if b.Name == "mobile" {
			hasMobile = true
		}
		if b.Name == "cn" {
			hasCN = true
		}
	}
	if !hasMobile {
		t.Error("可添加属性缺 mobile")
	}
	if hasCN {
		t.Error("已占用的 cn 不应出现在可添加列表")
	}
	// 可添加类：含 posixGroup/posixAccount（辅助可加），不含 top
	hasClass, hasTop := false, false
	for _, c := range ev.AvailableClasses {
		if c.Name == "posixAccount" {
			hasClass = true
		}
		if c.Name == "top" {
			hasTop = true
		}
	}
	if !hasClass {
		t.Error("可添加类缺 posixAccount")
	}
	if hasTop {
		t.Error("top 不应可添加")
	}
}

func TestEntryViewOU(t *testing.T) {
	s := entryFixture(t)
	ev := s.EntryView(map[string][]string{
		"objectClass": {"top", "organizationalUnit"},
		"ou":          {"tech"},
		"description": {"研发"},
	})
	if ev.Structural != "organizationalUnit" {
		t.Errorf("结构类 = %q", ev.Structural)
	}
	groupOf := func(n string) string {
		for _, a := range ev.Attrs {
			if a.Name == n {
				return a.Group
			}
		}
		return ""
	}
	if groupOf("ou") != "must" {
		t.Errorf("ou 分组 = %q", groupOf("ou"))
	}
	if groupOf("description") != "may" {
		t.Errorf("description 分组 = %q", groupOf("description"))
	}
}

func TestEntryViewUnknownEntry(t *testing.T) {
	s := entryFixture(t)
	ev := s.EntryView(map[string][]string{
		"objectClass": {"top", "完全不认识"},
		"cn":          {"x"},
	})
	if ev.Structural != "完全不认识" {
		t.Errorf("未知结构类兜底 = %q", ev.Structural)
	}
	// cn 在未知类下无 MUST/MAY 依据 → unknown 组
	groupOf := func(n string) string {
		for _, a := range ev.Attrs {
			if a.Name == n {
				return a.Group
			}
		}
		return ""
	}
	if groupOf("cn") != "unknown" {
		t.Errorf("cn 分组 = %q（未知类下应为 unknown）", groupOf("cn"))
	}
}
