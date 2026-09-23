// Package ldapclient 封装 go-ldap 连接与目录操作（M0 spike① 的产品化落点）。
//
// 设计要点：
//   - 支持 ldap://（明文/StartTLS）与 ldaps:// 三种方式；
//   - PagedSearch 宽容处理不支持分页控件的服务端（一次返回全部），
//     并防御 cookie 死循环；
//   - 操作属性（entryCSN 等）需显式请求，ReadEntry 默认带回 entryCSN，
//     供《技术架构设计》D4 的冲突检测使用。
package ldapclient

import (
	"crypto/tls"
	"errors"
	"fmt"
	"strings"
	"time"

	ldap "github.com/go-ldap/ldap/v3"
)

// Options 描述如何连上目录服务器。
type Options struct {
	URL                string // ldap://host:389 或 ldaps://host:636
	StartTLS           bool   // 仅对 ldap:// 有意义
	InsecureSkipVerify bool
}

// Client 是一个已建立的连接（可能已 bind）。非并发安全：由会话层保证串行使用。
type Client struct {
	conn *ldap.Conn
	opts Options
}

// Dial 建立连接（不做 bind）。默认 15s 读写超时：服务端或网络僵死时快速报错，
// 而不是让请求方永远挂起（对真实服务器同样重要的加固）。
func Dial(opts Options) (*Client, error) {
	if opts.URL == "" {
		return nil, errors.New("LDAP URL 为空")
	}
	tlsCfg := &tls.Config{InsecureSkipVerify: opts.InsecureSkipVerify} //nolint:gosec
	conn, err := ldap.DialURL(opts.URL, ldap.DialWithTLSConfig(tlsCfg))
	if err != nil {
		return nil, fmt.Errorf("连接 %s 失败: %w", opts.URL, err)
	}
	conn.SetTimeout(15 * time.Second)
	if opts.StartTLS && strings.HasPrefix(opts.URL, "ldap://") {
		if err := conn.StartTLS(tlsCfg); err != nil {
			conn.Close()
			return nil, fmt.Errorf("StartTLS 升级失败: %w", err)
		}
	}
	return &Client{conn: conn, opts: opts}, nil
}

// Bind 用给定身份认证。
func (c *Client) Bind(dn, password string) error {
	if err := c.conn.Bind(dn, password); err != nil {
		return fmt.Errorf("bind 失败: %w", err)
	}
	return nil
}

func (c *Client) Close() { c.conn.Close() }

// RootDSE 是服务端自述信息（自动探测的数据来源，见 R1.4）。
type RootDSE struct {
	NamingContexts     []string
	SupportedControl   []string
	SupportedExtension []string
	VendorName         string
	SubschemasSubentry string
}

// RootDSE 读取服务器自述（base="" 的 BaseObject 搜索）。
func (c *Client) RootDSE() (*RootDSE, error) {
	req := ldap.NewSearchRequest(
		"", ldap.ScopeBaseObject, ldap.NeverDerefAliases, 0, 0, false,
		"(objectClass=*)",
		[]string{"namingContexts", "supportedControl", "supportedExtension", "vendorName", "subschemaSubentry"},
		nil,
	)
	res, err := c.conn.Search(req)
	if err != nil {
		return nil, fmt.Errorf("读取 rootDSE 失败: %w", err)
	}
	if len(res.Entries) == 0 {
		return nil, errors.New("服务器未返回 rootDSE")
	}
	e := res.Entries[0]
	return &RootDSE{
		NamingContexts:     e.GetAttributeValues("namingContexts"),
		SupportedControl:   e.GetAttributeValues("supportedControl"),
		SupportedExtension: e.GetAttributeValues("supportedExtension"),
		VendorName:         e.GetAttributeValue("vendorName"),
		SubschemasSubentry: e.GetAttributeValue("subschemaSubentry"),
	}, nil
}

// Ping 探活：能否完成一次 rootDSE 读取。
func (c *Client) Ping() error {
	_, err := c.RootDSE()
	return err
}

// PagedSearch 按页大小分批读取全部匹配条目。
// 宽容策略：若服务端不回应分页控件（不支持分页），视为单页结果直接返回；
// 若 cookie 连续相同（服务端异常），中断并报错，防止死循环。
func (c *Client) PagedSearch(base string, scope int, filter string, attrs []string, pageSize uint32) ([]*ldap.Entry, error) {
	if pageSize == 0 {
		pageSize = 100
	}
	var all []*ldap.Entry
	cookie := []byte{}
	for {
		paging := ldap.NewControlPaging(pageSize)
		paging.Cookie = cookie
		req := ldap.NewSearchRequest(
			base, scope, ldap.NeverDerefAliases, 0, 0, false,
			filter, attrs, []ldap.Control{paging},
		)
		res, err := c.conn.Search(req)
		if err != nil {
			return all, fmt.Errorf("搜索失败 (base=%s): %w", base, err)
		}
		all = append(all, res.Entries...)

		var next []byte
		found := false
		for _, ctrl := range res.Controls {
			if p, ok := ctrl.(*ldap.ControlPaging); ok {
				next = p.Cookie
				found = true
			}
		}
		if !found || len(next) == 0 {
			return all, nil // 服务端不分页 / 已到最后一页
		}
		if string(next) == string(cookie) {
			return all, errors.New("服务端分页 cookie 未变化，中止以防死循环")
		}
		cookie = next
	}
}

// operationalAttrs 是默认随条目带回的操作属性。
var operationalAttrs = []string{"entryCSN", "modifyTimestamp", "modifiersName", "createTimestamp", "creatorsName"}

// ReadEntry 读取单条条目；attrs 为空时返回全部用户属性 + 关键操作属性。
func (c *Client) ReadEntry(dn string, attrs []string) (*ldap.Entry, error) {
	if len(attrs) == 0 {
		attrs = append([]string{"*"}, operationalAttrs...)
	}
	req := ldap.NewSearchRequest(
		dn, ldap.ScopeBaseObject, ldap.NeverDerefAliases, 0, 0, false,
		"(objectClass=*)", attrs, nil,
	)
	res, err := c.conn.Search(req)
	if err != nil {
		return nil, fmt.Errorf("读取条目失败: %w", err)
	}
	if len(res.Entries) == 0 {
		return nil, fmt.Errorf("条目不存在: %s", dn)
	}
	return res.Entries[0], nil
}

// CurrentCSN 返回条目当前 entryCSN（空串表示服务端未提供）。
func (c *Client) CurrentCSN(dn string) (string, error) {
	e, err := c.ReadEntry(dn, []string{"entryCSN"})
	if err != nil {
		return "", err
	}
	return e.GetAttributeValue("entryCSN"), nil
}

// AddEntry 新建条目。
func (c *Client) AddEntry(dn string, attrs map[string][]string) error {
	req := ldap.NewAddRequest(dn, nil)
	// objectClass 放前面，便于人工核对 LDIF
	if ocs := attrs["objectClass"]; len(ocs) > 0 {
		req.Attribute("objectClass", ocs)
	}
	for k, v := range attrs {
		if k == "objectClass" {
			continue
		}
		req.Attribute(k, v)
	}
	if err := c.conn.Add(req); err != nil {
		return fmt.Errorf("新建条目失败 (%s): %w", dn, err)
	}
	return nil
}

// ModifyAttributes 对条目做属性级修改（绝不整条替换，见架构 D3）。
func (c *Client) ModifyAttributes(dn string, changes []ldap.Change) error {
	req := ldap.NewModifyRequest(dn, nil)
	for _, ch := range changes {
		switch ch.Operation {
		case ldap.AddAttribute:
			req.Add(ch.Modification.Type, ch.Modification.Vals)
		case ldap.DeleteAttribute:
			req.Delete(ch.Modification.Type, ch.Modification.Vals)
		case ldap.ReplaceAttribute:
			req.Replace(ch.Modification.Type, ch.Modification.Vals)
		default:
			return fmt.Errorf("未知修改操作: %d", ch.Operation)
		}
	}
	if err := c.conn.Modify(req); err != nil {
		return fmt.Errorf("修改条目失败 (%s): %w", dn, err)
	}
	return nil
}

// ModRDN 移动/改名条目（newSuperior 为空表示仅改名）。
func (c *Client) ModRDN(dn, newRDN, newSuperior string) error {
	req := ldap.NewModifyDNRequest(dn, newRDN, true, newSuperior)
	if err := c.conn.ModifyDN(req); err != nil {
		return fmt.Errorf("移动/改名失败 (%s → %s): %w", dn, newRDN, err)
	}
	return nil
}

// DeleteEntry 删除条目（非空子树会收到服务端 notAllowedOnNonLeaf 错误）。
func (c *Client) DeleteEntry(dn string) error {
	req := ldap.NewDelRequest(dn, nil)
	if err := c.conn.Del(req); err != nil {
		return fmt.Errorf("删除条目失败 (%s): %w", dn, err)
	}
	return nil
}

// Subschema 读取 cn=Subschema 的 objectClasses/attributeTypes 原始描述。
func (c *Client) Subschema() (map[string][]string, error) {
	entry := "cn=Subschema"
	if root, err := c.RootDSE(); err == nil && root.SubschemasSubentry != "" {
		entry = root.SubschemasSubentry
	}
	req := ldap.NewSearchRequest(
		entry, ldap.ScopeBaseObject, ldap.NeverDerefAliases, 0, 0, false,
		"(objectClass=subschema)", []string{"objectClasses", "attributeTypes"}, nil,
	)
	res, err := c.conn.Search(req)
	if err != nil {
		return nil, fmt.Errorf("读取 subschema 失败: %w", err)
	}
	if len(res.Entries) == 0 {
		return nil, errors.New("服务器未返回 subschema 条目")
	}
	e := res.Entries[0]
	return map[string][]string{
		"objectClasses":  e.GetAttributeValues("objectClasses"),
		"attributeTypes": e.GetAttributeValues("attributeTypes"),
	}, nil
}

// Probe 是连接向导的探测结果（R1.4/R1.5）。
type Probe struct {
	Vendor          string   `json:"vendor"`
	NamingContexts  []string `json:"namingContexts"`
	SubschemasEntry string   `json:"subschemasSubentry"`
	Paging          bool     `json:"paging"`
	PPolicy         bool     `json:"ppolicy"`
	MemberOf        bool     `json:"memberOf"`
	EntryCount      int      `json:"entryCount"` // ≤ MaxProbeCount；超过显示为该上限
	CountCapped     bool     `json:"countCapped"`
	EmptyDir        bool     `json:"emptyDir"`
}

// MaxProbeCount 探测阶段统计条目数的上限（避免大目录全量计数）。
const MaxProbeCount = 500

// ProbeAndCount 在 bind 成功后探测服务端能力与目录规模。
// base 为空时取 namingContexts[0]。
func ProbeAndCount(c *Client, base string) (*Probe, error) {
	dse, err := c.RootDSE()
	if err != nil {
		return nil, err
	}
	p := &Probe{
		Vendor:          dse.VendorName,
		NamingContexts:  dse.NamingContexts,
		SubschemasEntry: dse.SubschemasSubentry,
	}
	for _, oid := range dse.SupportedControl {
		switch oid {
		case ldap.ControlTypePaging:
			p.Paging = true
		case "1.3.6.1.4.1.42.2.27.8.5.1": // ppolicy request control
			p.PPolicy = true
		}
	}
	if base == "" && len(p.NamingContexts) > 0 {
		base = p.NamingContexts[0]
	}
	// 规模统计（封顶 MaxProbeCount+1）
	entries, err := c.PagedSearch(base, ldap.ScopeWholeSubtree, "(objectClass=*)", []string{"1.1"}, MaxProbeCount+1)
	if err == nil {
		p.EntryCount = len(entries)
		if p.EntryCount > MaxProbeCount {
			p.EntryCount = MaxProbeCount
			p.CountCapped = true
		}
		// 空目录：只有根条目自身（或一条都没有）
		p.EmptyDir = p.EntryCount <= 1
	}
	// memberOf 探测：反向过滤查询能否被服务端接受
	if _, err := c.PagedSearch(base, ldap.ScopeWholeSubtree, "(memberOf=*)", []string{"1.1"}, 1); err == nil {
		p.MemberOf = true
	}
	return p, nil
}
