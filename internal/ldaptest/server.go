// Package ldaptest 提供进程内 LDAP 服务测试替身（基于 vjeantet/ldapserver）。
//
// 用途（M0 技术验证）：
//  1. 在没有 Docker/OpenLDAP 的机器上跑通 go-ldap 客户端全链路（bind/搜索/
//     分页/增改/modrdn），验证 BER 线协议兼容性；
//  2. e2e 冒烟与单元测试的后端；
//  3. cmd/server --mock 演示模式的数据源。
//
// 与真实 OpenLDAP 的差异（诚实声明）：
//   - 不校验 schema 与 ACL，任何 bind 成功者可做任意操作；
//   - 分页控件被忽略（一次返回全部），由客户端宽容处理——这本身是
//     ldapclient.PagedSearch 的设计目标（兼容不支持分页的服务端）；
//   - entryCSN 由替身自行维护，格式仿 OpenLDAP，但无同步语义；
//   - 子串过滤器 (cn=*x*) 已按 RFC4511 initial/any/final 语义实现（大小写
//     不敏感），但比较类（>=、<=）与可扩展匹配仍按宽松处理。
//     真实 OpenLDAP 行为由 CI 中的环境变量门控集成测试覆盖（见
//     internal/ldapclient/integration_test.go）。
package ldaptest

import (
	"fmt"
	"net"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
	"unsafe"

	goldap "newldap/internal/goldapmessage"
	"newldap/internal/ldapserver"

	ldap "github.com/go-ldap/ldap/v3"
)

// Entry 是替身内存中的一条目录条目。
type Entry struct {
	DN    string
	Attrs map[string][]string // 键保留原始大小写
}

// Get 大小写不敏感地取属性值。
func (e *Entry) Get(name string) []string {
	for k, v := range e.Attrs {
		if strings.EqualFold(k, name) {
			return v
		}
	}
	return nil
}

func (e *Entry) set(name string, vals []string) {
	for k := range e.Attrs {
		if strings.EqualFold(k, name) {
			e.Attrs[k] = vals
			return
		}
	}
	e.Attrs[name] = vals
}

func (e *Entry) add(name string, vals []string) {
	cur := e.Get(name)
	seen := map[string]bool{}
	for _, v := range cur {
		seen[strings.ToLower(v)] = true
	}
	for _, v := range vals {
		if !seen[strings.ToLower(v)] {
			cur = append(cur, v)
			seen[strings.ToLower(v)] = true
		}
	}
	if len(cur) > 0 {
		e.set(name, cur)
	}
}

func (e *Entry) del(name string, vals []string) {
	if len(vals) == 0 {
		for k := range e.Attrs {
			if strings.EqualFold(k, name) {
				delete(e.Attrs, k)
			}
		}
		return
	}
	cur := e.Get(name)
	var out []string
	for _, v := range cur {
		drop := false
		for _, d := range vals {
			if strings.EqualFold(v, d) {
				drop = true
			}
		}
		if !drop {
			out = append(out, v)
		}
	}
	e.set(name, out)
}

func (e *Entry) clone() *Entry {
	a := make(map[string][]string, len(e.Attrs))
	for k, v := range e.Attrs {
		a[k] = append([]string(nil), v...)
	}
	return &Entry{DN: e.DN, Attrs: a}
}

// ---------- 内存存储 ----------

type store struct {
	mu     sync.RWMutex
	byDN   map[string]*Entry // 键为小写 DN
	rootDN string
	csnSeq int64
}

func newStore(rootDN string) *store {
	return &store{byDN: map[string]*Entry{}, rootDN: rootDN}
}

func (s *store) key(dn string) string { return strings.ToLower(strings.TrimSpace(dn)) }

func (s *store) get(dn string) *Entry {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if e, ok := s.byDN[s.key(dn)]; ok {
		return e.clone()
	}
	return nil
}

func (s *store) nextCSN() string {
	n := atomic.AddInt64(&s.csnSeq, 1)
	return fmt.Sprintf("%s.%06dZ#%06d#000#000000",
		time.Now().UTC().Format("20060102150405"), n%1000000, n)
}

func (s *store) add(e *Entry) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	k := s.key(e.DN)
	if _, exists := s.byDN[k]; exists {
		return fmt.Errorf("条目已存在: %s", e.DN)
	}
	if e.Attrs == nil {
		e.Attrs = map[string][]string{}
	}
	e.Attrs["entryCSN"] = []string{s.nextCSN()}
	e.Attrs["createTimestamp"] = []string{time.Now().UTC().Format("20060102150405Z")}
	e.Attrs["creatorsName"] = []string{"cn=admin," + s.rootDN}
	e.Attrs["modifiersName"] = []string{"cn=admin," + s.rootDN}
	e.Attrs["modifyTimestamp"] = e.Attrs["createTimestamp"]
	s.byDN[k] = e
	return nil
}

func (s *store) modify(dn string, changes []ldap.Change) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	e, ok := s.byDN[s.key(dn)]
	if !ok {
		return fmt.Errorf("条目不存在: %s", dn)
	}
	for _, ch := range changes {
		attr := ch.Modification.Type
		vals := ch.Modification.Vals
		switch ch.Operation {
		case ldap.AddAttribute:
			e.add(attr, vals)
		case ldap.DeleteAttribute:
			e.del(attr, vals)
		case ldap.ReplaceAttribute:
			if len(vals) == 0 {
				e.del(attr, nil)
			} else {
				e.set(attr, vals)
			}
		}
	}
	e.Attrs["entryCSN"] = []string{s.nextCSN()}
	e.Attrs["modifyTimestamp"] = []string{time.Now().UTC().Format("20060102150405Z")}
	return nil
}

func (s *store) delete(dn string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	k := s.key(dn)
	if _, ok := s.byDN[k]; !ok {
		return fmt.Errorf("条目不存在: %s", dn)
	}
	prefix := k + ","
	for other := range s.byDN {
		if strings.HasPrefix(other, prefix) {
			return fmt.Errorf("条目非空，存在子条目: %s", other)
		}
	}
	delete(s.byDN, k)
	return nil
}

func (s *store) modrdn(dn, newRDN string, deleteOld bool, newSuperior string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	k := s.key(dn)
	e, ok := s.byDN[k]
	if !ok {
		return fmt.Errorf("条目不存在: %s", dn)
	}
	newParent := newSuperior
	if newParent == "" {
		if i := strings.Index(e.DN, ","); i >= 0 {
			newParent = e.DN[i+1:]
		}
	}
	newDN := strings.TrimSpace(newRDN) + "," + strings.TrimSpace(newParent)
	newKey := s.key(newDN)
	if newKey != k {
		if _, exists := s.byDN[newKey]; exists {
			return fmt.Errorf("目标条目已存在: %s", newDN)
		}
	}
	// 更新 RDN 属性值
	rdnAttr := newRDN
	rdnVal := newRDN
	if i := strings.Index(newRDN, "="); i > 0 {
		rdnAttr = newRDN[:i]
		rdnVal = newRDN[i+1:]
	}
	e.set(rdnAttr, []string{rdnVal})
	if deleteOld {
		// 旧 RDN 属性值若与新值不同则移除
		for _, v := range e.Get(rdnAttr) {
			if !strings.EqualFold(v, rdnVal) {
				e.del(rdnAttr, []string{v})
			}
		}
	}
	// 移动整棵子树
	type move struct{ from, to string }
	var moves []move
	for other := range s.byDN {
		if other == k || strings.HasPrefix(other, k+",") {
			orig := s.byDN[other].DN
			moves = append(moves, move{orig, newDN + orig[len(k):]})
		}
	}
	for _, m := range moves {
		ent := s.byDN[s.key(m.from)]
		delete(s.byDN, s.key(m.from))
		ent.DN = m.to
		s.byDN[s.key(m.to)] = ent
	}
	s.byDN[newKey].Attrs["entryCSN"] = []string{s.nextCSN()}
	return nil
}

func (s *store) search(base string, scope int, match func(*Entry) bool) []*Entry {
	s.mu.RLock()
	defer s.mu.RUnlock()
	bk := s.key(base)
	var out []*Entry
	for _, e := range s.byDN {
		if !s.inScope(e, bk, scope) {
			continue
		}
		if match(e) {
			out = append(out, e.clone())
		}
	}
	return out
}

func (s *store) inScope(e *Entry, base string, scope int) bool {
	k := s.key(e.DN)
	switch scope {
	case ldap.ScopeBaseObject:
		return k == base
	case ldap.ScopeSingleLevel:
		if k == base {
			return false
		}
		parent := ""
		if i := strings.Index(k, ","); i >= 0 { // 第一个逗号之后才是完整父 DN
			parent = strings.TrimSpace(k[i+1:])
		}
		return strings.EqualFold(parent, strings.TrimSpace(base)) || (base == "" && !strings.Contains(k, ","))
	default: // ScopeWholeSubtree
		return k == base || strings.HasSuffix(k, ","+base) || (base == "" && true)
	}
}

// ---------- 过滤器求值（goldap 类型树） ----------

func evalFilter(e *Entry, f goldap.Filter) bool {
	switch v := f.(type) {
	case goldap.FilterAnd:
		for _, sub := range v {
			if !evalFilter(e, sub) {
				return false
			}
		}
		return true
	case *goldap.FilterAnd:
		for _, sub := range *v {
			if !evalFilter(e, sub) {
				return false
			}
		}
		return true
	case goldap.FilterOr:
		for _, sub := range v {
			if evalFilter(e, sub) {
				return true
			}
		}
		return false
	case *goldap.FilterOr:
		for _, sub := range *v {
			if evalFilter(e, sub) {
				return true
			}
		}
		return false
	case goldap.FilterNot:
		return !evalFilter(e, v.Filter)
	case *goldap.FilterNot:
		return !evalFilter(e, v.Filter)
	case goldap.FilterEqualityMatch:
		return matchEquality(e, string(v.AttributeDesc()), string(v.AssertionValue()))
	case *goldap.FilterEqualityMatch:
		return matchEquality(e, string(v.AttributeDesc()), string(v.AssertionValue()))
	case goldap.FilterSubstrings:
		return matchSubstrings(e, string(v.Type_()), v.Substrings())
	case *goldap.FilterSubstrings:
		return matchSubstrings(e, string(v.Type_()), v.Substrings())
	case goldap.FilterPresent:
		return len(e.Get(string(v))) > 0
	case *goldap.FilterPresent:
		return len(e.Get(string(*v))) > 0
	default:
		// 比较/可扩展匹配等：M0 替身按宽松处理（见包注释）
		return true
	}
}

func matchEquality(e *Entry, attr, want string) bool {
	for _, v := range e.Get(attr) {
		if strings.EqualFold(v, want) {
			return true
		}
	}
	return false
}

// matchSubstrings 按 RFC4511 substring 断言匹配（initial/any/final，大小写不敏感，
// 对齐 OpenLDAP 常见属性的 caseIgnoreSubstringMatch）。
func matchSubstrings(e *Entry, attr string, subs []goldap.Substring) bool {
	for _, v := range e.Get(attr) {
		s := strings.ToLower(v)
		pos := 0
		ok := true
		for _, sub := range subs {
			var part string
			switch t := sub.(type) {
			case goldap.SubstringInitial:
				part = strings.ToLower(string(t))
				if !strings.HasPrefix(s[pos:], part) {
					ok = false
				} else {
					pos += len(part)
				}
			case goldap.SubstringAny:
				part = strings.ToLower(string(t))
				idx := strings.Index(s[pos:], part)
				if idx < 0 {
					ok = false
				} else {
					pos += idx + len(part)
				}
			case goldap.SubstringFinal:
				part = strings.ToLower(string(t))
				if !strings.HasSuffix(s[pos:], part) {
					ok = false
				} else {
					pos = len(s)
				}
			default:
				ok = false
			}
			if !ok {
				break
			}
		}
		if ok {
			return true
		}
	}
	return false
}

// ---------- 服务器 ----------

// Server 是可启动/停止的进程内 LDAP 服务。
type Server struct {
	srv  *ldapserver.Server
	st   *store
	addr string
}

// Start 启动监听 127.0.0.1 随机端口的替身；seedExample 为 true 时灌入示例组织。
func Start(seedExample bool) (*Server, error) {
	return StartAt(seedExample, "")
}

// StartAt 在指定地址（如 127.0.0.1:3890）启动替身；addr 为空时自动选择随机端口。
// 固定端口供外部 LDAP 客户端（phpLDAPadmin / Apache Directory Studio 等）交叉验证。
func StartAt(seedExample bool, addr string) (*Server, error) {
	st := newStore("dc=example,dc=cn")
	if seedExample {
		seedExampleData(st)
	}
	s := &Server{st: st}

	routes := ldapserver.NewRouteMux()
	routes.Bind(s.handleBind)
	routes.Search(s.handleSearch)
	routes.Add(s.handleAdd)
	routes.Modify(s.handleModify)
	routes.Delete(s.handleDelete)
	routes.NotFound(s.handleOther) // ModifyDN 等无专用路由的操作

	s.srv = ldapserver.NewServer()
	s.srv.Handle(routes)

	if addr != "" {
		errCh := make(chan error, 1)
		go func() { errCh <- s.srv.ListenAndServe(addr) }()
		select {
		case err := <-errCh:
			if err != nil {
				return nil, fmt.Errorf("监听 %s 失败: %w", addr, err)
			}
			return nil, fmt.Errorf("监听意外退出")
		case <-time.After(150 * time.Millisecond):
			s.addr = addr
			return s, nil
		}
	}
	for attempt := 0; attempt < 5; attempt++ {
		ln, err := net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			return nil, err
		}
		addr := ln.Addr().String()
		ln.Close()
		errCh := make(chan error, 1)
		go func(a string) { errCh <- s.srv.ListenAndServe(a) }(addr)
		select {
		case err := <-errCh: // 端口被抢占，换一个再试
			if err != nil {
				continue
			}
			return nil, fmt.Errorf("监听意外退出")
		case <-time.After(150 * time.Millisecond):
			s.addr = addr
			return s, nil
		}
	}
	return nil, fmt.Errorf("无法为测试替身分配端口")
}

// Addr 返回 ldap:// 形式的服务地址。
func (s *Server) Addr() string { return "ldap://" + s.addr }

// Stop 停止服务。
func (s *Server) Stop() { s.srv.Stop() }

// Entry 暴露条目快照（测试断言用）。
func (s *Server) Entry(dn string) *Entry { return s.st.get(dn) }

// Port 返回端口号（集成测试里拼地址用）。
func (s *Server) Port() int {
	i := strings.LastIndex(s.addr, ":")
	p, _ := strconv.Atoi(s.addr[i+1:])
	return p
}

func ldapErr(w ldapserver.ResponseWriter, code int, msg string) {
	res := ldapserver.NewResponse(code)
	res.SetDiagnosticMessage(msg)
	w.Write(res)
}

// writeEntry 用 goldap 类型写出一条搜索结果。
// 多值属性必须一次 AddAttribute 传入全部值（多次调用会拆成多个同名
// Attribute TLV，客户端只认第一个）。
func writeEntry(w ldapserver.ResponseWriter, dn string, attrs map[string][]string, include func(attr string) bool) {
	res := ldapserver.NewSearchResultEntry(dn)
	for attr, vals := range attrs {
		if !include(attr) {
			continue
		}
		conv := make([]goldap.AttributeValue, len(vals))
		for i, v := range vals {
			conv[i] = goldap.AttributeValue(v)
		}
		res.AddAttribute(goldap.AttributeDescription(attr), conv...)
	}
	w.Write(res)
}

func (s *Server) handleBind(w ldapserver.ResponseWriter, m *ldapserver.Message) {
	r := m.GetBindRequest()
	dn := string(r.Name())
	pw := string(r.AuthenticationSimple())
	if dn == "" {
		w.Write(ldapserver.NewBindResponse(ldap.LDAPResultSuccess))
		return
	}
	e := s.st.get(dn)
	if e == nil {
		w.Write(bindErr(ldap.LDAPResultInvalidCredentials, "账号或密码错误"))
		return
	}
	pws := e.Get("userPassword")
	ok := false
	for _, v := range pws {
		if v == pw {
			ok = true
		}
	}
	if !ok {
		w.Write(bindErr(ldap.LDAPResultInvalidCredentials, "账号或密码错误"))
		return
	}
	w.Write(ldapserver.NewBindResponse(ldap.LDAPResultSuccess))
}

// bindErr 构造 bind 错误响应（必须用 BindResponse，客户端才能解析出错误码）。
func bindErr(code int, msg string) goldap.BindResponse {
	res := ldapserver.NewBindResponse(code)
	res.SetDiagnosticMessage(msg)
	return res
}

// subschemaLines 是替身自述的迷你 schema（与 internal/schema fixture 同源的精简版）。
var subschemaLines = map[string][]string{
	"objectClasses": {
		"( 2.5.6.5 NAME 'organizationalUnit' DESC 'RFC2256: an organizational unit' SUP top STRUCTURAL MUST ou MAY ( userPassword $ description $ ouOrder ) )",
		"( 2.5.6.10 NAME 'simpleSecurityObject' DESC 'RFC2256: an object with a simple password' SUP top AUXILIARY MUST userPassword )",
		"( 2.5.6.8 NAME 'organizationalRole' DESC 'RFC2256: an organizational role' SUP top STRUCTURAL MUST cn MAY ( userPassword $ seeAlso $ description $ mail $ telephoneNumber ) )",
		"( 2.5.6.6 NAME 'person' DESC 'RFC2256: a person' SUP top STRUCTURAL MUST ( sn $ cn ) MAY ( userPassword $ telephoneNumber $ seeAlso $ description ) )",
		"( 2.5.6.7 NAME 'organizationalPerson' DESC 'RFC2256: an organizational person' SUP person STRUCTURAL MAY ( title $ ou $ l $ mobile $ mail ) )",
		"( 2.16.840.1.113730.3.2.2 NAME 'inetOrgPerson' DESC 'RFC2798: Internet Organizational Person' SUP organizationalPerson STRUCTURAL MAY ( audio $ businessCategory $ carLicense $ departmentNumber $ displayName $ employeeNumber $ employeeType $ givenName $ labeledURI $ mail $ manager $ mobile $ uid $ title ) )",
		"( 2.5.6.9 NAME 'groupOfNames' DESC 'RFC2256: a group of names (DNs)' SUP top STRUCTURAL MUST ( member $ cn ) MAY ( description $ owner $ ou $ o ) )",
		"( 1.3.6.1.1.1.2.0 NAME 'posixAccount' DESC 'Abstraction of an account with POSIX attributes' SUP top AUXILIARY MUST ( cn $ uid $ uidNumber $ gidNumber $ homeDirectory ) MAY ( userPassword $ loginShell ) )",
		"( 1.3.6.1.1.1.2.2 NAME 'posixGroup' DESC 'Abstraction of a group of accounts' SUP top STRUCTURAL MUST ( cn $ gidNumber ) MAY ( userPassword $ memberUid $ description ) )",
	},
	"attributeTypes": {
		"( 2.5.4.41 NAME 'name' DESC 'RFC4519: common supertype of name attributes' EQUALITY caseIgnoreMatch SYNTAX 1.3.6.1.4.1.1466.115.121.1.15{32768} )",
		"( 2.5.4.3 NAME ( 'cn' 'commonName' ) SUP name )",
		"( 2.5.4.4 NAME ( 'sn' 'surname' ) EQUALITY caseIgnoreMatch SYNTAX 1.3.6.1.4.1.1466.115.121.1.15{64} )",
		"( 0.9.2342.19200300.100.1.1 NAME ( 'uid' 'userId' ) DESC 'RFC4519: user identifier' EQUALITY caseIgnoreMatch SYNTAX 1.3.6.1.4.1.1466.115.121.1.15{256} )",
		"( 0.9.2342.19200300.100.1.3 NAME ( 'mail' 'rfc822Mailbox' ) EQUALITY caseIgnoreIA5Match SYNTAX 1.3.6.1.4.1.1466.115.121.1.26{256} )",
		"( 2.5.4.20 NAME 'telephoneNumber' EQUALITY telephoneNumberMatch SYNTAX 1.3.6.1.4.1.1466.115.121.1.50 )",
		"( 0.9.2342.19200300.100.1.41 NAME 'mobile' EQUALITY caseIgnoreMatch SYNTAX 1.3.6.1.4.1.1466.115.121.1.50 )",
		"( 2.5.4.12 NAME 'title' EQUALITY caseIgnoreMatch SYNTAX 1.3.6.1.4.1.1466.115.121.1.15{256} )",
		"( 2.5.4.13 NAME 'description' EQUALITY caseIgnoreMatch SYNTAX 1.3.6.1.4.1.1466.115.121.1.15{1024} )",
		"( 2.5.4.11 NAME ( 'ou' 'organizationalUnitName' ) SUP name )",
		"( 1.3.6.1.1.1.1.19 NAME 'gidNumber' DESC 'group id' EQUALITY integerMatch SYNTAX 1.3.6.1.4.1.1466.115.121.1.27 SINGLE-VALUE )",
		"( 2.5.4.35 NAME 'userPassword' EQUALITY octetStringMatch SYNTAX 1.3.6.1.4.1.1466.115.121.1.40{128} )",
		"( 1.3.6.1.4.1.99999.1 NAME 'ouOrder' DESC 'NewLDAP: display order of OU nodes' EQUALITY integerMatch SYNTAX 1.3.6.1.4.1.1466.115.121.1.27 SINGLE-VALUE )",
		"( 1.3.6.1.4.1.4203.666.1.7 NAME 'entryCSN' DESC 'change sequence number of the entry content' EQUALITY CSNMatch SYNTAX 1.3.6.1.4.1.4203.666.2.1 SINGLE-VALUE NO-USER-MODIFICATION USAGE directoryOperation )",
	},
}

func (s *Server) handleSearch(w ldapserver.ResponseWriter, m *ldapserver.Message) {
	r := m.GetSearchRequest()
	base := string(r.BaseObject())
	scope := int(r.Scope())
	filter := r.Filter()

	// 特殊条目：rootDSE 与 subschema
	if scope == ldap.ScopeBaseObject {
		if base == "" {
			writeEntry(w, "", map[string][]string{
				"objectClass":          {"top"},
				"namingContexts":       {s.st.rootDN},
				"supportedLDAPVersion": {"3"},
				"subschemaSubentry":    {"cn=Subschema"},
				"supportedControl":     {ldap.ControlTypePaging},
			}, func(string) bool { return true })
			w.Write(ldapserver.NewSearchResultDoneResponse(ldap.LDAPResultSuccess))
			return
		}
		if strings.EqualFold(base, "cn=Subschema") {
			attrs := map[string][]string{
				"objectClass": {"top", "subentry", "extensibleObject", "subschema"},
			}
			for k, lines := range subschemaLines {
				attrs[k] = lines
			}
			writeEntry(w, "cn=Subschema", attrs, func(string) bool { return true })
			w.Write(ldapserver.NewSearchResultDoneResponse(ldap.LDAPResultSuccess))
			return
		}
	}

	wantsAll := true
	attrSel := r.Attributes()
	var wanted map[string]bool
	if len(attrSel) > 0 {
		wantsAll = false
		wanted = map[string]bool{}
		for _, a := range attrSel {
			name := string(a)
			if name == "*" {
				wantsAll = true
			}
			wanted[strings.ToLower(name)] = true
		}
	}
	isOperational := func(name string) bool {
		switch strings.ToLower(name) {
		case "entrycsn", "createtimestamp", "modifytimestamp", "creatorsname", "modifiersname", "entryuuid":
			return true
		}
		return false
	}
	include := func(attr string) bool {
		if wanted != nil && wanted[strings.ToLower(attr)] {
			return true
		}
		return wantsAll && !isOperational(attr)
	}

	for _, e := range s.st.search(base, scope, func(e *Entry) bool { return evalFilter(e, filter) }) {
		writeEntry(w, e.DN, e.Attrs, include)
	}
	w.Write(ldapserver.NewSearchResultDoneResponse(ldap.LDAPResultSuccess))
}

func (s *Server) handleAdd(w ldapserver.ResponseWriter, m *ldapserver.Message) {
	r := m.GetAddRequest()
	e := &Entry{DN: string(r.Entry()), Attrs: map[string][]string{}}
	for _, attr := range r.Attributes() {
		vals := make([]string, 0, len(attr.Vals()))
		for _, v := range attr.Vals() {
			vals = append(vals, string(v))
		}
		e.Attrs[string(attr.Type_())] = vals
	}
	if err := s.st.add(e); err != nil {
		res := ldapserver.NewAddResponse(ldap.LDAPResultEntryAlreadyExists)
		w.Write(res)
		return
	}
	w.Write(ldapserver.NewAddResponse(ldap.LDAPResultSuccess))
}

func (s *Server) handleModify(w ldapserver.ResponseWriter, m *ldapserver.Message) {
	r := m.GetModifyRequest()
	var changes []ldap.Change
	for _, ch := range r.Changes() {
		op := uint(int32(ch.Operation()))
		pa := ch.Modification()
		vals := make([]string, 0, len(pa.Vals()))
		for _, v := range pa.Vals() {
			vals = append(vals, string(v))
		}
		changes = append(changes, ldap.Change{
			Operation: op,
			Modification: ldap.PartialAttribute{
				Type: string(pa.Type_()),
				Vals: vals,
			},
		})
	}
	if err := s.st.modify(string(r.Object()), changes); err != nil {
		res := ldapserver.NewModifyResponse(ldap.LDAPResultNoSuchObject)
		w.Write(res)
		return
	}
	w.Write(ldapserver.NewModifyResponse(ldap.LDAPResultSuccess))
}

func (s *Server) handleDelete(w ldapserver.ResponseWriter, m *ldapserver.Message) {
	r := m.GetDeleteRequest()
	if err := s.st.delete(string(r)); err != nil {
		// 区分"不存在"(32) 与"非空"(66)：真实 OpenLDAP 对不存在的条目返回 NoSuchObject
		code := ldap.LDAPResultNotAllowedOnNonLeaf
		if strings.Contains(err.Error(), "条目不存在") {
			code = ldap.LDAPResultNoSuchObject
		}
		res := ldapserver.NewDeleteResponse(code)
		w.Write(res)
		return
	}
	w.Write(ldapserver.NewDeleteResponse(ldap.LDAPResultSuccess))
}

// handleOther 处理没有专用路由的操作；M0 只需要 ModifyDN（modrdn）。
// goldap 未给 ModifyDNRequest 提供公开访问器，这里用反射读取其私有字段。
func (s *Server) handleOther(w ldapserver.ResponseWriter, m *ldapserver.Message) {
	req, ok := m.ProtocolOp().(goldap.ModifyDNRequest)
	if !ok {
		ldapErr(w, ldap.LDAPResultUnwillingToPerform, "测试替身不支持该操作")
		return
	}
	v := reflect.ValueOf(&req).Elem()
	entry := unexportedString(v.FieldByName("entry"))
	newRDN := unexportedString(v.FieldByName("newrdn"))
	delOld := unexportedBool(v.FieldByName("deleteoldrdn"))
	newSuperior := ""
	if f := v.FieldByName("newSuperior"); f.Kind() == reflect.Ptr && !f.IsNil() {
		newSuperior = unexportedString(f.Elem())
	}
	if err := s.st.modrdn(entry, newRDN, delOld, newSuperior); err != nil {
		ldapErr(w, ldap.LDAPResultNoSuchObject, err.Error())
		return
	}
	w.Write(goldap.ModifyDNResponse(ldapserver.NewResponse(ldap.LDAPResultSuccess)))
}

func unexportedString(v reflect.Value) string {
	if !v.IsValid() {
		return ""
	}
	r := reflect.NewAt(v.Type(), unsafe.Pointer(v.UnsafeAddr())).Elem() //nolint:gosec
	return r.String()
}

func unexportedBool(v reflect.Value) bool {
	if !v.IsValid() {
		return false
	}
	r := reflect.NewAt(v.Type(), unsafe.Pointer(v.UnsafeAddr())).Elem() //nolint:gosec
	return r.Bool()
}

// ---------- 示例数据 ----------

func seedExampleData(st *store) {
	root := "dc=example,dc=cn"
	mustAdd := func(e *Entry) {
		if err := st.add(e); err != nil {
			panic(err)
		}
	}
	mustAdd(&Entry{DN: root, Attrs: map[string][]string{
		"objectClass": {"top", "dcObject", "organization"},
		"dc":          {"example"},
		"o":           {"示例公司"},
	}})
	mustAdd(&Entry{DN: "cn=admin," + root, Attrs: map[string][]string{
		"objectClass":  {"simpleSecurityObject", "organizationalRole"},
		"cn":           {"admin"},
		"userPassword": {"admin123"},
		"description":  {"目录管理员"},
	}})
	for _, ou := range []string{"tech", "product", "market", "hr", "groups"} {
		mustAdd(&Entry{DN: "ou=" + ou + "," + root, Attrs: map[string][]string{
			"objectClass": {"top", "organizationalUnit"},
			"ou":          {ou},
		}})
	}
	persons := []struct{ uid, name, emp, ou, title string }{
		{"zhangwei", "张伟", "E1001", "tech", "平台研发工程师"},
		{"lina", "李娜", "E1002", "tech", "应用研发工程师"},
		{"wangqiang", "王强", "E1003", "tech", "运维工程师"},
		{"chenjing", "陈静", "E1005", "tech", "测试工程师"},
		{"fengxue", "冯雪", "E1011", "product", "产品经理"},
		{"hanmei", "韩梅", "E1013", "market", "市场专员"},
	}
	for _, p := range persons {
		dn := "uid=" + p.uid + ",ou=" + p.ou + "," + root
		mustAdd(&Entry{DN: dn, Attrs: map[string][]string{
			"objectClass":    {"top", "person", "organizationalPerson", "inetOrgPerson"},
			"uid":            {p.uid},
			"cn":             {p.name},
			"sn":             {string([]rune(p.name)[1:])},
			"employeeNumber": {p.emp},
			"mail":           {p.uid + "@example.cn"},
			"mobile":         {"138-" + p.emp[1:] + "-0000"},
			"title":          {p.title},
			"userPassword":   {"Passw0rd!"},
		}})
	}
	mustAdd(&Entry{DN: "cn=vpn-access,ou=groups," + root, Attrs: map[string][]string{
		"objectClass": {"top", "groupOfNames"},
		"cn":          {"vpn-access"},
		"description": {"远程 VPN 接入授权"},
		"member": {
			"uid=zhangwei,ou=tech," + root,
			"uid=fengxue,ou=product," + root,
			"uid=hanmei,ou=market," + root,
		},
	}})
}
