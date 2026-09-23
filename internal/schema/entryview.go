// EntryView：schema 感知的条目视图模型（贴合 LDAP 的展示层，M5+ 重设计）。
// 借鉴 phpLDAPadmin / LAM：属性按 必填/可选/操作/未识别 分组，标签中英双语，
// objectClass 区分结构/辅助类并可从服务端 schema 全集追加，
// "可添加属性"只列当前类链 MAY 且未占用的属性。
package schema

import "strings"

// AttrBrief 是可添加属性下拉里的一项。
type AttrBrief struct {
	Name   string `json:"name"`
	Label  string `json:"label"`
	Syntax string `json:"syntax,omitempty"`
}

// AttrView 是条目上一个属性的展示视图。
type AttrView struct {
	Name     string   `json:"name"`
	Label    string   `json:"label"` // 中文标签（未知属性回退属性名）
	Values   []string `json:"values"`
	Group    string   `json:"group"` // must | may | operational | unknown
	Syntax   string   `json:"syntax,omitempty"`
	Single   bool     `json:"single"`
	ReadOnly bool     `json:"readOnly"`
}

// ClassInfo 是 objectClass 展示信息。
type ClassInfo struct {
	Name string `json:"name"`
	Kind string `json:"kind"` // STRUCTURAL | AUXILIARY | ABSTRACT | unknown
}

// EntryView 是整个条目的展示模型。
type EntryView struct {
	Attrs            []AttrView  `json:"attrs"`
	Groups           []string    `json:"groupsOrder"`      // must/may/operational/unknown 展示顺序
	ObjectClasses    []ClassInfo `json:"objectClasses"`    // 当前条目的类
	Structural       string      `json:"structural"`       // 结构类（空=未识别）
	ClassesChain     []string    `json:"classesChain"`     // 结构类继承链
	AvailableClasses []ClassInfo `json:"availableClasses"` // 服务端 schema 中未使用的类（可添加）
	AvailableAttrs   []AttrBrief `json:"availableAttrs"`   // 当前类链 MAY 且未占用（可添加）
}

// knownOperational 常见操作属性（schema 未声明时兜底归组）。
var knownOperational = map[string]bool{
	"entrycsn": true, "createtimestamp": true, "modifytimestamp": true,
	"creatorsname": true, "modifiersname": true, "entryuuid": true,
	"entrydn": true, "subschemasubentry": true, "hassubordinates": true,
	"entrydiff": true, "structuralobjectclass": true,
}

// EntryView 依据解析好的 schema 为条目属性表生成视图模型。
func (s *Schema) EntryView(attrs map[string][]string) *EntryView {
	ev := &EntryView{
		Groups: []string{"must", "may", "operational", "unknown"},
	}

	// 当前 objectClass 与结构类
	var classes []string
	structural := ""
	for _, v := range attrs["objectClass"] {
		classes = append(classes, v)
	}
	structuralUnknown := ""
	for _, c := range classes {
		if oc := s.ocByName[strings.ToLower(c)]; oc != nil {
			kind := oc.Kind
			if kind == "" {
				kind = "STRUCTURAL"
			}
			ev.ObjectClasses = append(ev.ObjectClasses, ClassInfo{Name: c, Kind: kind})
			if kind == "STRUCTURAL" && !strings.EqualFold(c, "top") {
				structural = c // 取最后一个已知结构类（列表惯例：top → 最派生类）
			}
		} else {
			ev.ObjectClasses = append(ev.ObjectClasses, ClassInfo{Name: c, Kind: "unknown"})
			if !strings.EqualFold(c, "top") {
				structuralUnknown = c
			}
		}
	}
	if structural == "" {
		structural = structuralUnknown // 兜底：最后一个非 top 未知类
	}
	ev.Structural = structural
	if structural != "" {
		_, _, chain := s.EffectiveAttrs(structural)
		ev.ClassesChain = chain
	}

	// 有效 MUST/MAY（结构类链 + 在场的辅助类）
	inMust, inMay := map[string]bool{}, map[string]bool{}
	if structural != "" {
		m, y, _ := s.EffectiveAttrs(structural)
		for _, a := range m {
			inMust[strings.ToLower(a)] = true
		}
		for _, a := range y {
			inMay[strings.ToLower(a)] = true
		}
	}
	for _, c := range classes {
		oc := s.ocByName[strings.ToLower(c)]
		if oc == nil || (oc.Kind != "" && oc.Kind != "AUXILIARY") {
			continue
		}
		m, y, _ := s.EffectiveAttrs(oc.Names[0])
		for _, a := range m {
			inMust[strings.ToLower(a)] = true
		}
		for _, a := range y {
			inMay[strings.ToLower(a)] = true
		}
	}

	// 每个在场属性 → 视图
	for name, values := range attrs {
		l := strings.ToLower(name)
		if l == "objectclass" {
			continue
		}
		v := AttrView{Name: name, Label: AttrLabel(name), Values: values, Single: true}
		if at := s.atByName[l]; at != nil {
			v.Syntax = at.SyntaxOID()
			v.Single = at.SingleValue
			if at.NoUserModification || (at.Usage != "" && at.Usage != "userApplications") || knownOperational[l] {
				v.Group = "operational"
				v.ReadOnly = true
			}
		} else if knownOperational[l] {
			v.Group = "operational"
			v.ReadOnly = true
		}
		if v.Group == "" {
			switch {
			case inMust[l]:
				v.Group = "must"
			case inMay[l]:
				v.Group = "may"
			default:
				v.Group = "unknown"
			}
		}
		ev.Attrs = append(ev.Attrs, v)
	}
	sortAttrs(ev.Attrs)

	// 可添加属性：当前类链的 MUST+MAY 中未在场的（must 未占用的也列出，便于补值）
	present := map[string]bool{}
	for name := range attrs {
		present[strings.ToLower(name)] = true
	}
	seen := map[string]bool{}
	for _, a := range append(keysOf(inMust), keysOf(inMay)...) {
		l := strings.ToLower(a)
		if present[l] || seen[l] {
			continue
		}
		seen[l] = true
		b := AttrBrief{Name: l, Label: AttrLabel(a)}
		if at := s.atByName[l]; at != nil {
			b.Syntax = at.SyntaxOID()
		}
		ev.AvailableAttrs = append(ev.AvailableAttrs, b)
	}
	sortBriefs(ev.AvailableAttrs)

	// 可添加 objectClass：schema 全集 - 在场（排除 top/abstract）
	for _, oc := range s.ObjectClasses {
		name := oc.Names[0]
		onStage := false
		for _, c := range classes {
			if strings.EqualFold(c, name) {
				onStage = true
			}
		}
		if onStage || strings.EqualFold(name, "top") {
			continue
		}
		kind := oc.Kind
		if kind == "" {
			kind = "STRUCTURAL"
		}
		if kind == "ABSTRACT" {
			continue
		}
		ev.AvailableClasses = append(ev.AvailableClasses, ClassInfo{Name: name, Kind: kind})
	}
	return ev
}

func sortAttrs(a []AttrView) {
	for i := 1; i < len(a); i++ {
		for j := i; j > 0; j-- {
			x, y := a[j-1], a[j]
			if x.Group > y.Group || (x.Group == y.Group && x.Name > y.Name) {
				a[j-1], a[j] = y, x
			} else {
				break
			}
		}
	}
}

func sortBriefs(b []AttrBrief) {
	for i := 1; i < len(b); i++ {
		for j := i; j > 0; j-- {
			if b[j-1].Name > b[j].Name {
				b[j-1], b[j] = b[j], b[j-1]
			} else {
				break
			}
		}
	}
}

func keysOf(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
