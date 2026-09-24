// M2 领域端点：人员（R3）、用户组（R4）、schema 表单模型（R5.2）、
// LDIF 粘贴建条目（R5.3）。全部要求会话；写操作全部落审计。
package api

import (
	"strconv"

	"net/http"
	"strings"

	"newldap/internal/audit"
	"newldap/internal/auth"
	"newldap/internal/directory"
	"newldap/internal/group"
	"newldap/internal/people"
	"newldap/internal/schema"
)

func (d *Deps) mountM2(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/entry-view", d.withSession(d.handleEntryView))
	mux.HandleFunc("GET /api/v1/schema/form", d.withSession(d.handleSchemaForm))

	mux.HandleFunc("POST /api/v1/people", d.withSession(d.handlePersonCreate))
	mux.HandleFunc("GET /api/v1/people", d.withSession(d.handlePersonDetail))
	mux.HandleFunc("GET /api/v1/people/list", d.withSession(d.handlePeopleList))
	mux.HandleFunc("GET /api/v1/people/uid-suggest", d.withSession(d.handleUIDSuggest))
	mux.HandleFunc("POST /api/v1/people/password", d.withSession(d.handlePersonPassword))
	mux.HandleFunc("POST /api/v1/people/disable", d.withSession(d.handlePersonDisable))
	mux.HandleFunc("POST /api/v1/people/enable", d.withSession(d.handlePersonEnable))
	mux.HandleFunc("POST /api/v1/people/dept", d.withSession(d.handlePersonDept))

	mux.HandleFunc("GET /api/v1/groups", d.withSession(d.handleGroupsList))
	mux.HandleFunc("POST /api/v1/groups", d.withSession(d.handleGroupCreate))
	mux.HandleFunc("POST /api/v1/groups/members", d.withSession(d.handleGroupMembers))
	mux.HandleFunc("PUT /api/v1/groups", d.withSession(d.handleGroupUpdate))

	mux.HandleFunc("POST /api/v1/entry/ldif", d.withSession(d.handleLDIFCreate))
}

// ---------- schema 表单模型（R5.2） ----------

func (d *Deps) handleSchemaForm(w http.ResponseWriter, r *http.Request, s *auth.Session) {
	class := r.URL.Query().Get("class")
	if class == "" {
		class = "inetOrgPerson"
	}
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
	fm := sm.FormModel(class)
	if fm == nil || len(fm.Fields) == 0 {
		writeErr(w, http.StatusNotFound, "未知的 objectClass："+class)
		return
	}
	writeJSON(w, http.StatusOK, fm)
}

// ---------- 人员（R3） ----------

// handleUIDSuggest 依据姓名生成账号建议（拼音 + 冲突加序号），供新建表单
// 的「按姓名生成」按钮使用；ou 用于唯一性检查范围，缺省用 Base DN。
func (d *Deps) handleUIDSuggest(w http.ResponseWriter, r *http.Request, s *auth.Session) {
	cn := strings.TrimSpace(r.URL.Query().Get("cn"))
	if cn == "" {
		writeErr(w, http.StatusBadRequest, "缺少 cn 参数（姓名）")
		return
	}
	searchBase := d.curProfile().BaseDN
	if searchBase == "" {
		searchBase = strings.TrimSpace(r.URL.Query().Get("ou"))
	}
	uid, err := people.New(s.Conn, d.curProfile().BaseDN).UniqueUID(people.GenerateUID(cn), searchBase)
	if err != nil {
		writeErr(w, http.StatusBadGateway, "生成账号建议失败："+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"uid": uid})
}

func (d *Deps) handlePersonCreate(w http.ResponseWriter, r *http.Request, s *auth.Session) {
	var req people.CreateRequest
	if err := readJSON(r, &req); err != nil {
		writeErr(w, http.StatusBadRequest, "请求体不合法")
		return
	}
	// 录入策略（系统设置）：账号长度 + 指定初始密码时的复杂度
	if err := d.curPolicy().ValidateUID(req.UID); err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	if req.Initial != "" {
		if err := d.curPolicy().ValidatePassword(req.Initial); err != nil {
			writeErr(w, http.StatusBadRequest, err.Error())
			return
		}
	}
	dn, initial, err := people.New(s.Conn, d.curProfile().BaseDN).Create(req)
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	d.Audit.Log(audit.Event{Actor: s.BindDN, IP: clientIP(r), Op: "create-person", DN: dn, Result: "ok",
		Detail: "uid=" + req.UID}) // 初始密码不落审计
	writeJSON(w, http.StatusCreated, map[string]string{"dn": dn, "initialPassword": initial})
}


// handlePeopleList 人员列表（含禁用态；userPassword 只在服务端判断，不出网）。
func (d *Deps) handlePeopleList(w http.ResponseWriter, r *http.Request, s *auth.Session) {
	base := strings.TrimSpace(r.URL.Query().Get("base"))
	if base == "" {
		base = d.curProfile().BaseDN
	}
	list, err := people.New(s.Conn, d.curProfile().BaseDN).List(base, []string{
		"cn", "uid", "sn", "mail", "mobile", "employeeNumber", "title", "description",
	})
	if err != nil {
		writeErr(w, http.StatusBadGateway, "读取人员列表失败："+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"people": list, "total": len(list)})
}

func (d *Deps) handlePersonDetail(w http.ResponseWriter, r *http.Request, s *auth.Session) {
	dn := r.URL.Query().Get("dn")
	if dn == "" {
		writeErr(w, http.StatusBadRequest, "缺少 dn 参数")
		return
	}
	det, err := people.New(s.Conn, d.curProfile().BaseDN).Detail(dn)
	if err != nil {
		writeErr(w, http.StatusNotFound, "读取人员失败："+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, det)
}

func (d *Deps) handlePersonPassword(w http.ResponseWriter, r *http.Request, s *auth.Session) {
	var req struct{ DN, Password string }
	if err := readJSON(r, &req); err != nil || req.DN == "" {
		writeErr(w, http.StatusBadRequest, "请求体不合法（需要 dn）")
		return
	}
	// 管理员指定的密码按策略校验；留空则由生成器产出（本身满足强度）
	if req.Password != "" {
		if err := d.curPolicy().ValidatePassword(req.Password); err != nil {
			writeErr(w, http.StatusBadRequest, err.Error())
			return
		}
	}
	pw, err := people.New(s.Conn, d.curProfile().BaseDN).ResetPassword(req.DN, req.Password)
	if err != nil {
		if !writeLdapErr(w, err, "重置密码") {
			writeErr(w, http.StatusBadGateway, "重置失败："+err.Error())
		}
		return
	}
	d.Audit.Log(audit.Event{Actor: s.BindDN, IP: clientIP(r), Op: "reset-password", DN: req.DN, Result: "ok"})
	writeJSON(w, http.StatusOK, map[string]string{"password": pw})
}

func (d *Deps) handlePersonDisable(w http.ResponseWriter, r *http.Request, s *auth.Session) {
	var req struct{ DN string }
	if err := readJSON(r, &req); err != nil || req.DN == "" {
		writeErr(w, http.StatusBadRequest, "请求体不合法（需要 dn）")
		return
	}
	if err := people.New(s.Conn, d.curProfile().BaseDN).Disable(req.DN); err != nil {
		if !writeLdapErr(w, err, "禁用账号") {
			writeErr(w, http.StatusBadGateway, "禁用失败："+err.Error())
		}
		return
	}
	d.Audit.Log(audit.Event{Actor: s.BindDN, IP: clientIP(r), Op: "disable", DN: req.DN, Result: "ok",
		Detail: "采用 {crypt}! 前缀约定，可逆"})
	writeJSON(w, http.StatusOK, map[string]string{"ok": "true"})
}

func (d *Deps) handlePersonEnable(w http.ResponseWriter, r *http.Request, s *auth.Session) {
	var req struct{ DN string }
	if err := readJSON(r, &req); err != nil || req.DN == "" {
		writeErr(w, http.StatusBadRequest, "请求体不合法（需要 dn）")
		return
	}
	if err := people.New(s.Conn, d.curProfile().BaseDN).Enable(req.DN); err != nil {
		if !writeLdapErr(w, err, "启用账号") {
			writeErr(w, http.StatusBadGateway, "启用失败："+err.Error())
		}
		return
	}
	d.Audit.Log(audit.Event{Actor: s.BindDN, IP: clientIP(r), Op: "enable", DN: req.DN, Result: "ok"})
	writeJSON(w, http.StatusOK, map[string]string{"ok": "true"})
}

func (d *Deps) handlePersonDept(w http.ResponseWriter, r *http.Request, s *auth.Session) {
	var req struct{ DN, NewParent string }
	if err := readJSON(r, &req); err != nil || req.DN == "" || req.NewParent == "" {
		writeErr(w, http.StatusBadRequest, "请求体不合法（需要 dn 与 newParent）")
		return
	}
	newDN, err := people.New(s.Conn, d.curProfile().BaseDN).SetDept(req.DN, req.NewParent)
	if err != nil {
		if !writeLdapErr(w, err, "调动部门") {
			writeErr(w, http.StatusBadGateway, "调动失败："+err.Error())
		}
		return
	}
	d.Audit.Log(audit.Event{Actor: s.BindDN, IP: clientIP(r), Op: "move", DN: req.DN, Result: "ok", Detail: "→ " + newDN})
	writeJSON(w, http.StatusOK, map[string]string{"dn": newDN})
}

// ---------- 用户组（R4） ----------

func (d *Deps) handleGroupsList(w http.ResponseWriter, r *http.Request, s *auth.Session) {
	list, err := group.New(s.Conn, d.curProfile().BaseDN).List()
	if err != nil {
		writeErr(w, http.StatusBadGateway, "读取用户组失败："+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"groups": list, "total": len(list)})
}

func (d *Deps) handleGroupCreate(w http.ResponseWriter, r *http.Request, s *auth.Session) {
	var req group.CreateRequest
	if err := readJSON(r, &req); err != nil {
		writeErr(w, http.StatusBadRequest, "请求体不合法")
		return
	}
	dn, err := group.New(s.Conn, d.curProfile().BaseDN).Create(req)
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	d.Audit.Log(audit.Event{Actor: s.BindDN, IP: clientIP(r), Op: "create-group", DN: dn, Result: "ok",
		Detail: "场景=" + req.Scenario})
	writeJSON(w, http.StatusCreated, map[string]string{"dn": dn})
}

func (d *Deps) handleGroupMembers(w http.ResponseWriter, r *http.Request, s *auth.Session) {
	var req struct {
		DN string              `json:"dn"`
		Ch group.MemberChanges `json:"changes"`
	}
	if err := readJSON(r, &req); err != nil || req.DN == "" {
		writeErr(w, http.StatusBadRequest, "请求体不合法（需要 dn 与 changes）")
		return
	}
	if err := group.New(s.Conn, d.curProfile().BaseDN).UpdateMembers(req.DN, req.Ch); err != nil {
		if !writeLdapErr(w, err, "更新组成员") {
			writeErr(w, http.StatusBadGateway, "成员更新失败："+err.Error())
		}
		return
	}
	d.Audit.Log(audit.Event{Actor: s.BindDN, IP: clientIP(r), Op: "modify", DN: req.DN, Result: "ok",
		Detail: "成员 +" + itoa(len(req.Ch.Add)) + " / -" + itoa(len(req.Ch.Remove))})
	writeJSON(w, http.StatusOK, map[string]string{"ok": "true"})
}

func (d *Deps) handleGroupUpdate(w http.ResponseWriter, r *http.Request, s *auth.Session) {
	var req struct{ DN, Description string }
	if err := readJSON(r, &req); err != nil || req.DN == "" {
		writeErr(w, http.StatusBadRequest, "请求体不合法（需要 dn）")
		return
	}
	if err := group.New(s.Conn, d.curProfile().BaseDN).UpdateDescription(req.DN, req.Description); err != nil {
		if !writeLdapErr(w, err, "更新条目") {
			writeErr(w, http.StatusBadGateway, "更新失败："+err.Error())
		}
		return
	}
	d.Audit.Log(audit.Event{Actor: s.BindDN, IP: clientIP(r), Op: "modify", DN: req.DN, Result: "ok", Detail: "描述"})
	writeJSON(w, http.StatusOK, map[string]string{"ok": "true"})
}

// ---------- LDIF 粘贴建条目（R5.3） ----------

func (d *Deps) handleLDIFCreate(w http.ResponseWriter, r *http.Request, s *auth.Session) {
	var req struct{ LDIF string }
	if err := readJSON(r, &req); err != nil || strings.TrimSpace(req.LDIF) == "" {
		writeErr(w, http.StatusBadRequest, "请求体不合法（需要 ldif）")
		return
	}
	cr, err := directory.ParseLDIFEntry(req.LDIF)
	if err != nil {
		writeErr(w, http.StatusBadRequest, "LDIF 解析失败："+err.Error())
		return
	}
	if err := directory.New(s.Conn).Create(*cr); err != nil {
		if !writeLdapErr(w, err, "创建条目") {
			writeErr(w, http.StatusBadGateway, "创建失败："+err.Error())
		}
		return
	}
	d.Audit.Log(audit.Event{Actor: s.BindDN, IP: clientIP(r), Op: "create", DN: cr.DN, Result: "ok", Detail: "LDIF 粘贴"})
	writeJSON(w, http.StatusCreated, map[string]string{"dn": cr.DN})
}

func itoa(n int) string { return strconv.Itoa(n) }

// handleEntryView 条目详情 + schema 视图（M5+ 重设计：四组属性/双语标签/
// objectClass 信息/可添加类与属性，贴合 LDAP 的展示层）。
func (d *Deps) handleEntryView(w http.ResponseWriter, r *http.Request, s *auth.Session) {
	dn := r.URL.Query().Get("dn")
	if dn == "" {
		writeErr(w, http.StatusBadRequest, "缺少 dn 参数")
		return
	}
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
	e, err := s.Conn.ReadEntry(dn, nil)
	if err != nil {
		writeErr(w, http.StatusNotFound, "条目不存在或无权读取")
		return
	}
	attrs := map[string][]string{}
	for _, a := range e.Attributes {
		attrs[a.Name] = a.Values
	}
	directory.StripSensitive(attrs)
	view := sm.EntryView(attrs)
	writeJSON(w, http.StatusOK, map[string]any{
		"dn": e.DN, "entryCSN": e.GetAttributeValue("entryCSN"), "attrs": attrs, "view": view,
	})
}
