// M8 端点：目录档案的网页侧管理（R14）——
// ① 修改组织 Base DN（dc）：所有用户后续访问即用新值（持久化，向导保存后优先于环境变量）；
// ② phpLDAPadmin 账密代填桥接页：用连接档案里的管理员凭据自动登录 PLDA。
package api

import (
	"html/template"
	"io"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"sync"

	"newldap/internal/audit"
	"newldap/internal/auth"
)

func (d *Deps) mountM8(mux *http.ServeMux) {
	mux.HandleFunc("PUT /api/v1/profile/basedn", d.withSession(d.handleBaseDNChange))
	mux.HandleFunc("GET /api/v1/plda/bridge", d.withSession(d.handlePLDABridge))
	mux.Handle("/plda/", d.pldaProxy())
}

// handleBaseDNChange 修改本工具管理的 Base DN（组织子树根）。
// 只改"管理范围"，不会物理重命名目录服务器上的后缀；保存后所有用户生效。
func (d *Deps) handleBaseDNChange(w http.ResponseWriter, r *http.Request, s *auth.Session) {
	var req struct{ BaseDN string }
	if err := readJSON(r, &req); err != nil {
		writeErr(w, http.StatusBadRequest, "请求体不合法")
		return
	}
	base := strings.TrimSpace(req.BaseDN)
	base = strings.TrimSuffix(base, ",")
	if base == "" || !strings.Contains(base, "=") || strings.Contains(base, " ") {
		writeErr(w, http.StatusBadRequest, "Base DN 不合法：形如 dc=corp,dc=com 或 ou=xxx,dc=…，不能含空格")
		return
	}
	// 在当前连接的服务器上验证该 DN 真实存在（base 范围读一次）
	if _, err := s.Conn.ReadEntry(base, []string{"1.1"}); err != nil {
		if !writeLdapErr(w, err, "读取新 Base DN") {
			writeErr(w, http.StatusBadRequest, "新 Base DN 在目录服务器上不存在："+err.Error()+"（如需管理别处的目录请用连接向导整档切换）")
		}
		return
	}
	old := d.curProfile().BaseDN
	d.mu.Lock()
	d.Cfg.Profile.BaseDN = base
	d.Cfg.WizardSaved = true // 网页保存的档案优先于环境变量（重启后仍生效）
	_, err := d.saveConfigLocked()
	d.mu.Unlock()
	if err != nil {
		d.Audit.Log(audit.Event{Actor: s.BindDN, IP: clientIP(r), Op: "change-basedn", Result: "fail", Detail: err.Error()})
		writeErr(w, http.StatusInternalServerError, "已在本进程生效，但写入配置文件失败（重启后回退）："+err.Error())
		return
	}
	d.Audit.Log(audit.Event{Actor: s.BindDN, IP: clientIP(r), Op: "change-basedn", DN: base, Result: "ok",
		Detail: "from=" + old})
	writeJSON(w, http.StatusOK, map[string]string{"baseDN": base})
}

// handlePLDABridge phpLDAPadmin 自动登录桥接（服务端完成）：
// 服务端直接向 PLDA 提交登录表单（真实 http POST），拿到 PLDA 会话 Cookie 后
// 原样下发给浏览器（同源 /plda/ 反代，Cookie 域一致），再 303 跳到 /plda/index.php。
// 全程无需浏览器执行任何脚本——任何浏览器（含内嵌 WebView）行为一致。
func (d *Deps) handlePLDABridge(w http.ResponseWriter, r *http.Request, s *auth.Session) {
	p := d.curProfile()
	target := d.Cfg.Server.PLDAURL
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if target == "" {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(bridgeErrPage("未配置 phpLDAPadmin 地址（NEWLDAP_PLDA_URL）")))
		return
	}
	target = strings.TrimSuffix(target, "/")
	if p.BindDN == "" || p.BindPassword == "" {
		w.Write([]byte(bridgeErrPage("连接档案未保存管理员密码（未启用\u201c记住密码\u201d），无法代填。<br/>请直接<a href='/plda/index.php'>打开 phpLDAPadmin 手动登录</a>，账号：<b>" + template.HTMLEscapeString(p.BindDN) + "</b>")))
		return
	}

	// 服务端登录 PLDA（跟随重定向，验证确实登上了）
	form := url.Values{}
	form.Set("cmd", "login")
	form.Set("server_id", "1")
	form.Set("nodecode[login_pass]", "1")
	form.Set("login", p.BindDN)
	form.Set("login_pass", p.BindPassword)
	form.Set("submit", "Authenticate")
	jar := &cookieJar{cookies: map[string][]*http.Cookie{}}
	cli := &http.Client{Jar: jar, CheckRedirect: func(req *http.Request, via []*http.Request) error {
		if len(via) >= 5 {
			return http.ErrUseLastResponse
		}
		return nil
	}}
	resp, err := cli.PostForm(target+"/cmd.php", form)
	if err != nil {
		w.Write([]byte(bridgeErrPage("连接 phpLDAPadmin 失败：" + template.HTMLEscapeString(err.Error()) + "<br/><a href='" + target + "'>直连原始地址</a>")))
		return
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if !strings.Contains(string(body), "logged_in") {
		w.Write([]byte(bridgeErrPage("phpLDAPadmin 拒绝了这组凭据（账号或密码不匹配）。<br/>可<a href='/plda/cmd.php?cmd=login_form&amp;server_id=1'>手动登录</a>，账号：<b>" + template.HTMLEscapeString(p.BindDN) + "</b>")))
		return
	}
	// 会话 Cookie 移交给浏览器：去掉 Domain（同 host 生效），Path 统一 /
	for _, c := range jar.all() {
		http.SetCookie(w, &http.Cookie{
			Name: c.Name, Value: c.Value, Path: "/", HttpOnly: c.HttpOnly,
			Secure: false, SameSite: http.SameSiteLaxMode,
		})
	}
	http.Redirect(w, r, "/plda/index.php", http.StatusSeeOther)
}

// cookieJar 最小 CookieJar 实现（PLDA 登录用）。
type cookieJar struct {
	mu      sync.Mutex
	cookies map[string][]*http.Cookie
}

func (j *cookieJar) SetCookies(u *url.URL, cookies []*http.Cookie) {
	j.mu.Lock()
	defer j.mu.Unlock()
	for _, c := range cookies {
		j.cookies[c.Name] = append(j.cookies[c.Name], c)
	}
}
func (j *cookieJar) Cookies(u *url.URL) []*http.Cookie {
	j.mu.Lock()
	defer j.mu.Unlock()
	var out []*http.Cookie
	seen := map[string]bool{}
	for _, cs := range j.cookies {
		for i := len(cs) - 1; i >= 0; i-- {
			if !seen[cs[i].Name] {
				seen[cs[i].Name] = true
				out = append(out, cs[i])
			}
			break
		}
	}
	return out
}
func (j *cookieJar) all() []*http.Cookie { return j.Cookies(&url.URL{}) }

func bridgeErrPage(msg string) string {
	return `<!doctype html><html lang="zh-CN"><head><meta charset="utf-8"><title>无法自动登录</title>
<style>body{font-family:system-ui,sans-serif;background:#f7f8fc;display:flex;align-items:center;justify-content:center;height:100vh;color:#b3383c}</style></head>
<body><div style="background:#fdf0f0;border:1px solid #f5c6cb;border-radius:12px;padding:24px 30px">` + msg + `</div></body></html>`
}

// pldaProxy 把 phpLDAPadmin 反代到本源 /plda/ 前缀下：
// 与桥接页同源后，自动登录表单提交（同源 POST 导航）不再被浏览器/内嵌环境拦截，
// PLDA 的会话 Cookie 也落在同一 host 上，登录后目录浏览直接可用。
// PLDA 内部使用相对链接（cmd.php、images/…），在 /plda/ 前缀下自然解析。
func (d *Deps) pldaProxy() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		base := strings.TrimSuffix(d.Cfg.Server.PLDAURL, "/")
		if base == "" {
			http.Error(w, "未配置 phpLDAPadmin 地址（NEWLDAP_PLDA_URL）", http.StatusNotFound)
			return
		}
		u, err := url.Parse(base)
		if err != nil {
			http.Error(w, "PLDA 地址不合法："+err.Error(), http.StatusBadGateway)
			return
		}
		proxy := &httputil.ReverseProxy{
			Director: func(req *http.Request) {
				req.URL.Scheme = u.Scheme
				req.URL.Host = u.Host
				req.URL.Path = strings.TrimPrefix(req.URL.Path, "/plda")
				if req.URL.Path == "" {
					req.URL.Path = "/index.php"
				}
				req.Host = u.Host
			},
			ModifyResponse: func(resp *http.Response) error {
				// PLDA 重定向大多为相对地址，少数绝对地址改写回 /plda 前缀
				loc := resp.Header.Get("Location")
				if loc != "" && strings.HasPrefix(loc, base) {
					resp.Header.Set("Location", "/plda"+strings.TrimPrefix(loc, base))
				}
				return nil
			},
		}
		proxy.ServeHTTP(w, r)
	})
}
