// Package schema：表单模型生成（R5.2 动态表单 / R5.4 自定义 schema 兼容）。
// 依据解析出的 objectClass 定义（含 SUP 继承链）生成表单字段描述：
// 必填性来自 MUST/MAY，控件类型来自 LDAP 语法 OID，多值性来自 SINGLE-VALUE。
package schema

import "strings"

// Field 是动态表单的一个字段描述。
type Field struct {
	Name     string `json:"name"`               // 属性名（规范小写）
	Label    string `json:"label"`              // 中文标签（未知属性回退属性名）
	Required bool   `json:"required"`           // 来自 MUST（含继承）
	Multi    bool   `json:"multi"`              // 非 SINGLE-VALUE 即多值
	Control  string `json:"control"`            // text|textarea|number|email|tel|datetime|boolean|binary
	Syntax   string `json:"syntax,omitempty"`   // 语法 OID（去 {len}）
	Desc     string `json:"desc,omitempty"`     // 来自 schema DESC
	ReadOnly bool   `json:"readOnly,omitempty"` // NO-USER-MODIFICATION
}

// FormModel 是某个结构类的编辑表单模型。
type FormModel struct {
	StructuralClass string   `json:"structuralClass"` // 主结构类
	Classes         []string `json:"classes"`         // 建议写入的 objectClass 链
	Fields          []Field  `json:"fields"`
}

// 中文标签表：常见人员/部门/组属性；未收录的属性回退属性名本身。
var attrLabels = map[string]string{
	"cn":                         "姓名 / 名称",
	"sn":                         "姓氏",
	"uid":                        "账号 (uid)",
	"mail":                       "邮箱",
	"mobile":                     "手机",
	"telephonenumber":            "电话",
	"employeenumber":             "工号",
	"title":                      "职务",
	"ou":                         "部门",
	"o":                          "公司",
	"description":                "描述",
	"displayname":                "显示名",
	"l":                          "城市",
	"st":                         "省份",
	"street":                     "街道",
	"postofficebox":              "信箱",
	"postalcode":                 "邮编",
	"roomnumber":                 "房间号",
	"employeetype":               "用工类型",
	"departmentnumber":           "部门编号",
	"labeleduri":                 "主页链接",
	"givenname":                  "名字",
	"initials":                   "缩写",
	"manager":                    "直属上级",
	"secretary":                  "助理",
	"seealso":                    "参见",
	"member":                     "成员",
	"memberuid":                  "成员账号",
	"gidnumber":                  "组 ID (gidNumber)",
	"uidnumber":                  "用户 ID (uidNumber)",
	"loginshell":                 "登录 Shell",
	"homedirectory":              "主目录",
	"gecos":                      "GECOS 备注",
	"userpassword":               "密码",
	"objectclass":                "对象类",
	"telefax":                    "传真",
	"facsimiletelephonenumber":   "传真号",
	"telexnumber":                "电传号",
	"teletexterminalidentifier":  "智能终端标识",
	"x121address":                "X.121 网络地址",
	"internationalisdnnumber":    "ISDN 号码",
	"physicaldeliveryofficename": "办公投递点",
	"preferredlanguage":          "首选语言",
	"businesscategory":           "业务类别",
	"carlicense":                 "车牌号",
	"audio":                      "音频简介",
	"homephone":                  "家庭电话",
	"homepage":                   "个人主页",
	"jpegphoto":                  "照片 (JPEG)",
	"photo":                      "照片",
	"usersmimecertificate":       "S/MIME 证书",
	"usercertificate":            "数字证书",
	"cacertificate":              "CA 证书",
	"certificateserialnumber":    "证书序列号",
	"cnamerecord":                "CNAME 记录",
	"dnsrecord":                  "DNS 记录",
	"mxrecord":                   "MX 记录",
	"arecord":                    "A 记录",
	"owner":                      "负责人",
	"uniquemember":               "成员 (唯一)",
	"dmdname":                    "DMD 名称",
	"roleoccupant":               "角色担任者",
	"membernisnetgroup":          "NIS 成员",
	"nisnetgrouptriple":          "NIS 三元组",
	"iphostnumber":               "主机 IP",
	"ipnetworknumber":            "网络号",
	"macaddress":                 "MAC 地址",
	"bootparameter":              "启动参数",
	"bootfile":                   "启动文件",
	"shadowlastchange":           "密码最后修改日",
	"shadowmax":                  "密码最长有效天数",
	"shadowmin":                  "密码最短有效天数",
	"shadowwarning":              "到期提醒天数",
	"shadowinactive":             "闲置锁定天数",
	"shadowexpire":               "账号过期日",
	"shadowflag":                 "Shadow 标志",
	"structuralobjectclass":      "结构类",
	"entryuuid":                  "全局唯一 ID",
	"entrydn":                    "条目 DN",
	"entrycsn":                   "变更序号",
	"createtimestamp":            "创建时间",
	"modifytimestamp":            "最后修改时间",
	"creatorsname":               "创建者",
	"modifiersname":              "最后修改者",
	"hassubordinates":            "是否有子条目",
	"subschemasubentry":          "Schema 位置",
	"namingcontexts":             "命名上下文",
	"supportedcontrol":           "支持的控制",
	"supportedextension":         "支持的扩展",
	"supportedsaslmechanisms":    "支持的 SASL 机制",
	"supportedldapversion":       "支持的 LDAP 版本",
	"vendorname":                 "厂商名",
	"vendorversion":              "厂商版本",
	"info":                       "备注信息",
	"keyword":                    "关键词",
	"generationqualifier":        "世代称谓",
	"dc":                         "域名组件 (dc)",
	"domaincomponent":            "域名组件 (dc)",
	"c":                          "国家代码",
	"co":                         "国家名",
	"countryname":                "国家名",
	"maillocaladdress":           "本地邮件地址",
	"mailhost":                   "邮件主机",
	"mailroutingaddress":         "邮件路由地址",
	"delegatedto":                "委托给",
	"tea":                        " Tea 属性（遗留）",
}

// AttrLabel 返回属性的中文标签。
func AttrLabel(name string) string {
	if l, ok := attrLabels[strings.ToLower(name)]; ok {
		return l
	}
	return name
}

// 语法 OID → 控件类型
var syntaxControls = map[string]string{
	"1.3.6.1.4.1.1466.115.121.1.15": "text",     // DirectoryString
	"1.3.6.1.4.1.1466.115.121.1.41": "text",     // PostalAddress
	"1.3.6.1.4.1.1466.115.121.1.26": "text",     // IA5String（mail 等特判为 email）
	"1.3.6.1.4.1.1466.115.121.1.50": "tel",      // TelephoneNumber
	"1.3.6.1.4.1.1466.115.121.1.27": "number",   // INTEGER
	"1.3.6.1.4.1.1466.115.121.1.7":  "boolean",  // Boolean
	"1.3.6.1.4.1.1466.115.121.1.24": "datetime", // GeneralizedTime
	"1.3.6.1.4.1.1466.115.121.1.36": "datetime", // NumericString(时间类)
	"1.3.6.1.4.1.1466.115.121.1.28": "binary",   // Binary(证书等)
	"1.3.6.1.4.1.1466.115.121.1.5":  "binary",   // Binary(照片)
	"1.3.6.1.4.1.1466.115.121.1.40": "text",     // OctetString
}

// excludeFromForm 不进表单的属性（操作属性 / 结构字段）。
var excludeFromForm = map[string]bool{
	"objectclass":  true,
	"userpassword": true, // 密码走专门通道（重置/改密），不进普通表单
}

// AttributeTypeSyntaxAttrs：无 SYNTAX 时按属性名兜底（SUP 继承由调用方解析）。
var attrSyntaxFallback = map[string]string{
	"cn": "1.3.6.1.4.1.1466.115.121.1.15",
	"sn": "1.3.6.1.4.1.1466.115.121.1.15",
	"ou": "1.3.6.1.4.1.1466.115.121.1.15",
}

// EffectiveAttrs 沿 SUP 链收集一个 objectClass 的有效 MUST/MAY（去重、保持顺序）。
// 未知父类（schema 未加载的自定义类）按 OpenLDAP 惯例忽略其约束。
func (s *Schema) EffectiveAttrs(className string) (must, may []string, classes []string) {
	seen := map[string]bool{}
	var walk func(name string)
	walk = func(name string) {
		l := strings.ToLower(name)
		if seen[l] {
			return
		}
		seen[l] = true
		oc := s.ocByName[l]
		if oc == nil {
			return // schema 未定义（可能靠 SUP 的兜底），跳过约束收集
		}
		classes = append(classes, oc.Names[0])
		for _, sup := range oc.Sup {
			walk(sup)
		}
		for _, a := range oc.Must {
			if !containsFold(must, a) {
				must = append(must, a)
			}
		}
		for _, a := range oc.May {
			if !containsFold(may, a) && !containsFold(must, a) {
				may = append(may, a)
			}
		}
	}
	walk(className)
	return must, may, classes
}

func containsFold(list []string, v string) bool {
	for _, x := range list {
		if strings.EqualFold(x, v) {
			return true
		}
	}
	return false
}

// FormModel 为指定结构类生成表单模型。
// order 里先 MUST 后 MAY；每类内按 attrLabels 已知属性优先、字母序兜底。
func (s *Schema) FormModel(className string) *FormModel {
	must, may, classes := s.EffectiveAttrs(className)
	fm := &FormModel{StructuralClass: className, Classes: classes}

	field := func(name string, required bool) Field {
		l := strings.ToLower(name)
		f := Field{Name: l, Label: AttrLabel(l), Required: required, Control: "text"}
		if at := s.atByName[l]; at != nil {
			if at.SingleValue {
				f.Multi = false
			} else {
				f.Multi = true
			}
			syn := at.SyntaxOID()
			if syn == "" {
				syn = attrSyntaxFallback[l]
			}
			f.Syntax = syn
			f.Desc = at.Desc
			f.ReadOnly = at.NoUserModification || (at.Usage != "" && at.Usage != "userApplications")
			f.Control = controlFor(l, syn)
			// 长文本属性用 textarea
			if l == "description" {
				f.Control = "textarea"
			}
		} else {
			f.Multi = true // 未知属性默认多值（LDAP 惯例）
		}
		return f
	}

	known := func(f Field) int { // 已知中文标签排前
		if _, ok := attrLabels[f.Name]; ok {
			return 0
		}
		return 1
	}
	// 简单排序：已知在前，字母序在后（稳定）
	sortFields := func(fs []Field) {
		for i := 1; i < len(fs); i++ {
			for j := i; j > 0; j-- {
				a, b := fs[j-1], fs[j]
				if known(a) > known(b) || (known(a) == known(b) && a.Name > b.Name) {
					fs[j-1], fs[j] = b, a
				} else {
					break
				}
			}
		}
	}

	var musts, mays []Field
	for _, a := range must {
		if excludeFromForm[strings.ToLower(a)] {
			continue
		}
		if s.atByName[strings.ToLower(a)] != nil && s.atByName[strings.ToLower(a)].NoUserModification {
			continue // MUST 但不可用户修改（如 posixAccount 的继承场景）——跳过
		}
		musts = append(musts, field(a, true))
	}
	for _, a := range may {
		if excludeFromForm[strings.ToLower(a)] {
			continue
		}
		mays = append(mays, field(a, false))
	}
	sortFields(musts)
	sortFields(mays)
	fm.Fields = append(fm.Fields, musts...)
	fm.Fields = append(fm.Fields, mays...)
	return fm
}

func controlFor(attr, syntax string) string {
	if syntax != "" {
		if c, ok := syntaxControls[syntax]; ok {
			if c == "text" && (attr == "mail" || strings.HasPrefix(attr, "mail")) {
				return "email"
			}
			return c
		}
	}
	return "text"
}
