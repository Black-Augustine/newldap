// M6 端点：帮助中心 · 对接指引（R12）。
// 对接专用只读账号：位置可选（默认建议 ou=services，目录结构由用户约定），
// organizationalRole + simpleSecurityObject 标准类承载，创建即真实 LDAP Add；
// 凭据自测：用新连接按所给 DN+密码做一次验证 bind，把结果交给对方系统前先自证。
package api

import (
	"crypto/rand"
	"errors"
	"math/big"
	"net/http"
	"regexp"
	"strings"

	ldap "github.com/go-ldap/ldap/v3"

	"newldap/internal/audit"
	"newldap/internal/auth"
	"newldap/internal/ldapclient"
)

func (d *Deps) mountM6(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/v1/integration/bindacct", d.withSession(d.handleBindAcctCreate))
	mux.HandleFunc("POST /api/v1/integration/bindtest", d.withSession(d.handleBindTest))
}

var bindAcctNameRe = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9._-]{0,31}$`)

// handleBindAcctCreate 创建对接专用只读账号。
// body: {name, parent}；parent 为空 → 默认 ou=services,<base>（不存在时自动创建），
// 也可指定已有任意容器 OU 或根。返回 DN 与一次性展示的初始密码。
func (d *Deps) handleBindAcctCreate(w http.ResponseWriter, r *http.Request, s *auth.Session) {
	var req struct{ Name, Parent string }
	if err := readJSON(r, &req); err != nil {
		writeErr(w, http.StatusBadRequest, "请求体不合法")
		return
	}
	name := strings.TrimSpace(req.Name)
	if !bindAcctNameRe.MatchString(name) {
		writeErr(w, http.StatusBadRequest, "账号名需以字母开头，仅含字母/数字/点/下划线/连字符，长度 1-32")
		return
	}
	base := d.curProfile().BaseDN
	parent := strings.TrimSpace(req.Parent)
	if parent == "" {
		parent = "ou=services," + base
	}
	parent = strings.TrimSuffix(strings.TrimSuffix(parent, ","), " ")
	if parent != base && !strings.HasSuffix(strings.ToLower(parent), ","+strings.ToLower(base)) {
		writeErr(w, http.StatusBadRequest, "创建位置必须在 Base DN（"+base+"）之内")
		return
	}

	// 位置校验 / 自动建 ou=services
	if _, err := s.Conn.ReadEntry(parent, []string{"1.1"}); err != nil {
		if !strings.EqualFold(parent, "ou=services,"+base) {
			if writeLdapErr(w, err, "读取创建位置") {
				return
			}
			writeErr(w, http.StatusBadRequest, "创建位置不存在："+parent)
			return
		}
		if err := s.Conn.AddEntry(parent, map[string][]string{
			"objectClass": {"top", "organizationalUnit"},
			"ou":          {"services"},
			"description": {"对接系统专用账号（NewLDAP 帮助中心自动创建）"},
		}); err != nil {
			if writeLdapErr(w, err, "自动创建 ou=services") {
				return
			}
			writeErr(w, http.StatusBadGateway, "自动创建 ou=services 失败："+err.Error())
			return
		}
	}

	dn := "cn=" + name + "," + parent
	pw, err := genPassword(16)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "生成密码失败："+err.Error())
		return
	}
	err = s.Conn.AddEntry(dn, map[string][]string{
		"objectClass": {"organizationalRole", "simpleSecurityObject"},
		"cn":          {name},
		"description": {"对接只读账号（由 NewLDAP 帮助中心创建）"},
		"userPassword": {pw},
	})
	if err != nil {
		if writeLdapErr(w, err, "创建对接账号") {
			d.Audit.Log(audit.Event{Actor: s.BindDN, IP: clientIP(r), Op: "create-bindacct", DN: dn, Result: "fail", Detail: err.Error()})
			return
		}
		writeErr(w, http.StatusBadGateway, "创建对接账号失败："+err.Error())
		d.Audit.Log(audit.Event{Actor: s.BindDN, IP: clientIP(r), Op: "create-bindacct", DN: dn, Result: "fail", Detail: err.Error()})
		return
	}
	d.Audit.Log(audit.Event{Actor: s.BindDN, IP: clientIP(r), Op: "create-bindacct", DN: dn, Result: "ok",
		Detail: "parent=" + parent}) // 密码不落审计
	writeJSON(w, http.StatusCreated, map[string]string{"dn": dn, "password": pw})
}

// handleBindTest 凭据自测：新开连接用所给 DN+密码 bind 一次。
// 永远 200 返回 {ok, message}（凭据错误是"测试结果"而非接口错误）。
func (d *Deps) handleBindTest(w http.ResponseWriter, r *http.Request, s *auth.Session) {
	var req struct{ DN, Password string }
	if err := readJSON(r, &req); err != nil {
		writeErr(w, http.StatusBadRequest, "请求体不合法")
		return
	}
	p := d.curProfile()
	c, err := ldapclient.Dial(ldapclient.Options{
		URL:                p.URL,
		StartTLS:           p.StartTLS,
		InsecureSkipVerify: p.InsecureSkipVerify,
	})
	if err != nil {
		writeErr(w, http.StatusBadGateway, "无法连接目录服务器："+err.Error())
		return
	}
	defer c.Close()
	err = c.Bind(strings.TrimSpace(req.DN), req.Password)
	result, msg := "ok", "bind 验证通过：这组 DN + 密码能通过目录认证，可以填入对方系统"
	if err != nil {
		result = "fail"
		var le *ldap.Error
		if errors.As(err, &le) && le.ResultCode == ldap.LDAPResultInvalidCredentials {
			msg = "bind 验证失败（49 invalidCredentials）：DN 或密码不正确。常见原因：复制时带了空格、密码已被重置、DN 抄错一段"
		} else {
			msg = "bind 验证失败：" + err.Error()
		}
	}
	d.Audit.Log(audit.Event{Actor: s.BindDN, IP: clientIP(r), Op: "bind-test", DN: req.DN, Result: result})
	writeJSON(w, http.StatusOK, map[string]any{"ok": err == nil, "message": msg})
}

// genPassword 密码学随机口令：大小写字母 + 数字 + 固定符号，保证四类各至少一位。
func genPassword(n int) (string, error) {
	const (
		lo  = "abcdefghjkmnpqrstuvwxyz"
		up  = "ABCDEFGHJKMNPQRSTUVWXYZ"
		dig = "23456789"
		sym = "!#$%^*-_=+"
		all = lo + up + dig + sym
	)
	pick := func(set string) (byte, error) {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(set))))
		if err != nil {
			return 0, err
		}
		return set[n.Int64()], nil
	}
	buf := make([]byte, 0, n)
	for _, set := range []string{up, dig, lo} {
		b, err := pick(set)
		if err != nil {
			return "", err
		}
		buf = append(buf, b)
	}
	for len(buf) < n {
		b, err := pick(all)
		if err != nil {
			return "", err
		}
		buf = append(buf, b)
	}
	// Fisher-Yates 打乱，避免固定位置
	for i := len(buf) - 1; i > 0; i-- {
		j, err := rand.Int(rand.Reader, big.NewInt(int64(i+1)))
		if err != nil {
			return "", err
		}
		buf[i], buf[j.Int64()] = buf[j.Int64()], buf[i]
	}
	return string(buf), nil
}
