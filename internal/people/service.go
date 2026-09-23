// Package people 人员领域层（R3.1~R3.5）：
// 简单模式的人员创建/详情（含所属组反查）/重置密码/禁用启用/调动部门。
// 底层全部是标准 LDAP 操作，不写私有 objectClass（R8.3）。
package people

import (
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"strings"

	ldap "github.com/go-ldap/ldap/v3"

	"newldap/internal/directory"
	"newldap/internal/ldapclient"
)

const personClasses = "inetOrgPerson"

// Service 绑定在一个已认证连接上。BaseDN 用于账号（uid）全组织唯一性检查。
type Service struct {
	C      *ldapclient.Client
	BaseDN string
}

func New(c *ldapclient.Client, baseDN string) *Service { return &Service{C: c, BaseDN: baseDN} }

// CreateRequest 简单模式新建人员（R3.1）。底层自动处理 objectClass 与 DN。
type CreateRequest struct {
	CN      string `json:"cn"`  // 姓名（必填）
	UID     string `json:"uid"` // 账号（必填；留空时接口层仍会自动生成，兼容 Excel 导入等调用方）
	SN      string `json:"sn"`  // 姓氏（缺省自动取：复姓取前两字，单字姓取第一字）
	OU      string `json:"ou"`  // 所属部门 DN（必填）
	Mail    string `json:"mail"`
	Mobile  string `json:"mobile"`
	EmpNo   string `json:"employeeNumber"`
	Title   string `json:"title"`
	Initial string `json:"initialPassword"` // 缺省自动生成
}

// compoundSurnames 常见复姓（姓氏取姓名前两字）。
var compoundSurnames = map[string]bool{
	"欧阳": true, "司马": true, "上官": true, "诸葛": true, "司徒": true, "皇甫": true,
	"尉迟": true, "长孙": true, "慕容": true, "宇文": true, "公孙": true, "轩辕": true,
	"钟离": true, "南宫": true, "独孤": true, "夏侯": true, "闻人": true, "东方": true,
	"端木": true, "申屠": true, "澹台": true, "公冶": true, "太叔": true,
}

// Surname 取姓名的姓氏：复姓匹配取前两字，否则取第一个字；
// 纯英文姓名（含空格）取最后一个单词。
func Surname(cn string) string {
	cn = strings.TrimSpace(cn)
	if cn == "" {
		return ""
	}
	if isASCII(cn) {
		parts := strings.Fields(cn)
		if len(parts) > 1 {
			return parts[len(parts)-1]
		}
		return cn
	}
	r := []rune(cn)
	if len(r) >= 2 {
		if compoundSurnames[string(r[:2])] {
			return string(r[:2])
		}
	}
	return string(r[0])
}

// Create 新建人员，返回 DN 与初始密码（只出现一次，审计不落值）。
func (s *Service) Create(req CreateRequest) (dn, initialPassword string, err error) {
	req.CN = strings.TrimSpace(req.CN)
	req.UID = strings.TrimSpace(req.UID)
	// 简单模式友好：只填姓名即可——账号留空时自动生成（拼音+去重）
	if req.CN == "" {
		return "", "", errors.New("姓名为必填")
	}
	if req.UID == "" {
		if req.OU == "" {
			return "", "", errors.New("所属部门为必填（账号自动生成需要部门位置）")
		}
		var err error
		req.UID, err = s.UniqueUID(GenerateUID(req.CN), s.BaseDN)
		if err != nil {
			return "", "", err
		}
	}
	if strings.ContainsAny(req.UID, ",=+<>#;\"\\") || strings.Contains(req.UID, "/") {
		return "", "", fmt.Errorf("账号 %q 含非法字符（, = + < > # ; \\ \" /）", req.UID)
	}
	// uid 全组织唯一（BaseDN 子树范围内），重复账号拒绝创建
	uniqueBase := s.BaseDN
	if uniqueBase == "" {
		uniqueBase = req.OU
	}
	if uniqueBase != "" {
		if hit, _ := s.C.PagedSearch(uniqueBase, ldap.ScopeWholeSubtree,
			"(uid="+ldap.EscapeFilter(req.UID)+")", []string{"1.1"}, 2); len(hit) > 0 {
			return "", "", fmt.Errorf("账号 %s 已被使用（uid 需全组织唯一），请换一个", req.UID)
		}
	}
	sn := req.SN
	if sn == "" {
		sn = Surname(req.CN)
	}
	if sn == "" {
		sn = req.CN
	}
	if req.Initial == "" {
		req.Initial, _ = RandomPassword()
	}
	dn = "uid=" + req.UID + "," + strings.TrimSpace(req.OU)
	attrs := map[string][]string{
		"objectClass":  {"top", "person", "organizationalPerson", personClasses},
		"uid":          {req.UID},
		"cn":           {req.CN},
		"sn":           {sn},
		"userPassword": {req.Initial},
	}
	for k, v := range map[string]string{
		"mail": req.Mail, "mobile": req.Mobile,
		"employeeNumber": req.EmpNo, "title": req.Title,
	} {
		if v != "" {
			attrs[k] = []string{v}
		}
	}
	if err := s.C.AddEntry(dn, attrs); err != nil {
		return "", "", err
	}
	return dn, req.Initial, nil
}

// RandomPassword 生成 12 位含大小写/数字/符号的随机密码。
func RandomPassword() (string, error) {
	const lower = "abcdefghijkmnpqrstuvwxyz"
	const upper = "ABCDEFGHJKLMNPQRSTUVWXYZ"
	const digit = "23456789"
	const symbol = "!@#$%&*"
	all := lower + upper + digit + symbol
	pick := func(set string) byte { //nolint:gosec
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(set))))
		if err != nil {
			return set[0]
		}
		return set[n.Int64()]
	}
	buf := []byte{pick(lower), pick(upper), pick(digit), pick(symbol)}
	for len(buf) < 12 {
		buf = append(buf, pick(all))
	}
	// 洗牌
	for i := len(buf) - 1; i > 0; i-- {
		n, _ := rand.Int(rand.Reader, big.NewInt(int64(i+1)))
		j := int(n.Int64())
		buf[i], buf[j] = buf[j], buf[i]
	}
	return string(buf), nil
}

// ListPerson 列表项（含禁用态；密码不出服务端）。
type ListPerson struct {
	DN       string `json:"dn"`
	Attrs    map[string][]string `json:"attrs"`
	Disabled bool   `json:"disabled"`
}

// List 返回 base 子树内的人员列表（一次搜索），禁用态由 userPassword 前缀判定。
// attrs 指定要带回的属性（已剥离敏感属性），供表格/选择器使用。
func (s *Service) List(base string, attrs []string) ([]ListPerson, error) {
	fetch := append([]string{"userPassword"}, attrs...)
	hits, err := s.C.PagedSearch(base, ldap.ScopeWholeSubtree,
		"(objectClass=inetOrgPerson)", fetch, 2000)
	if err != nil {
		return nil, err
	}
	out := make([]ListPerson, 0, len(hits))
	for _, e := range hits {
		m := map[string][]string{}
		for _, a := range e.Attributes {
			if a.Name != "" && !strings.EqualFold(a.Name, "userPassword") {
				m[a.Name] = a.Values
			}
		}
		out = append(out, ListPerson{DN: e.DN, Attrs: m, Disabled: IsDisabled(e.GetAttributeValues("userPassword"))})
	}
	return out, nil
}

// Detail 人员详情 + 所属组反查（R4.2 反向视图：groupOfNames.member=DN 或
// posixGroup.memberUid=uid）。
type GroupRef struct {
	DN   string `json:"dn"`
	Name string `json:"name"`
	Type string `json:"type"` // 权限组 | 登录组
}

type Detail struct {
	*directory.EntryDetail
	Groups   []GroupRef `json:"groups"`
	Disabled bool       `json:"disabled"` // 账号是否处于禁用态（读取密码前缀判断，不回传密码本身）
}

func (s *Service) Detail(dn string) (*Detail, error) {
	e, err := s.C.ReadEntry(dn, nil)
	if err != nil {
		return nil, err
	}
	uid := e.GetAttributeValue("uid")
	d := &Detail{EntryDetail: &directory.EntryDetail{
		DN: e.DN, CSN: e.GetAttributeValue("entryCSN"), Attrs: map[string][]string{},
	}}
	for _, a := range e.Attributes {
		d.Attrs[a.Name] = a.Values
	}
	d.Disabled = IsDisabled(e.GetAttributeValues("userPassword"))
	directory.StripSensitive(d.Attrs) // userPassword 等绝不回传浏览器
	// 反查两类组
	if hits, err := s.C.PagedSearch("", ldap.ScopeWholeSubtree,
		"(&(objectClass=groupOfNames)(member="+ldap.EscapeFilter(dn)+"))",
		[]string{"cn"}, 100); err == nil {
		for _, h := range hits {
			d.Groups = append(d.Groups, GroupRef{DN: h.DN, Name: h.GetAttributeValue("cn"), Type: "权限组"})
		}
	}
	if uid != "" {
		if hits, err := s.C.PagedSearch("", ldap.ScopeWholeSubtree,
			"(&(objectClass=posixGroup)(memberUid="+ldap.EscapeFilter(uid)+"))",
			[]string{"cn"}, 100); err == nil {
			for _, h := range hits {
				d.Groups = append(d.Groups, GroupRef{DN: h.DN, Name: h.GetAttributeValue("cn"), Type: "登录组"})
			}
		}
	}
	return d, nil
}

// ResetPassword 重置密码（R3.3）：password 为空则随机生成；返回新密码（只此一次）。
func (s *Service) ResetPassword(dn, password string) (string, error) {
	if password == "" {
		p, err := RandomPassword()
		if err != nil {
			return "", err
		}
		password = p
	}
	if err := s.C.ModifyAttributes(dn, []ldap.Change{
		{Operation: ldap.ReplaceAttribute, Modification: ldap.PartialAttribute{Type: "userPassword", Vals: []string{password}}},
	}); err != nil {
		return "", err
	}
	return password, nil
}

// 禁用约定（架构 D6）：对存储的密码值加 "{crypt}!" 前缀 —— bind 校验必失败，
// 且前缀可逆（启用时剥离即恢复原密码）。界面需写明采用的方式。
const disablePrefix = "{crypt}!"

// Disable 禁用账号（R3.4）：登录将被拒绝，原密码保留（可逆）。
func (s *Service) Disable(dn string) error {
	e, err := s.C.ReadEntry(dn, []string{"userPassword"})
	if err != nil {
		return err
	}
	var changes []ldap.Change
	for _, pw := range e.GetAttributeValues("userPassword") {
		if strings.HasPrefix(pw, disablePrefix) {
			continue // 已禁用
		}
		changes = append(changes, ldap.Change{
			Operation:    ldap.DeleteAttribute,
			Modification: ldap.PartialAttribute{Type: "userPassword", Vals: []string{pw}},
		}, ldap.Change{
			Operation:    ldap.AddAttribute,
			Modification: ldap.PartialAttribute{Type: "userPassword", Vals: []string{disablePrefix + pw}},
		})
	}
	if len(changes) == 0 {
		return errors.New("账号已是禁用状态")
	}
	return s.C.ModifyAttributes(dn, changes)
}

// Enable 启用账号：剥离禁用前缀，恢复原密码。
func (s *Service) Enable(dn string) error {
	e, err := s.C.ReadEntry(dn, []string{"userPassword"})
	if err != nil {
		return err
	}
	var changes []ldap.Change
	for _, pw := range e.GetAttributeValues("userPassword") {
		if !strings.HasPrefix(pw, disablePrefix) {
			continue
		}
		changes = append(changes, ldap.Change{
			Operation:    ldap.DeleteAttribute,
			Modification: ldap.PartialAttribute{Type: "userPassword", Vals: []string{pw}},
		}, ldap.Change{
			Operation:    ldap.AddAttribute,
			Modification: ldap.PartialAttribute{Type: "userPassword", Vals: []string{strings.TrimPrefix(pw, disablePrefix)}},
		})
	}
	if len(changes) == 0 {
		return errors.New("账号未被禁用")
	}
	return s.C.ModifyAttributes(dn, changes)
}

// IsDisabled 判断人员是否处于禁用态（供列表展示）。
func IsDisabled(pwValues []string) bool {
	for _, pw := range pwValues {
		if strings.HasPrefix(pw, disablePrefix) {
			return true
		}
	}
	return false
}

// SetDept 调动部门（R3.5）= 移动条目（modrdn），属性全保留。
func (s *Service) SetDept(dn, newParentOU string) (string, error) {
	return directory.New(s.C).Move(directory.MoveRequest{DN: dn, NewParent: newParentOU})
}
