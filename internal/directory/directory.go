// Package directory 提供目录树浏览与条目读写（含 entryCSN 冲突检测，
// 《技术架构设计》D3/D4）。Service 绑定在一个已 bind 的 ldapclient 连接上，
// 因此其能力上限 = 该 bind 身份在服务端 ACL 下的权限。
package directory

import (
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"

	ldap "github.com/go-ldap/ldap/v3"

	"newldap/internal/ldapclient"
)

// ErrConflict 表示条目在编辑期间被其他客户端修改（entryCSN 不一致）。
var ErrConflict = errors.New("条目已被其他客户端修改，请刷新后重试")

// sensitiveAttrs 是各领域层统一的敏感属性黑名单（小写键）：
// 任何面向浏览器的条目数据出口都必须先 StripSensitive。
var sensitiveAttrs = map[string]bool{
	"userpassword": true,
}

// StripSensitive 原地剔除条目属性表中的敏感属性。
func StripSensitive(m map[string][]string) {
	for k := range m {
		if sensitiveAttrs[strings.ToLower(k)] {
			delete(m, k)
		}
	}
}

// DropSensitiveAttrs 从请求属性清单中剔除敏感属性（请求侧防线）。
func DropSensitiveAttrs(attrs []string) []string {
	out := make([]string, 0, len(attrs))
	for _, a := range attrs {
		if !sensitiveAttrs[strings.ToLower(a)] {
			out = append(out, a)
		}
	}
	return out
}

// ErrNotFound 表示条目不存在。
var ErrNotFound = errors.New("条目不存在")

// ErrNotEmpty 表示条目非空且未勾选级联删除（R2.5 删除保护）。
var ErrNotEmpty = errors.New("该条目下还有子条目，需确认级联删除后才能删除")

// Service 基于一个连接的目录操作集。
type Service struct {
	C *ldapclient.Client
}

func New(c *ldapclient.Client) *Service { return &Service{C: c} }

// Node 是树的一层子节点。
type Node struct {
	DN          string   `json:"dn"`
	RDN         string   `json:"rdn"`
	Name        string   `json:"name"` // 友好名（ou/uid/cn 的值）
	Classes     []string `json:"objectClass"`
	HasChildren bool     `json:"hasChildren"`
	ChildCount  int      `json:"childCount"` // 直接子条目数（R2.1 节点计数）
	Order       int      `json:"order"`      // 手动排序值（ouOrder 属性，缺省 0 → 按名称）
	DisplayName string   `json:"displayName,omitempty"` // OU 的中文名（第二个 ou 值）
	Children    []Node   `json:"children,omitempty"`
}

// nodeOrder/ouDisplayName 从条目属性解析排序值与中文名（OU 的第二个 ou 值）。
func nodeOrder(e *ldap.Entry) int {
	if v := e.GetAttributeValue("ouOrder"); v != "" {
		if n, err := strconv.Atoi(strings.TrimSpace(v)); err == nil {
			return n
		}
	}
	return 0
}

func ouDisplayName(e *ldap.Entry) string {
	vals := e.GetAttributeValues("ou")
	if len(vals) > 1 && vals[1] != vals[0] {
		return vals[1]
	}
	return ""
}

// sortNodes 固定排序：先按 ouOrder（0 排最后），同序按名称。
func sortNodes(list []Node) []Node {
	sort.SliceStable(list, func(i, j int) bool {
		oi, oj := list[i].Order, list[j].Order
		if oi != oj {
			if oi == 0 {
				return false
			}
			if oj == 0 {
				return true
			}
			return oi < oj
		}
		if list[i].Order == 0 {
			return list[i].Name < list[j].Name
		}
		return false
	})
	return list
}

// Children 返回 base 的直接子节点（一层），并统计每个节点的直接子条目数。
func (s *Service) Children(base string) ([]Node, error) {
	return s.children(base, "(objectClass=*)")
}

// OUChildren 返回 base 的直接子 OU，hasChildren 也按"是否存在子 OU"统计。
// 供部门选择器使用：没有下级部门的节点即叶子（前端不显示展开箭头）。
func (s *Service) OUChildren(base string) ([]Node, error) {
	return s.children(base, "(objectClass=organizationalUnit)")
}

// ContainerChildren 返回 base 的直接子节点（不含人员），hasChildren 也按
// "是否存在非人员子节点"统计。供侧栏目录树使用：树只呈现部门与组等容器，
// 人员行不出现，无下级容器的节点即叶子。
func (s *Service) ContainerChildren(base string) ([]Node, error) {
	return s.children(base, "(!(objectClass=inetOrgPerson))")
}

// OUTree 一次子树搜索返回 base 下全部 OU 的嵌套树（部门选择器预载用）。
// 预载完整树可让 el-tree-select 直接显示预设值——懒加载场景下预设值会被
// Element Plus 在节点就绪前清空（已知行为）。
func (s *Service) OUTree(base string) ([]Node, error) {
	entries, err := s.C.PagedSearch(base, ldap.ScopeWholeSubtree,
		"(objectClass=organizationalUnit)", []string{"ou", "objectClass"}, 2000)
	if err != nil {
		return nil, err
	}
	nodes := map[string]*Node{}
	var order []string
	for _, e := range entries {
		if e.DN == base {
			continue
		}
		n := &Node{DN: e.DN, Classes: e.GetAttributeValues("objectClass")}
		if i := strings.Index(e.DN, ","); i >= 0 {
			n.RDN = e.DN[:i]
		} else {
			n.RDN = e.DN
		}
		n.Name = e.GetAttributeValue("ou")
		if n.Name == "" && strings.Contains(n.RDN, "=") {
			if i := strings.Index(n.RDN, "="); i > 0 {
				n.Name = n.RDN[i+1:]
			}
		}
		n.Order = nodeOrder(e)
		n.DisplayName = ouDisplayName(e)
		nodes[e.DN] = n
		order = append(order, e.DN)
	}
	byName := func(list []Node) []Node { // 同层排序：ouOrder 优先，缺省按名称
		return sortNodes(list)
	}
	var roots []Node
	for _, dn := range order {
		n := nodes[dn]
		parent := ""
		if i := strings.Index(dn, ","); i >= 0 {
			parent = strings.TrimSpace(dn[i+1:])
		}
		if p, ok := nodes[parent]; ok {
			p.Children = append(p.Children, *n)
			p.HasChildren = true
			p.ChildCount++
		} else {
			roots = append(roots, *n)
		}
	}
	// 展开嵌套的子节点也要排序：递归整理
	var fix func(list []Node) []Node
	fix = func(list []Node) []Node {
		for i := range list {
			if list[i].Children != nil {
				list[i].Children = byName(fix(list[i].Children))
			}
		}
		return byName(list)
	}
	return fix(roots), nil
}

func (s *Service) children(base, filter string) ([]Node, error) {
	entries, err := s.C.PagedSearch(base, ldap.ScopeSingleLevel, filter, nil, 200)
	if err != nil {
		return nil, err
	}
	var nodes []Node
	for _, e := range entries {
		n := Node{DN: e.DN, Classes: e.GetAttributeValues("objectClass")}
		if i := strings.Index(e.DN, ","); i >= 0 {
			n.RDN = e.DN[:i]
		} else {
			n.RDN = e.DN
		}
		for _, rdnAttr := range []string{"ou", "uid", "cn"} {
			if v := e.GetAttributeValue(rdnAttr); v != "" {
				n.Name = v
				break
			}
		}
		if n.Name == "" && len(n.RDN) > 0 {
			if i := strings.Index(n.RDN, "="); i > 0 {
				n.Name = n.RDN[i+1:]
			}
		}
		n.Order = nodeOrder(e)
		n.DisplayName = ouDisplayName(e)
		nodes = append(nodes, n)
	}
	// 统计每个节点的直接子条目数（attrs=1.1 的单层搜索，只取 DN）
	for i := range nodes {
		kids, err := s.C.PagedSearch(nodes[i].DN, ldap.ScopeSingleLevel, filter, []string{"1.1"}, 200)
		if err == nil {
			nodes[i].ChildCount = len(kids)
			nodes[i].HasChildren = len(kids) > 0
		}
	}
	return sortNodes(nodes), nil
}

// EntryDetail 是条目详情（属性 + 版本标记）。
type EntryDetail struct {
	DN    string              `json:"dn"`
	Attrs map[string][]string `json:"attrs"`
	CSN   string              `json:"entryCSN"`
}

// Get 读取条目全量属性与 entryCSN。敏感属性（userPassword 等）在任何出口
// 都必须剥离——这里是详情读取的唯一组装点（安全不变量集中强制）。
func (s *Service) Get(dn string) (*EntryDetail, error) {
	e, err := s.C.ReadEntry(dn, nil)
	if err != nil {
		return nil, ErrNotFound
	}
	d := &EntryDetail{DN: e.DN, Attrs: map[string][]string{}, CSN: e.GetAttributeValue("entryCSN")}
	for _, a := range e.Attributes {
		d.Attrs[a.Name] = a.Values
	}
	StripSensitive(d.Attrs)
	return d, nil
}

// Change 是一次属性级变更。
type Change struct {
	Op   string   `json:"op"`   // add | delete | replace
	Attr string   `json:"attr"` // 属性名
	Vals []string `json:"vals"`
}

// Update 以乐观并发控制执行属性级修改：
// 先读 entryCSN 与 expectedCSN 比对，不一致返回 ErrConflict（HTTP 409）。
// 仅提交用户实际修改的属性，绝不触碰未提及字段（LAM 并存纪律，架构 D3）。
func (s *Service) Update(dn, expectedCSN string, changes []Change) (string, error) {
	if len(changes) == 0 {
		return "", errors.New("没有可应用的修改")
	}
	cur, err := s.C.CurrentCSN(dn)
	if err != nil {
		return "", ErrNotFound
	}
	if expectedCSN != "" && cur != "" && expectedCSN != cur {
		return "", ErrConflict
	}
	var lchanges []ldap.Change
	for _, ch := range changes {
		var op uint
		switch ch.Op {
		case "add":
			op = ldap.AddAttribute
		case "delete":
			op = ldap.DeleteAttribute
		case "replace":
			op = ldap.ReplaceAttribute
		default:
			return "", fmt.Errorf("未知操作类型: %q", ch.Op)
		}
		lchanges = append(lchanges, ldap.Change{
			Operation: op,
			Modification: ldap.PartialAttribute{
				Type: ch.Attr,
				Vals: ch.Vals,
			},
		})
	}
	if err := s.C.ModifyAttributes(dn, lchanges); err != nil {
		return "", err
	}
	newCSN, _ := s.C.CurrentCSN(dn)
	return newCSN, nil
}

// MoveRequest 移动/改名（modrdn）。
type MoveRequest struct {
	DN        string `json:"dn"`
	NewRDN    string `json:"newRDN"`    // 如 uid=zhangwei；改名时必填
	NewParent string `json:"newParent"` // 新父 DN；移动时必填
}

// Move 执行移动/改名，返回新 DN。
func (s *Service) Move(req MoveRequest) (string, error) {
	if req.DN == "" {
		return "", errors.New("缺少 dn")
	}
	newRDN := strings.TrimSpace(req.NewRDN)
	parent := strings.TrimSpace(req.NewParent)
	if newRDN == "" {
		if i := strings.Index(req.DN, ","); i > 0 {
			newRDN = req.DN[:i]
		} else {
			return "", errors.New("缺少 newRDN")
		}
	}
	if parent == "" {
		if i := strings.Index(req.DN, ","); i >= 0 {
			parent = req.DN[i+1:]
		}
	}
	if err := s.C.ModRDN(req.DN, newRDN, parent); err != nil {
		return "", err
	}
	return newRDN + "," + parent, nil
}

// MovePreview 预览移动/改名结果（R2.4：操作前展示新 DN 与影响面），不执行任何写操作。
type MovePreview struct {
	OldDN  string `json:"oldDN"`
	NewDN  string `json:"newDN"`
	Rename bool   `json:"rename"` // 是否同时改名
	Moved  int    `json:"moved"`  // 随之迁移的子孙条目数（不含自身）
}

// MovePreview 校验请求并计算新 DN 与受影响条目数；父 DN 不存在时返回 ErrNotFound。
func (s *Service) MovePreview(req MoveRequest) (*MovePreview, error) {
	if req.DN == "" {
		return nil, errors.New("缺少 dn")
	}
	newRDN := strings.TrimSpace(req.NewRDN)
	parent := strings.TrimSpace(req.NewParent)
	if newRDN == "" {
		if i := strings.Index(req.DN, ","); i > 0 {
			newRDN = req.DN[:i]
		} else {
			return nil, errors.New("缺少 newRDN")
		}
	}
	if parent == "" {
		if i := strings.Index(req.DN, ","); i >= 0 {
			parent = req.DN[i+1:]
		}
	}
	newDN := newRDN + "," + parent
	if newDN == req.DN {
		return nil, errors.New("新 DN 与当前 DN 相同")
	}
	// 目标父条目必须已存在
	if _, err := s.C.ReadEntry(parent, []string{"1.1"}); err != nil {
		return nil, ErrNotFound
	}
	pv := &MovePreview{OldDN: req.DN, NewDN: newDN}
	if oldRDN := req.DN; strings.Index(oldRDN, ",") > 0 {
		pv.Rename = oldRDN[:strings.Index(oldRDN, ",")] != newRDN
	} else {
		pv.Rename = oldRDN != newRDN
	}
	if all, err := s.C.PagedSearch(req.DN, ldap.ScopeWholeSubtree, "(objectClass=*)", []string{"1.1"}, 500); err == nil {
		pv.Moved = len(all) - 1
	}
	return pv, nil
}

// KeywordFilter 把用户关键字转成常用属性子串搜索过滤器（R2.2 友好搜索；
// 转义由 ldap.EscapeFilter 保证不被注入过滤表达式）。
func KeywordFilter(kw string) string {
	kw = strings.TrimSpace(kw)
	if kw == "" {
		return "(objectClass=*)"
	}
	esc := ldap.EscapeFilter(kw)
	return "(|(cn=*" + esc + "*)(uid=*" + esc + "*)(sn=*" + esc + "*)(mail=*" + esc +
		"*)(employeeNumber=*" + esc + "*)(ou=*" + esc + "*)(description=*" + esc + "*))"
}

// CreateRequest 新建条目（R2.3 OU 等）。attrs 不必包含 objectClass（由 Classes 提供）。
type CreateRequest struct {
	DN      string              `json:"dn"`
	Classes []string            `json:"objectClass"`
	Attrs   map[string][]string `json:"attrs"`
}

// Create 新建条目。校验 RDN 属性值必须与 DN 中的一致（ou=sales,dc=x 的 attrs 必含 ou=sales），
// 避免服务端 schema 校验报出晦涩错误。
func (s *Service) Create(req CreateRequest) error {
	req.DN = strings.TrimSpace(req.DN)
	if req.DN == "" {
		return errors.New("缺少 dn")
	}
	if len(req.Classes) == 0 {
		return errors.New("缺少 objectClass")
	}
	rdn := req.DN
	if i := strings.Index(req.DN, ","); i > 0 {
		rdn = req.DN[:i]
	}
	eq := strings.Index(rdn, "=")
	if eq <= 0 {
		return fmt.Errorf("dn 的 RDN 不合法（应为 属性=值 形式）: %s", rdn)
	}
	rdnAttr, rdnVal := rdn[:eq], rdn[eq+1:]
	attrs := map[string][]string{}
	for k, v := range req.Attrs {
		attrs[k] = v
	}
	attrs["objectClass"] = req.Classes
	// RDN 属性必须存在且首值等于 RDN 值
	if cur, ok := attrs[rdnAttr]; !ok || len(cur) == 0 || cur[0] != rdnVal {
		attrs[rdnAttr] = append([]string{rdnVal}, cur...)
	}
	return s.C.AddEntry(req.DN, attrs)
}

// DeletePreview 是删除前的影响面评估（R2.5）。
type DeletePreview struct {
	DN       string   `json:"dn"`
	Children int      `json:"children"` // 直接子条目数
	Subtree  int      `json:"subtree"`  // 全部子孙条目数（不含自身）
	Sample   []string `json:"sample"`   // 子孙 DN 样例（最多 20 条，深度优先取）
	Empty    bool     `json:"empty"`    // true = 可直接删除，无需级联确认
}

// DeletePreview 收集删除影响面：子孙数量与样例清单。
func (s *Service) DeletePreview(dn string) (*DeletePreview, error) {
	if strings.TrimSpace(dn) == "" {
		return nil, errors.New("缺少 dn")
	}
	kids, err := s.C.PagedSearch(dn, ldap.ScopeSingleLevel, "(objectClass=*)", []string{"1.1"}, 200)
	if err != nil {
		return nil, err
	}
	all, err := s.C.PagedSearch(dn, ldap.ScopeWholeSubtree, "(objectClass=*)", []string{"1.1"}, 500)
	if err != nil {
		return nil, err
	}
	desc := make([]string, 0, len(all))
	for _, e := range all {
		if e.DN != dn {
			desc = append(desc, e.DN)
		}
	}
	p := &DeletePreview{DN: dn, Children: len(kids), Subtree: len(desc), Empty: len(desc) == 0}
	if len(desc) > 20 {
		p.Sample = desc[:20]
	} else {
		p.Sample = desc
	}
	return p, nil
}

// Delete 删除条目。非空且 cascade=false 时返回 ErrNotEmpty（由 API 层转为 409）；
// cascade=true 时自底向上先删全部子孙再删自身。
func (s *Service) Delete(dn string, cascade bool) error {
	dn = strings.TrimSpace(dn)
	if dn == "" {
		return errors.New("缺少 dn")
	}
	all, err := s.C.PagedSearch(dn, ldap.ScopeWholeSubtree, "(objectClass=*)", []string{"1.1"}, 500)
	if err != nil {
		return err
	}
	var desc []string
	for _, e := range all {
		if e.DN != dn {
			desc = append(desc, e.DN)
		}
	}
	if len(desc) > 0 && !cascade {
		return ErrNotEmpty
	}
	// 按 DN 深度倒序（逗号数多者先删），保证父节点删除时子节点已不存在
	sort.Slice(desc, func(i, j int) bool {
		return strings.Count(desc[i], ",") > strings.Count(desc[j], ",")
	})
	for _, d := range desc {
		if err := s.C.DeleteEntry(d); err != nil {
			return fmt.Errorf("级联删除中止于 %s: %w", d, err)
		}
	}
	return s.C.DeleteEntry(dn)
}
