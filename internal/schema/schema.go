// Package schema 解析 RFC 4512 语法的 subschema（cn=Subschema 返回的
// objectClasses / attributeTypes 描述串），形成内存模型，供动态表单与
// 专家模式使用（《技术架构设计》D2）。
package schema

import (
	"fmt"
	"strings"
)

// ObjectClass 对应一条 objectClasses 描述。
type ObjectClass struct {
	OID   string
	Names []string
	Desc  string
	Sup   []string
	Kind  string // STRUCTURAL / ABSTRACT / AUXILIARY（空值视为 STRUCTURAL）
	Must  []string
	May   []string
	Ext   map[string][]string // X- 扩展
}

// AttributeType 对应一条 attributeTypes 描述。
type AttributeType struct {
	OID                string
	Names              []string
	Desc               string
	Sup                []string
	Equality           string
	Ordering           string
	Substr             string
	Syntax             string // 原样保留，如 1.3.6.1.4.1.1466.115.121.1.15{64}
	SingleValue        bool
	Collective         bool
	NoUserModification bool
	Usage              string
	Ext                map[string][]string
}

// SyntaxOID 返回去掉 {len} 限长的语法 OID。
func (a *AttributeType) SyntaxOID() string {
	if i := strings.IndexByte(a.Syntax, '{'); i >= 0 {
		return a.Syntax[:i]
	}
	return a.Syntax
}

// Schema 是一次 subschema 解析结果，内含按名称索引。
type Schema struct {
	ObjectClasses  []*ObjectClass
	AttributeTypes []*AttributeType
	ocByName       map[string]*ObjectClass
	atByName       map[string]*AttributeType
}

func (s *Schema) ObjectClass(name string) *ObjectClass     { return s.ocByName[strings.ToLower(name)] }
func (s *Schema) AttributeType(name string) *AttributeType { return s.atByName[strings.ToLower(name)] }

// ---------- 词法 ----------

type tokKind int

const (
	tokPunct tokKind = iota // ( ) $
	tokWord                 // 裸词：OID、关键字、宏名
	tokQStr                 // '单引号字符串'
)

type token struct {
	kind tokKind
	val  string
}

func tokenize(s string) []token {
	var toks []token
	i := 0
	for i < len(s) {
		c := s[i]
		switch {
		case c == ' ' || c == '\t' || c == '\n' || c == '\r':
			i++
		case c == '(' || c == ')' || c == '$':
			toks = append(toks, token{tokPunct, string(c)})
			i++
		case c == '\'':
			i++
			var b strings.Builder
			for i < len(s) {
				if s[i] == '\\' && i+1 < len(s) {
					b.WriteByte(s[i+1])
					i += 2
					continue
				}
				if s[i] == '\'' {
					i++
					break
				}
				b.WriteByte(s[i])
				i++
			}
			toks = append(toks, token{tokQStr, b.String()})
		default:
			j := i
			for j < len(s) && !strings.ContainsRune(" \t\n\r()'$", rune(s[j])) {
				j++
			}
			toks = append(toks, token{tokWord, s[i:j]})
			i = j
		}
	}
	return toks
}

// ---------- 语法 ----------

type parser struct {
	toks []token
	pos  int
}

func (p *parser) peek() token { return p.toks[p.pos] }
func (p *parser) next() token { t := p.toks[p.pos]; p.pos++; return t }
func (p *parser) done() bool  { return p.pos >= len(p.toks) }

// values 读取一个值或值列表：'q' / word / ( v1 $ v2 … )
func (p *parser) values() []string {
	t := p.peek()
	switch {
	case t.kind == tokQStr:
		p.next()
		return []string{t.val}
	case t.kind == tokWord:
		p.next()
		return []string{t.val}
	case t.kind == tokPunct && t.val == "(":
		p.next()
		var out []string
		for !p.done() {
			t = p.next()
			if t.kind == tokPunct && t.val == ")" {
				break
			}
			if t.kind == tokPunct && t.val == "$" {
				continue
			}
			out = append(out, t.val)
		}
		return out
	}
	return nil
}

// ParseObjectClass 解析一条 objectClasses 描述串（不含 "objectClasses:" 前缀）。
func ParseObjectClass(s string) (*ObjectClass, error) {
	p := &parser{toks: tokenize(s)}
	if p.done() || p.peek().val != "(" {
		return nil, fmt.Errorf("objectClass 应以 ( 开始: %.40s", s)
	}
	p.next()
	oc := &ObjectClass{Ext: map[string][]string{}}
	oc.OID = p.next().val
	for !p.done() {
		t := p.next()
		if t.kind == tokPunct && t.val == ")" {
			return oc, nil
		}
		if t.kind != tokWord {
			continue // 容忍多余标点
		}
		switch t.val {
		case "NAME":
			oc.Names = p.values()
		case "DESC":
			if vs := p.values(); len(vs) > 0 {
				oc.Desc = vs[0]
			}
		case "SUP":
			oc.Sup = p.values()
		case "OBSOLETE":
			// 无值关键字
		case "ABSTRACT", "STRUCTURAL", "AUXILIARY":
			oc.Kind = t.val
		case "MUST":
			oc.Must = p.values()
		case "MAY":
			oc.May = p.values()
		default:
			if strings.HasPrefix(t.val, "X-") || len(t.val) > 0 {
				oc.Ext[t.val] = p.values()
			}
		}
	}
	return nil, fmt.Errorf("objectClass 描述未闭合: %.40s", s)
}

// ParseAttributeType 解析一条 attributeTypes 描述串。
func ParseAttributeType(s string) (*AttributeType, error) {
	p := &parser{toks: tokenize(s)}
	if p.done() || p.peek().val != "(" {
		return nil, fmt.Errorf("attributeType 应以 ( 开始: %.40s", s)
	}
	p.next()
	at := &AttributeType{Ext: map[string][]string{}}
	at.OID = p.next().val
	for !p.done() {
		t := p.next()
		if t.kind == tokPunct && t.val == ")" {
			return at, nil
		}
		if t.kind != tokWord {
			continue
		}
		switch t.val {
		case "NAME":
			at.Names = p.values()
		case "DESC":
			if vs := p.values(); len(vs) > 0 {
				at.Desc = vs[0]
			}
		case "SUP":
			at.Sup = p.values()
		case "EQUALITY":
			at.Equality = p.values()[0]
		case "ORDERING":
			at.Ordering = p.values()[0]
		case "SUBSTR":
			at.Substr = p.values()[0]
		case "SYNTAX":
			at.Syntax = p.values()[0]
		case "SINGLE-VALUE":
			at.SingleValue = true
		case "COLLECTIVE":
			at.Collective = true
		case "NO-USER-MODIFICATION":
			at.NoUserModification = true
		case "USAGE":
			at.Usage = p.values()[0]
		case "OBSOLETE":
		default:
			at.Ext[t.val] = p.values()
		}
	}
	return nil, fmt.Errorf("attributeType 描述未闭合: %.40s", s)
}

// Parse 从 subschema 条目的属性集合（通常来自对 cn=Subschema 的 BaseObject
// 搜索，Attributes = ["objectClasses", "attributeTypes"]）构建 Schema。
func Parse(attrs map[string][]string) (*Schema, error) {
	s := &Schema{
		ocByName: map[string]*ObjectClass{},
		atByName: map[string]*AttributeType{},
	}
	for _, line := range attrs["objectClasses"] {
		oc, err := ParseObjectClass(line)
		if err != nil {
			return nil, err
		}
		s.ObjectClasses = append(s.ObjectClasses, oc)
		for _, n := range oc.Names {
			s.ocByName[strings.ToLower(n)] = oc
		}
	}
	for _, line := range attrs["attributeTypes"] {
		at, err := ParseAttributeType(line)
		if err != nil {
			return nil, err
		}
		s.AttributeTypes = append(s.AttributeTypes, at)
		for _, n := range at.Names {
			s.atByName[strings.ToLower(n)] = at
		}
	}
	return s, nil
}
