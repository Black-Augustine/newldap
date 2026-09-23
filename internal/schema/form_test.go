package schema

import (
	"strings"
	"testing"
)

func formFixture(t *testing.T) *Schema {
	t.Helper()
	s, err := Parse(loadFixture(t))
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func TestFormModelInetOrgPerson(t *testing.T) {
	s := formFixture(t)
	fm := s.FormModel("inetOrgPerson")
	if fm.StructuralClass != "inetOrgPerson" {
		t.Fatalf("主类 = %q", fm.StructuralClass)
	}
	// objectClass 链应含 inetOrgPerson 自身（父类未在 fixture 中定义则被忽略）
	if len(fm.Classes) == 0 || fm.Classes[0] != "inetOrgPerson" {
		t.Errorf("classes = %v", fm.Classes)
	}
	var cn, sn Field
	for _, f := range fm.Fields {
		if f.Name == "cn" {
			cn = f
		}
		if f.Name == "sn" {
			sn = f
		}
	}
	// cn 继承自 name（SUP）；fixture 中 organizationalPerson 未定义 → cn/sn 的 MUST
	// 来自 inetOrgPerson 自身 MAY/公共认知。此处断言：cn 必填（人员类 MUST cn 是
	// schema 公识，且 fixture inetOrgPerson 未列 MUST —— 该断言在真实 OpenLDAP
	// subschema 下成立；fixture 精简版下我们至少要求 cn 出现且标签正确）
	if cn.Name == "" {
		t.Fatal("表单缺少 cn 字段")
	}
	if !cn.Required {
		t.Error("cn 继承自 person 的 MUST，应为必填")
	}
	if cn.Label != "姓名 / 名称" {
		t.Errorf("cn 标签 = %q", cn.Label)
	}
	if sn.Name == "" || sn.Label != "姓氏" {
		t.Errorf("sn = %+v", sn)
	}
	if !sn.Required {
		t.Error("sn 继承自 person 的 MUST，应为必填")
	}
	// objectClass / userPassword 不得进表单
	for _, f := range fm.Fields {
		if f.Name == "objectclass" || f.Name == "userpassword" {
			t.Errorf("排除属性进入表单: %+v", f)
		}
		if f.ReadOnly {
			t.Errorf("普通属性被标记只读: %+v", f)
		}
	}
	// 邮箱控件（mail IA5String → email）
	var mail Field
	for _, f := range fm.Fields {
		if f.Name == "mail" {
			mail = f
		}
	}
	if mail.Name == "" || mail.Control != "email" {
		t.Errorf("mail 控件 = %+v", mail)
	}
	if !mail.Multi {
		t.Error("mail 应为多值")
	}
}

func TestFormModelPosixGroup(t *testing.T) {
	s := formFixture(t)
	fm := s.FormModel("posixGroup")
	var gid Field
	for _, f := range fm.Fields {
		if f.Name == "gidnumber" {
			gid = f
		}
	}
	if gid.Name == "" {
		t.Fatal("posixGroup 表单缺少 gidNumber")
	}
	if !gid.Required {
		t.Error("gidNumber 来自 MUST，应为必填")
	}
	if gid.Control != "number" {
		t.Errorf("gidNumber 控件 = %q", gid.Control)
	}
}

func TestFormModelOrganizationalUnit(t *testing.T) {
	s := formFixture(t)
	fm := s.FormModel("organizationalUnit")
	var ou Field
	for _, f := range fm.Fields {
		if f.Name == "ou" {
			ou = f
		}
	}
	if ou.Name == "" || !ou.Required {
		t.Fatalf("organizationalUnit 的 ou 应为必填: %+v", ou)
	}
	var desc Field
	for _, f := range fm.Fields {
		if f.Name == "description" {
			desc = f
		}
	}
	if desc.Control != "textarea" {
		t.Errorf("description 控件 = %q", desc.Control)
	}
}

func TestFormModelUnknownClass(t *testing.T) {
	s := formFixture(t)
	fm := s.FormModel("完全不存在的类")
	if fm == nil || len(fm.Fields) != 0 {
		t.Errorf("未知类应返回空字段模型: %+v", fm)
	}
}

func TestEffectiveAttrsInheritance(t *testing.T) {
	s := formFixture(t)
	// inetOrgPerson → organizationalPerson → person → top 全链解析
	must, may, classes := s.EffectiveAttrs("inetOrgPerson")
	if !containsFoldStr(must, "cn") || !containsFoldStr(must, "sn") {
		t.Errorf("继承后 MUST 应含 cn/sn: %v", must)
	}
	if len(classes) != 3 { // inetOrgPerson + organizationalPerson + person（top 未定义被忽略）
		t.Errorf("classes = %v，应为 3 层", classes)
	}
	found := false
	for _, a := range may {
		if a == "mail" || a == "uid" {
			found = true
		}
	}
	if !found {
		t.Errorf("MAY 缺少 mail/uid: %v", may)
	}
}

func containsFoldStr(list []string, v string) bool {
	for _, x := range list {
		if strings.EqualFold(x, v) {
			return true
		}
	}
	return false
}
