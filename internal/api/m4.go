// M4 端点：员工自助改密（公开入口，独立限流，防枚举文案）。
package api

import (
	"errors"
	"net/http"
	"sync"
	"time"

	"newldap/internal/audit"
	"newldap/internal/ldapclient"
	"newldap/internal/selfservice"
)

var (
	ipLimiter   *selfservice.RateLimiter
	acctLimiter *selfservice.RateLimiter
	limiterOnce sync.Once
)

func limiters() (ip, acct *selfservice.RateLimiter) {
	limiterOnce.Do(func() {
		ipLimiter = selfservice.NewRateLimiter(10, 10*time.Minute)  // 每 IP 10 次/10 分钟
		acctLimiter = selfservice.NewRateLimiter(5, 10*time.Minute) // 每账号 5 次/10 分钟
	})
	return ipLimiter, acctLimiter
}

func (d *Deps) mountM4(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/v1/selfservice/password", d.handleSelfPassword)
}

// handleSelfPassword 公开改密（R9.1/R9.2）：
// 账号+旧密码自 bind 验证 → 改自己的 userPassword。
// 安全：统一错误文案防账号枚举；按 IP 与账号双维度限流；审计不落密码值。
func (d *Deps) handleSelfPassword(w http.ResponseWriter, r *http.Request) {
	// R9.4：开启强制后仅接受 HTTPS（直连 TLS 或反向代理声明 X-Forwarded-Proto: https）
	if d.Cfg.Server.ForceHTTPS && r.TLS == nil && r.Header.Get("X-Forwarded-Proto") != "https" {
		writeErr(w, http.StatusBadRequest, "该入口要求加密访问（HTTPS）。请使用 https:// 地址打开本页面")
		return
	}
	ipL, acctL := limiters()
	ip := clientIP(r)
	if !ipL.Allow("ip:" + ip) {
		writeJSON(w, http.StatusTooManyRequests, map[string]string{
			"error": "该网络地址尝试次数过多，请约 10 分钟后再试", "code": "rate-limited",
		})
		return
	}

	var req struct{ Account, OldPassword, NewPassword string }
	if err := readJSON(r, &req); err != nil {
		writeErr(w, http.StatusBadRequest, "请求体不合法")
		return
	}

	p := d.curProfile() // 拷贝快照：避免与连接向导热替换 Profile 的数据竞争
	svc := &selfservice.Service{
		Opts: ldapclient.Options{
			URL:                p.URL,
			StartTLS:           p.StartTLS,
			InsecureSkipVerify: p.InsecureSkipVerify,
		},
		LookupBindDN:   p.BindDN,
		LookupBindPass: p.BindPassword,
		BaseDN:         p.BaseDN,
		RL:             acctL,
	}
	err := svc.Change(req.Account, req.OldPassword, req.NewPassword)
	// R9.5 双维度限流：凭据失败也计入 IP 维度（否则单一 IP 可对无限多账号各猜 5 次）
	if errors.Is(err, selfservice.ErrBadCredentials) || errors.Is(err, selfservice.ErrAccountNotFound) {
		ipL.RecordFail("ip:" + ip)
	}

	result := "ok"
	status := http.StatusOK
	if err != nil {
		result = "fail"
		switch {
		case errors.Is(err, selfservice.ErrRateLimited):
			status = http.StatusTooManyRequests
		case errors.Is(err, selfservice.ErrBadCredentials), errors.Is(err, selfservice.ErrAccountNotFound):
			status = http.StatusUnauthorized
		default:
			status = http.StatusBadRequest
		}
	}
	// 审计：只记账号与结果，绝不记录任何密码值
	d.Audit.Log(audit.Event{
		Actor: "uid=" + req.Account, IP: ip, Op: "self-password-change",
		Result: result, Detail: auditDetail(err),
	})
	if err != nil {
		writeJSON(w, status, map[string]string{"error": err.Error(), "code": "policy"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"ok": "true"})
}

func auditDetail(err error) string {
	if err == nil {
		return ""
	}
	return err.Error() // 已是友好文案；不含密码值
}
