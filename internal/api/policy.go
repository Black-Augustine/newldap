// M7 端点：系统设置 · 录入与安全策略（R13）。
// 策略保存在本机配置文件（与连接档案同一份 YAML），不是目录数据；
// 读取时归一（0/越界回默认），修改有审计，热生效（无需重启）。
package api

import (
	"net/http"
	"strconv"

	"newldap/internal/audit"
	"newldap/internal/auth"
	"newldap/internal/config"
)

func (d *Deps) mountM7(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/policy", d.withSession(d.handlePolicyGet))
	mux.HandleFunc("PUT /api/v1/policy", d.withSession(d.handlePolicyPut))
}

// curPolicy 返回归一后的策略副本。
func (d *Deps) curPolicy() config.Policy {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.Cfg.Policy.Normalized()
}

func (d *Deps) handlePolicyGet(w http.ResponseWriter, r *http.Request, s *auth.Session) {
	writeJSON(w, http.StatusOK, d.curPolicy())
}

func (d *Deps) handlePolicyPut(w http.ResponseWriter, r *http.Request, s *auth.Session) {
	var req config.Policy
	if err := readJSON(r, &req); err != nil {
		writeErr(w, http.StatusBadRequest, "请求体不合法")
		return
	}
	d.mu.Lock()
	d.Cfg.Policy = req.Normalized()
	policy := d.Cfg.Policy
	_, err := d.saveConfigLocked() // Server+Profile+Policy 整体落盘，避免相互覆盖丢字段
	d.mu.Unlock()
	if err != nil {
		// 落盘失败（如目录只读）时仍保留内存热生效，但要如实告知
		d.Audit.Log(audit.Event{Actor: s.BindDN, IP: clientIP(r), Op: "policy-update", Result: "fail", Detail: err.Error()})
		writeErr(w, http.StatusInternalServerError, "策略已在本进程生效，但写入配置文件失败（重启后将回退）："+err.Error())
		return
	}
	d.Audit.Log(audit.Event{Actor: s.BindDN, IP: clientIP(r), Op: "policy-update", Result: "ok", Detail: policyDetail(policy)})
	writeJSON(w, http.StatusOK, policy)
}

func policyDetail(p config.Policy) string {
	d := "密码≥" + strconv.Itoa(p.PasswordMinLen)
	if p.PasswordComplexity {
		d += "+字母数字"
	}
	d += " uid=" + strconv.Itoa(p.UIDMinLen) + "-" + strconv.Itoa(p.UIDMaxLen)
	if p.EmailAutofill {
		d += " 邮箱自动拼接=开"
	}
	return d
}

// saveConfigLocked 在持有写锁时把当前完整配置（Server+Profile+Policy）落盘。
// 供策略保存与连接向导复用，保证任一路径保存都不丢其他字段。
func (d *Deps) saveConfigLocked() (plaintext bool, err error) {
	return (&config.Config{Server: d.Cfg.Server, Profile: d.Cfg.Profile, Policy: d.Cfg.Policy, WizardSaved: d.Cfg.WizardSaved}).
		SaveToFile(config.ActiveConfigFile())
}
