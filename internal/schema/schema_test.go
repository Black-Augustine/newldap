package schema

import (
	"os"
	"strings"
	"testing"
)

// loadFixture 读取真实风格的 OpenLDAP subschema 输出。
// 文件格式与 slapd 返回一致：每条 "objectClasses: ( … )"，长条目折行续排。
func loadFixture(t *testing.T) map[string][]string {
	t.Helper()
	b, err := os.ReadFile("testdata/subschema.txt")
	if err != nil {
		t.Fatal(err)
	}
	attrs := map[string][]string{}
	var curKey string
	var cur strings.Builder
	flush := func() {
		if curKey != "" {
			attrs[curKey] = append(attrs[curKey], strings.TrimSpace(cur.String()))
			cur.Reset()
		}
	}
	for _, line := range strings.Split(string(b), "\n") {
		line = strings.TrimRight(line, "\r")
		if line == "" {
			continue
		}
		if i := strings.Index(line, ": "); i > 0 && !strings.HasPrefix(line, " ") {
			flush()
			curKey = line[:i]
			cur.WriteString(line[i+2:])
		} else {
			cur.WriteString(" " + strings.TrimSpace(line))
		}
	}
	flush()
	return attrs
}

func TestParseFixture(t *testing.T) {
	s, err := Parse(loadFixture(t))
	if err != nil {
		t.Fatal(err)
	}
	if len(s.ObjectClasses) != 10 {
		t.Errorf("objectClass 数量 = %d，应为 10", len(s.ObjectClasses))
	}
	if len(s.AttributeTypes) != 14 {
		t.Errorf("attributeType 数量 = %d，应为 14", len(s.AttributeTypes))
	}
}

func TestObjectClassDetails(t *testing.T) {
	s, err := Parse(loadFixture(t))
	if err != nil {
		t.Fatal(err)
	}
	inet := s.ObjectClass("inetOrgPerson")
	if inet == nil {
		t.Fatal("未找到 inetOrgPerson")
	}
	if inet.Kind != "STRUCTURAL" {
		t.Errorf("inetOrgPerson Kind = %q", inet.Kind)
	}
	if len(inet.Sup) != 1 || inet.Sup[0] != "organizationalPerson" {
		t.Errorf("inetOrgPerson SUP = %v", inet.Sup)
	}
	assertIn := func(list []string, want string) {
		t.Helper()
		for _, v := range list {
			if strings.EqualFold(v, want) {
				return
			}
		}
		t.Errorf("inetOrgPerson MAY 缺少 %s: %v", want, list)
	}
	assertIn(inet.May, "mail")
	assertIn(inet.May, "employeeNumber")

	posix := s.ObjectClass("posixAccount")
	if posix == nil || posix.Kind != "AUXILIARY" {
		t.Fatalf("posixAccount 应为 AUXILIARY: %+v", posix)
	}
	// MUST ( cn $ uid $ uidNumber $ gidNumber $ homeDirectory ) 是 $ 分隔的多值括号列表
	if len(posix.Must) != 5 {
		t.Errorf("posixAccount MUST = %v，应为 5 项", posix.Must)
	}

	gon := s.ObjectClass("groupOfNames")
	if gon == nil || len(gon.Must) != 2 {
		t.Errorf("groupOfNames MUST = %v", gon.Must)
	}

	// 无 MUST/MAY 的极简类
	no := s.ObjectClass("namedObject")
	if no == nil || len(no.Must) != 0 || len(no.May) != 1 {
		t.Errorf("namedObject = %+v", no)
	}

	// X- 扩展捕获
	ext := s.ObjectClass("msDS-Authz")
	if ext == nil || len(ext.Ext["X-ORDERED"]) != 1 {
		t.Errorf("msDS-Authz X-ORDERED = %v", ext.Ext)
	}
}

func TestAttributeTypeDetails(t *testing.T) {
	s, err := Parse(loadFixture(t))
	if err != nil {
		t.Fatal(err)
	}
	uid := s.AttributeType("uid")
	if uid == nil {
		t.Fatal("未找到 uid")
	}
	if len(uid.Names) != 2 || uid.Names[0] != "uid" || uid.Names[1] != "userId" {
		t.Errorf("uid 别名 = %v", uid.Names)
	}
	if uid.SyntaxOID() != "1.3.6.1.4.1.1466.115.121.1.15" || uid.Syntax != "1.3.6.1.4.1.1466.115.121.1.15{256}" {
		t.Errorf("uid 语法 = %q / %q", uid.Syntax, uid.SyntaxOID())
	}
	if uid.SingleValue {
		t.Error("uid 不应是单值")
	}

	csn := s.AttributeType("entryCSN")
	if csn == nil || !csn.SingleValue || !csn.NoUserModification {
		t.Errorf("entryCSN 标志错误: %+v", csn)
	}
	if csn.Usage != "directoryOperation" {
		t.Errorf("entryCSN USAGE = %q", csn.Usage)
	}

	// cn 继承自 name（SUP），且无自身语法
	cn := s.AttributeType("commonName") // 通过别名也应可查到
	if cn == nil || len(cn.Sup) != 1 || cn.Sup[0] != "name" {
		t.Errorf("cn = %+v", cn)
	}
	name := s.AttributeType("name")
	if name == nil || name.Equality != "caseIgnoreMatch" {
		t.Errorf("name EQUALITY = %q", name.Equality)
	}

	// 重复定义时后者覆盖索引（OpenLDAP 自定义 schema 覆盖内置的常见情况）
	if sn := s.AttributeType("sn"); sn == nil || sn.Desc != "重复 NAME 的废弃测试条目" {
		t.Errorf("同名覆盖未生效: %+v", sn)
	}
}

func TestParseMalformed(t *testing.T) {
	if _, err := ParseObjectClass("inetOrgPerson（无括号）"); err == nil {
		t.Error("无括号开头应报错")
	}
	if _, err := ParseObjectClass("( 1.2.3 NAME 'x' SUP top STRUCTURAL"); err == nil {
		t.Error("未闭合应报错")
	}
}
