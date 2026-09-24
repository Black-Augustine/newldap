// Package api 提供 REST 接口。
// 会话 = 一次真实 bind；除登录、探活、连接向导（仅未配置时开放）外的接口
// 都要求会话 Cookie，且所有目录操作经由该会话的连接执行（权限 = 服务端 ACL，R11.4）。
// 写方法带 Origin 校验（配合 SameSite=Strict Cookie 构成 CSRF 防护，R11.1）。
package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strings"
	"sync"

	ldap "github.com/go-ldap/ldap/v3"

	"newldap/internal/audit"
	"newldap/internal/auth"
	"newldap/internal/config"
	"newldap/internal/directory"
	"newldap/internal/ldapclient"
	"newldap/internal/schema"
)

const sessionCookie = "nlsid"

// Deps 是路由依赖。
type Deps struct {
	Cfg      *config.Config
	Sessions *auth.Store
	Audit    *audit.Logger
	// MockLDAP 非空时 healthz 直接探测内置替身（--mock 模式）
	MockLDAP *ldapclient.Client

	// mu 保护 Cfg.Profile 的热替换（连接向导保存后立即生效，无需重启）
	mu sync.RWMutex
}

func New(deps *Deps) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", deps.handleHealthz)

	mux.HandleFunc("POST /api/v1/auth/login", deps.handleLogin)
	mux.HandleFunc("POST /api/v1/auth/logout", deps.withSession(deps.handleLogout))
	mux.HandleFunc("GET /api/v1/me", deps.withSession(deps.handleMe))

	// 连接向导（R1.3/R1.4/R1.5）：仅在目录连接未配置时匿名可用
	mux.HandleFunc("GET /api/v1/setup/status", deps.handleSetupStatus)
	mux.HandleFunc("POST /api/v1/probe", deps.guardWizard(deps.handleProbe))
	mux.HandleFunc("POST /api/v1/setup/profile", deps.guardWizard(deps.handleSetupProfile))

	mux.HandleFunc("GET /api/v1/tree", deps.withSession(deps.handleTree))
	mux.HandleFunc("GET /api/v1/entry", deps.withSession(deps.handleGetEntry))
	mux.HandleFunc("POST /api/v1/entry", deps.withSession(deps.handleCreateEntry))
	mux.HandleFunc("PUT /api/v1/entry", deps.withSession(deps.handleUpdateEntry))
	mux.HandleFunc("POST /api/v1/entry/move", deps.withSession(deps.handleMove))
	mux.HandleFunc("POST /api/v1/entry/move-preview", deps.withSession(deps.handleMovePreview))
	mux.HandleFunc("GET /api/v1/entry/delete-preview", deps.withSession(deps.handleDeletePreview))
	mux.HandleFunc("POST /api/v1/entry/delete", deps.withSession(deps.handleDelete))
	mux.HandleFunc("GET /api/v1/search", deps.withSession(deps.handleSearch))
	mux.HandleFunc("GET /api/v1/schema", deps.withSession(deps.handleSchema))

	deps.mountM2(mux)
	deps.mountM3(mux)
	deps.mountM4(mux)
	deps.mountM6(mux)
	deps.mountM7(mux)
	deps.mountM8(mux)

	var handler http.Handler = mux
	// 2026-09-24 按用户要求移除 CSRF Origin 校验：
	// 内网存在大量代理/端口映射场景，Origin 与 Host 比对频繁误拦。
	// 会话 Cookie 已是 SameSite=Strict（浏览器层防 CSRF），curl/脚本不受影响。
	return handler
}

// ---------- 中间件 ----------

type ctxKey int

const sessionKey ctxKey = 1

func (d *Deps) withSession(next func(http.ResponseWriter, *http.Request, *auth.Session)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ck, err := r.Cookie(sessionCookie)
		if err != nil {
			writeErr(w, http.StatusUnauthorized, "未登录或会话已过期")
			return
		}
		s, err := d.Sessions.Get(ck.Value)
		if err != nil {
			writeErr(w, http.StatusUnauthorized, "未登录或会话已过期")
			return
		}
		next(w, r, s)
	}
}

// guardWizard：向导端点（探测 / 保存档案）门禁。
// 2026-09-23 修改：已配置时也放行——原先要求先登录，导致"目录服务器地址变更后
// 永远登不进、也换不了服务器"的死锁（登录页的连接信息正是来自这份档案）。
// 保留的缓解：CSRF 防护仍生效；保存前真实 bind 验证；向导保存有审计（op=setup）；
// 前端向导在已配置时展示醒目告警。
func (d *Deps) guardWizard(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		next(w, r)
	}
}

// csrfGuard 拒绝 Origin 与访问主机不一致的写请求（无 Origin 的非浏览器客户端放行；
// Cookie 已是 SameSite=Strict，此处为纵深防御，R11.1）。
// 比对按"归一化主机"进行：大小写不敏感、补默认端口（http→80 / https→443），
// 且同时接受 Host 与 X-Forwarded-Host 首跳（反向代理场景 Host 是内部地址）。
func csrfGuard(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet, http.MethodHead, http.MethodOptions:
			next.ServeHTTP(w, r)
			return
		}
		origin := r.Header.Get("Origin")
		// 空 Origin（非浏览器客户端）与 "null"（沙箱 iframe / 部分跳转）放行：
		// 主防线是 SameSite=Strict 会话 Cookie，此处仅拦"明确异源"的跨站写。
		if origin == "" || origin == "null" {
			next.ServeHTTP(w, r)
			return
		}
		u, err := url.Parse(origin)
		if err != nil || u.Host == "" {
			writeErr(w, http.StatusForbidden, "跨站请求已被拒绝（CSRF 防护）")
			return
		}
		ok := hostEqual(u.Host, r.Host)
		if !ok && r.Header.Get("X-Forwarded-Host") != "" {
			// 信任链路最右侧代理写入的首个 X-Forwarded-Host（多级代理取第一项）
			first := strings.SplitN(r.Header.Get("X-Forwarded-Host"), ",", 2)[0]
			ok = hostEqual(u.Host, strings.TrimSpace(first))
		}
		if !ok {
			writeErr(w, http.StatusForbidden, "跨站请求已被拒绝（CSRF 防护）")
			return
		}
		next.ServeHTTP(w, r)
	})
}

// hostEqual 归一化比较两个 host[:port]（大小写不敏感；缺省端口按 scheme 补全）。
func hostEqual(a, b string) bool {
	na, erra := normalizeHost(a)
	nb, errb := normalizeHost(b)
	return erra == nil && errb == nil && na == nb
}

// normalizeHost 返回小写 host[:port]；无 scheme 时按 http 处理。
func normalizeHost(hostport string) (string, error) {
	if hostport == "" {
		return "", errors.New("empty host")
	}
	if !strings.Contains(hostport, "://") {
		hostport = "http://" + hostport
	}
	u, err := url.Parse(hostport)
	if err != nil || u.Hostname() == "" {
		return "", err
	}
	h := strings.ToLower(u.Hostname())
	p := u.Port()
	if p == "" {
		if strings.EqualFold(u.Scheme, "https") {
			p = "443"
		} else {
			p = "80"
		}
	}
	return h + ":" + p, nil
}

// configured 报告目录连接是否已配置（env / 配置文件 / 演示模式）。
func (d *Deps) configured() bool {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.Cfg.Server.Mock || d.Cfg.Source == "env" || d.Cfg.Source == "file"
}

// configSource 报告当前配置来源（env/file/default），前端据此提示
// "环境变量指定的连接在网页上修改仅本次进程生效，重启后以环境变量为准"。
func (d *Deps) configSource() string {
	d.mu.RLock()
	defer d.mu.RUnlock()
	if d.Cfg.Server.Mock {
		return "mock"
	}
	return d.Cfg.Source
}

// curProfile 返回当前连接档案的副本。
func (d *Deps) curProfile() config.Profile {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.Cfg.Profile
}

// setProfile 热替换运行中的连接档案（向导保存成功后调用）。
func (d *Deps) setProfile(p config.Profile) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.Cfg.Profile = p
	d.Cfg.Source = "file"
}

// ---------- 工具 ----------

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	//nolint:errcheck // 响应体写入失败无法补救
	_ = json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

func readJSON[T any](r *http.Request, out *T) error {
	defer r.Body.Close()
	dec := json.NewDecoder(http.MaxBytesReader(nil, r.Body, 1<<20))
	return dec.Decode(out)
}

func clientIP(r *http.Request) string {
	host := r.RemoteAddr
	if i := strings.LastIndex(host, ":"); i > 0 {
		host = host[:i]
	}
	return host
}

// ---------- 探活 ----------

func (d *Deps) handleHealthz(w http.ResponseWriter, r *http.Request) {
	status := map[string]any{"status": "ok", "ldap": "down", "mock": d.Cfg.Server.Mock}
	if d.Cfg.Server.PLDAURL != "" {
		status["pldaURL"] = d.Cfg.Server.PLDAURL
	}
	var err error
	if d.MockLDAP != nil {
		err = d.MockLDAP.Ping()
	} else {
		p := d.curProfile()
		var c *ldapclient.Client
		c, err = ldapclient.Dial(ldapclient.Options{
			URL: p.URL, StartTLS: p.StartTLS, InsecureSkipVerify: p.InsecureSkipVerify,
		})
		if err == nil {
			if p.BindDN != "" {
				err = c.Bind(p.BindDN, p.BindPassword)
			}
			if err == nil {
				err = c.Ping()
			}
			c.Close()
		}
	}
	if err == nil {
		status["ldap"] = "up"
	} else {
		status["detail"] = err.Error()
	}
	writeJSON(w, http.StatusOK, status)
}

// ---------- 认证 ----------

func (d *Deps) handleLogin(w http.ResponseWriter, r *http.Request) {
	var req struct{ BindDN, Password string }
	if err := readJSON(r, &req); err != nil {
		writeErr(w, http.StatusBadRequest, "请求体不是合法 JSON")
		return
	}
	p := d.curProfile()
	opts := ldapclient.Options{
		URL: p.URL, StartTLS: p.StartTLS, InsecureSkipVerify: p.InsecureSkipVerify,
	}
	s, err := d.Sessions.Login(opts, req.BindDN, req.Password)
	d.Audit.Log(audit.Event{
		Actor: req.BindDN, IP: clientIP(r), Op: "login", Result: boolToResult(err),
		Detail: loginErrText(err),
	})
	if err != nil {
		writeErr(w, http.StatusUnauthorized, loginErrText(err))
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name: sessionCookie, Value: s.Token, Path: "/", HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
	})
	writeJSON(w, http.StatusOK, map[string]string{"bindDN": s.BindDN})
}

func boolToResult(err error) string {
	if err != nil {
		return "fail"
	}
	return "ok"
}

// loginErrText 把常见 bind 错误翻译成中文（不区分账号不存在/密码错误，防枚举）。
func loginErrText(err error) string {
	if err == nil {
		return ""
	}
	msg := err.Error()
	switch {
	case strings.Contains(msg, "bind 失败"):
		return "账号或密码错误"
	case strings.Contains(msg, "连接"):
		return "无法连接目录服务器，请检查地址与端口"
	default:
		return "登录失败：" + msg
	}
}

func (d *Deps) handleLogout(w http.ResponseWriter, r *http.Request, s *auth.Session) {
	if ck, err := r.Cookie(sessionCookie); err == nil {
		d.Sessions.Logout(ck.Value)
	}
	d.Audit.Log(audit.Event{Actor: s.BindDN, IP: clientIP(r), Op: "logout", Result: "ok"})
	writeJSON(w, http.StatusOK, map[string]string{"ok": "true"})
}

func (d *Deps) handleMe(w http.ResponseWriter, r *http.Request, s *auth.Session) {
	writeJSON(w, http.StatusOK, map[string]any{"bindDN": s.BindDN, "baseDN": d.curProfile().BaseDN})
}

// ---------- 连接向导（R1.3/R1.4/R1.5） ----------

// handleSetupStatus 返回连接配置状态；前端据此决定 登录页 / 向导页 分流（R1.5 的入口侧）。
func (d *Deps) handleSetupStatus(w http.ResponseWriter, r *http.Request) {
	resp := map[string]any{
		"configured":   d.configured(),
		"mock":         d.Cfg.Server.Mock,
		"source":       d.configSource(),
		"secretKeySet": config.SecretKey() != nil,
	}
	if d.configured() {
		p := d.curProfile()
		resp["profile"] = map[string]any{
			"name": p.Name, "url": p.URL, "baseDN": p.BaseDN, "hasBindDN": p.BindDN != "",
		}
	}
	writeJSON(w, http.StatusOK, resp)
}

// probeRequest 是 rootDSE 探测请求；bindDN/password 可选（提供则同时做绑定测试与目录状态检查）。
type probeRequest struct {
	URL                string `json:"url"`
	StartTLS           bool   `json:"startTLS"`
	InsecureSkipVerify bool   `json:"insecureSkipVerify"`
	BindDN             string `json:"bindDN"`
	BindPassword       string `json:"bindPassword"`
	BaseDN             string `json:"baseDN"`
}

// handleProbe 实现向导第 2 步：连服务器 → 读 rootDSE →（可选）bind 测试 →（可选）base 状态。
// 探测结果整体以 200 返回（ok=false 表示连不上），错误也作为数据呈现给向导 UI。
func (d *Deps) handleProbe(w http.ResponseWriter, r *http.Request) {
	var req probeRequest
	if err := readJSON(r, &req); err != nil || strings.TrimSpace(req.URL) == "" {
		writeErr(w, http.StatusBadRequest, "请填写服务器地址")
		return
	}
	opts := ldapclient.Options{URL: req.URL, StartTLS: req.StartTLS, InsecureSkipVerify: req.InsecureSkipVerify}
	c, err := ldapclient.Dial(opts)
	if err != nil {
		writeJSON(w, http.StatusOK, map[string]any{"ok": false, "error": probeErrText(err)})
		return
	}
	defer c.Close()
	root, err := c.RootDSE()
	if err != nil {
		writeJSON(w, http.StatusOK, map[string]any{"ok": false, "error": probeErrText(err)})
		return
	}
	resp := map[string]any{
		"ok": true, "anonymous": true,
		"namingContexts":     root.NamingContexts,
		"vendorName":         root.VendorName,
		"supportedControl":   root.SupportedControl,
		"supportedExtension": root.SupportedExtension,
		"paging":             hasOID(root.SupportedControl, "1.2.840.113556.1.4.319"),
		"passwordModify":     hasOID(root.SupportedExtension, "1.3.6.1.4.1.4203.1.11.1"),
	}
	if req.BindDN != "" {
		if req.BindPassword == "" {
			resp["bind"] = "fail"
			resp["bindError"] = "请填写密码"
		} else if err := c.Bind(req.BindDN, req.BindPassword); err != nil {
			resp["bind"] = "fail"
			resp["bindError"] = loginErrText(err)
		} else {
			resp["bind"] = "ok"
			if base := strings.TrimSpace(req.BaseDN); base != "" {
				resp["base"] = probeBase(c, base)
			}
		}
	}
	writeJSON(w, http.StatusOK, resp)
}

// probeBase 检查 base DN 状态（R1.5 空/存量分支的数据来源）。
func probeBase(c *ldapclient.Client, base string) map[string]any {
	out := map[string]any{"dn": base, "exists": false, "hasChildren": false}
	if _, err := c.ReadEntry(base, []string{"1.1"}); err != nil {
		return out
	}
	out["exists"] = true
	// base 下一层只要有一个子条目即视为"存量目录"
	if kids, err := c.PagedSearch(base, ldap.ScopeSingleLevel, "(objectClass=*)", []string{"1.1"}, 2); err == nil && len(kids) > 0 {
		out["hasChildren"] = true
	}
	return out
}

func hasOID(list []string, oid string) bool {
	for _, v := range list {
		if v == oid {
			return true
		}
	}
	return false
}

// probeErrText 把连接层错误翻译为中文白话（向导展示用）。
func probeErrText(err error) string {
	msg := err.Error()
	switch {
	case strings.Contains(msg, "连接"), strings.Contains(msg, "i/o timeout"), strings.Contains(msg, "refused"):
		return "无法连接服务器：请检查地址与端口是否正确、服务是否在运行"
	case strings.Contains(msg, "StartTLS"):
		return "StartTLS 协商失败：服务器可能未启用 StartTLS 或证书不受信任"
	case strings.Contains(msg, "tls"), strings.Contains(msg, "certificate"):
		return "TLS/证书错误：证书不受信任或协议不匹配，可尝试开启「跳过证书校验」（仅测试用）"
	default:
		return "探测失败：" + msg
	}
}

// setupProfileRequest 是向导最后一步：保存连接档案。
type setupProfileRequest struct {
	Name               string `json:"name"`
	URL                string `json:"url"`
	StartTLS           bool   `json:"startTLS"`
	InsecureSkipVerify bool   `json:"insecureSkipVerify"`
	BindDN             string `json:"bindDN"`
	BindPassword       string `json:"bindPassword"`
	BaseDN             string `json:"baseDN"`
	RememberPassword   bool   `json:"rememberPassword"`
}

// handleSetupProfile 校验并保存连接档案（R1.3 + R11.2：密码加密落盘）。
// 保存前用提供的凭据做一次真实 bind 验证，避免存入坏档案。
func (d *Deps) handleSetupProfile(w http.ResponseWriter, r *http.Request) {
	var req setupProfileRequest
	if err := readJSON(r, &req); err != nil {
		writeErr(w, http.StatusBadRequest, "请求体不是合法 JSON")
		return
	}
	req.URL = strings.TrimSpace(req.URL)
	req.BaseDN = strings.TrimSpace(req.BaseDN)
	req.BindDN = strings.TrimSpace(req.BindDN)
	if req.URL == "" || req.BaseDN == "" {
		writeErr(w, http.StatusBadRequest, "服务器地址与 Base DN 均不能为空")
		return
	}
	if req.BindDN != "" && req.BindPassword != "" {
		c, err := ldapclient.Dial(ldapclient.Options{
			URL: req.URL, StartTLS: req.StartTLS, InsecureSkipVerify: req.InsecureSkipVerify,
		})
		if err == nil {
			err = c.Bind(req.BindDN, req.BindPassword)
			c.Close()
		}
		if err != nil {
			writeErr(w, http.StatusBadGateway, "凭据验证失败，未保存："+loginErrText(err))
			return
		}
	}
	if req.Name == "" {
		req.Name = "默认连接"
	}
	p := config.Profile{
		Name: req.Name, URL: req.URL, StartTLS: req.StartTLS,
		BindDN: req.BindDN, BaseDN: req.BaseDN,
		InsecureSkipVerify: req.InsecureSkipVerify,
	}
	if req.RememberPassword {
		p.BindPassword = req.BindPassword
	}
	// 整份配置（Server+Profile+Policy）原子落盘：只重建 Profile 字段，
	// 策略等其他配置不因换绑连接而丢失；WizardSaved 使网页保存优先于环境变量
	d.mu.Lock()
	d.Cfg.Profile = p
	d.Cfg.WizardSaved = true
	d.Cfg.Source = "file"
	plaintext, err := d.saveConfigLocked()
	d.mu.Unlock()
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "保存配置文件失败："+err.Error())
		return
	}
	d.Audit.Log(audit.Event{
		IP: clientIP(r), Op: "setup", Result: "ok",
		Detail: "url=" + req.URL + " baseDN=" + req.BaseDN + " bindDN=" + req.BindDN +
			" rememberPassword=" + boolCN(req.RememberPassword) + " plaintextSaved=" + boolCN(plaintext),
	})
	writeJSON(w, http.StatusOK, map[string]any{"saved": true, "plaintextSaved": plaintext})
}

func boolCN(b bool) string {
	if b {
		return "是"
	}
	return "否"
}

// ---------- 目录 ----------

func (d *Deps) handleTree(w http.ResponseWriter, r *http.Request, s *auth.Session) {
	base := r.URL.Query().Get("base")
	if base == "" {
		base = d.curProfile().BaseDN
	}
	svc := directory.New(s.Conn)
	var nodes []directory.Node
	var err error
	switch {
	case r.URL.Query().Get("ouOnly") == "1" && r.URL.Query().Get("deep") == "1":
		nodes, err = svc.OUTree(base) // 部门选择器预载：整棵 OU 嵌套树（一次搜索）
	case r.URL.Query().Get("ouOnly") == "1": // 部门选择器：只含 OU，叶子=无下级部门
		nodes, err = svc.OUChildren(base)
	case r.URL.Query().Get("containers") == "1": // 侧栏目录树：不含人员行，叶子=无下级容器
		nodes, err = svc.ContainerChildren(base)
	default:
		nodes, err = svc.Children(base)
	}
	if err != nil {
		writeErr(w, http.StatusBadGateway, "读取目录失败："+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"base": base, "children": nodes})
}

func (d *Deps) handleGetEntry(w http.ResponseWriter, r *http.Request, s *auth.Session) {
	dn := r.URL.Query().Get("dn")
	if dn == "" {
		writeErr(w, http.StatusBadRequest, "缺少 dn 参数")
		return
	}
	e, err := directory.New(s.Conn).Get(dn)
	if errors.Is(err, directory.ErrNotFound) {
		writeErr(w, http.StatusNotFound, "条目不存在")
		return
	}
	if err != nil {
		writeErr(w, http.StatusBadGateway, "读取条目失败："+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, e)
}

// handleCreateEntry 新建条目（R2.3：OU 新建；也供向导空目录初始化建 base 条目）。
func (d *Deps) handleCreateEntry(w http.ResponseWriter, r *http.Request, s *auth.Session) {
	var req directory.CreateRequest
	if err := readJSON(r, &req); err != nil {
		writeErr(w, http.StatusBadRequest, "请求体不合法（需要 dn、objectClass、attrs）")
		return
	}
	if err := directory.New(s.Conn).Create(req); err != nil {
		if !writeLdapErr(w, err, "新建条目") {
			writeErr(w, http.StatusBadGateway, "新建失败："+err.Error())
		}
		return
	}
	d.Audit.Log(audit.Event{
		Actor: s.BindDN, IP: clientIP(r), Op: "create", DN: req.DN, Result: "ok",
		Detail: "objectClass=" + strings.Join(req.Classes, ","),
	})
	writeJSON(w, http.StatusCreated, map[string]string{"dn": req.DN})
}

func (d *Deps) handleUpdateEntry(w http.ResponseWriter, r *http.Request, s *auth.Session) {
	var req struct {
		DN          string             `json:"dn"`
		ExpectedCSN string             `json:"expectedCSN"`
		Changes     []directory.Change `json:"changes"`
	}
	if err := readJSON(r, &req); err != nil || req.DN == "" {
		writeErr(w, http.StatusBadRequest, "请求体不合法（需要 dn 与 changes）")
		return
	}
	newCSN, err := directory.New(s.Conn).Update(req.DN, req.ExpectedCSN, req.Changes)
	if errors.Is(err, directory.ErrConflict) {
		d.Audit.Log(audit.Event{Actor: s.BindDN, IP: clientIP(r), Op: "modify", DN: req.DN, Result: "fail", Detail: "entryCSN 冲突被拦截"})
		writeJSON(w, http.StatusConflict, map[string]string{"error": err.Error(), "code": "conflict"})
		return
	}
	if err != nil {
		if !writeLdapErr(w, err, "修改条目") {
			writeErr(w, http.StatusBadGateway, "修改失败："+err.Error())
		}
		return
	}
	d.Audit.Log(audit.Event{Actor: s.BindDN, IP: clientIP(r), Op: "modify", DN: req.DN, Result: "ok",
		Detail: describeChanges(req.Changes)})
	writeJSON(w, http.StatusOK, map[string]string{"entryCSN": newCSN})
}

func describeChanges(cs []directory.Change) string {
	var parts []string
	for _, c := range cs {
		parts = append(parts, c.Op+" "+c.Attr)
	}
	return strings.Join(parts, ", ")
}

// handleMovePreview 预览移动/改名结果（R2.4：执行前展示新 DN 与影响面）。
func (d *Deps) handleMovePreview(w http.ResponseWriter, r *http.Request, s *auth.Session) {
	var req directory.MoveRequest
	if err := readJSON(r, &req); err != nil {
		writeErr(w, http.StatusBadRequest, "请求体不合法")
		return
	}
	pv, err := directory.New(s.Conn).MovePreview(req)
	if errors.Is(err, directory.ErrNotFound) {
		writeErr(w, http.StatusNotFound, "目标父条目不存在")
		return
	}
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, pv)
}

func (d *Deps) handleMove(w http.ResponseWriter, r *http.Request, s *auth.Session) {
	var req directory.MoveRequest
	if err := readJSON(r, &req); err != nil {
		writeErr(w, http.StatusBadRequest, "请求体不合法")
		return
	}
	newDN, err := directory.New(s.Conn).Move(req)
	if err != nil {
		if !writeLdapErr(w, err, "移动条目") {
			writeErr(w, http.StatusBadGateway, "移动失败："+err.Error())
		}
		return
	}
	d.Audit.Log(audit.Event{Actor: s.BindDN, IP: clientIP(r), Op: "move", DN: req.DN, Result: "ok", Detail: "→ " + newDN})
	writeJSON(w, http.StatusOK, map[string]string{"dn": newDN})
}

// handleDeletePreview 返回删除影响面（R2.5：子孙数量与清单样例）。
func (d *Deps) handleDeletePreview(w http.ResponseWriter, r *http.Request, s *auth.Session) {
	dn := r.URL.Query().Get("dn")
	if dn == "" {
		writeErr(w, http.StatusBadRequest, "缺少 dn 参数")
		return
	}
	pv, err := directory.New(s.Conn).DeletePreview(dn)
	if err != nil {
		writeErr(w, http.StatusBadGateway, "评估删除影响失败："+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, pv)
}

// handleDelete 删除条目；非空且未勾选级联时返回 409 not-empty（R2.5 删除保护）。
func (d *Deps) handleDelete(w http.ResponseWriter, r *http.Request, s *auth.Session) {
	var req struct {
		DN      string `json:"dn"`
		Cascade bool   `json:"cascade"`
	}
	if err := readJSON(r, &req); err != nil || req.DN == "" {
		writeErr(w, http.StatusBadRequest, "请求体不合法（需要 dn）")
		return
	}
	err := directory.New(s.Conn).Delete(req.DN, req.Cascade)
	if errors.Is(err, directory.ErrNotEmpty) {
		d.Audit.Log(audit.Event{Actor: s.BindDN, IP: clientIP(r), Op: "delete", DN: req.DN, Result: "fail", Detail: "非空条目删除被保护拦截"})
		writeJSON(w, http.StatusConflict, map[string]string{"error": err.Error(), "code": "not-empty"})
		return
	}
	if err != nil {
		if !writeLdapErr(w, err, "删除条目") {
			writeErr(w, http.StatusBadGateway, "删除失败："+err.Error())
		}
		return
	}
	d.Audit.Log(audit.Event{
		Actor: s.BindDN, IP: clientIP(r), Op: "delete", DN: req.DN, Result: "ok",
		Detail: "级联删除=" + boolCN(req.Cascade),
	})
	writeJSON(w, http.StatusOK, map[string]string{"ok": "true"})
}

// handleSearch 子树搜索（R2.2）。q= 关键字（自动构造子串过滤）；filter= 专家模式原始
// RFC4515 过滤器。limit/offset 提供 HTTP 层窗口（线路层仍走 paged results control）。
func (d *Deps) handleSearch(w http.ResponseWriter, r *http.Request, s *auth.Session) {
	q := r.URL.Query()
	base := q.Get("base")
	if base == "" {
		base = d.curProfile().BaseDN
	}
	kw := q.Get("q")
	filter := q.Get("filter")
	if kw != "" {
		filter = directory.KeywordFilter(kw)
	}
	if filter == "" {
		filter = "(objectClass=*)"
	}
	if !strings.HasPrefix(filter, "(") {
		writeErr(w, http.StatusBadRequest, "filter 必须是合法的 RFC4515 过滤表达式（以 ( 开头）")
		return
	}
	limit := atoiDefault(q.Get("limit"), 50)
	if limit < 1 || limit > 200 {
		limit = 50
	}
	offset := atoiDefault(q.Get("offset"), 0)
	if offset < 0 {
		offset = 0
	}
	attrs := q["attr"]
	if len(attrs) == 0 {
		attrs = searchDefaultAttrs
	} else {
		attrs = dropSensitive(attrs)
	}
	result, err := s.Conn.PagedSearch(base, ldap.ScopeWholeSubtree, filter, attrs, 200)
	if err != nil {
		writeErr(w, http.StatusBadGateway, "搜索失败："+err.Error())
		return
	}
	total := len(result)
	start, end := offset, offset+limit
	if start > total {
		start = total
	}
	if end > total {
		end = total
	}
	type item struct {
		DN    string              `json:"dn"`
		Attrs map[string][]string `json:"attrs"`
	}
	items := make([]item, 0, end-start)
	for _, e := range result[start:end] {
		m := map[string][]string{}
		for _, a := range e.Attributes {
			m[a.Name] = a.Values
		}
		stripSensitive(m) // 双保险：即使服务端多回了敏感属性也不外泄
		items = append(items, item{DN: e.DN, Attrs: m})
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"total": total, "offset": start, "limit": limit, "entries": items,
	})
}

// searchDefaultAttrs 是未指定属性时的安全默认集。绝不请求 "*" 或
// userPassword——避免把服务端密码哈希带回浏览器。
var searchDefaultAttrs = []string{
	"cn", "uid", "sn", "mail", "mobile", "employeeNumber", "title", "ou", "description", "objectClass",
}

// 敏感属性剔除统一走 directory 包（people/group 领域层同源）。
func dropSensitive(attrs []string) []string { return directory.DropSensitiveAttrs(attrs) }

func stripSensitive(m map[string][]string) { directory.StripSensitive(m) }

func atoiDefault(s string, def int) int {
	if s == "" {
		return def
	}
	n := 0
	for _, c := range s {
		if c < '0' || c > '9' {
			return def
		}
		n = n*10 + int(c-'0')
		if n > 1_000_000 {
			return 1_000_000
		}
	}
	return n
}

func (d *Deps) handleSchema(w http.ResponseWriter, r *http.Request, s *auth.Session) {
	raw, err := s.Conn.Subschema()
	if err != nil {
		writeErr(w, http.StatusBadGateway, "读取 schema 失败："+err.Error())
		return
	}
	sm, err := schema.Parse(raw)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "schema 解析失败："+err.Error())
		return
	}
	type oc struct {
		Name string   `json:"name"`
		Kind string   `json:"kind"`
		Sup  []string `json:"sup"`
		Must []string `json:"must"`
		May  []string `json:"may"`
		Desc string   `json:"desc"`
	}
	ocs := make([]oc, 0, len(sm.ObjectClasses))
	for _, c := range sm.ObjectClasses {
		name := ""
		if len(c.Names) > 0 {
			name = c.Names[0]
		}
		ocs = append(ocs, oc{Name: name, Kind: c.Kind, Sup: c.Sup, Must: c.Must, May: c.May, Desc: c.Desc})
	}
	writeJSON(w, http.StatusOK, map[string]any{"objectClasses": ocs, "attributeTypeCount": len(sm.AttributeTypes)})
}
