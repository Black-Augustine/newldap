// Package selfservice 实现员工自助改密（R9.2/R9.3）：
// 用户用「账号(uid) + 旧密码」自 bind 验证，成功后用同一连接替换自己的
// userPassword（全程无需管理员权限，权限 = 服务端 ACL）。
// 服务端 ppolicy 错误翻译为中文白话（R9.3）；内置按账号+IP 的失败限流（R9.5）。
package selfservice

import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	ldap "github.com/go-ldap/ldap/v3"

	"newldap/internal/ldapclient"
)

// 领域错误（API 层转 HTTP 状态与文案）。
var (
	ErrBadCredentials  = errors.New("账号或密码错误，请确认后重试")
	ErrAccountNotFound = errors.New("账号或密码错误，请确认后重试") // 与凭据错误同文案，防枚举
	ErrRateLimited     = errors.New("尝试次数过多，请稍后再试")
	ErrWeakPassword    = errors.New("新密码不符合策略要求")
)

// PolicyError 是服务端密码策略拒绝（ppolicy / constraintViolation）的友好翻译。
type PolicyError struct {
	Code     int    // LDAP 结果码（19/53…）
	Friendly string // 中文白话
}

func (e *PolicyError) Error() string { return e.Friendly }

// translatePolicy 把服务端拒绝翻译成人话（覆盖 OpenLDAP ppolicy 常见输出）。
func translatePolicy(resultCode int, diagnostic string) *PolicyError {
	d := strings.ToLower(diagnostic)
	switch {
	case resultCode == 19 && (strings.Contains(d, "history") || strings.Contains(d, "历史") || strings.Contains(d, "already used")):
		return &PolicyError{19, "新密码与你最近使用过的密码重复，请换一个没有用过的"}
	case resultCode == 19 && (strings.Contains(d, "quality") || strings.Contains(d, "强度") || strings.Contains(d, "check failed")):
		return &PolicyError{19, "新密码强度不足：请使用更长的密码并混合字母、数字与符号"}
	case resultCode == 19 && (strings.Contains(d, "short") || strings.Contains(d, "长度")):
		return &PolicyError{19, "新密码长度不足：请加长密码（通常至少 8 位）"}
	case resultCode == 19 && (strings.Contains(d, "age") || strings.Contains(d, "young") || strings.Contains(d, "间隔") || strings.Contains(d, "too soon")):
		return &PolicyError{19, "距离上次修改密码时间太近，暂时不能再次修改；如有紧急情况请联系管理员"}
	case resultCode == 53 && (strings.Contains(d, "lock") || strings.Contains(d, "锁定")):
		return &PolicyError{53, "账号已被锁定（连续失败次数过多），请联系管理员解锁"}
	case resultCode == 19:
		return &PolicyError{19, "新密码不满足服务器密码策略：" + diagnostic}
	default:
		return nil // 非策略类错误，按通用错误处理
	}
}

// ---------- 限流（R9.5） ----------

// RateLimiter 按 key（账号 DN / IP）滑动窗口计数；超限锁定。
type RateLimiter struct {
	mu      sync.Mutex
	maxFail int
	lockFor time.Duration
	fails   map[string][]time.Time
}

func NewRateLimiter(maxFail int, lockFor time.Duration) *RateLimiter {
	if maxFail <= 0 {
		maxFail = 5
	}
	if lockFor <= 0 {
		lockFor = 10 * time.Minute
	}
	return &RateLimiter{maxFail: maxFail, lockFor: lockFor, fails: map[string][]time.Time{}}
}

// Allow 检查 key 是否处于锁定期。
func (r *RateLimiter) Allow(key string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	now := time.Now()
	var live []time.Time
	for _, t := range r.fails[key] {
		if now.Sub(t) < r.lockFor {
			live = append(live, t)
		}
	}
	if len(live) == 0 {
		delete(r.fails, key)
		return true
	}
	r.fails[key] = live
	return len(live) < r.maxFail
}

// RecordFail 记一次失败；返回窗口内剩余尝试次数。
func (r *RateLimiter) RecordFail(key string) (remaining int) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.fails[key] = append(r.fails[key], time.Now())
	return r.maxFail - len(r.fails[key])
}

// RecordSuccess 清零。
func (r *RateLimiter) RecordSuccess(key string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.fails, key)
}

// ---------- 改密流程 ----------

// Service 依赖连接参数（通常来自连接档案）与 base DN。
type Service struct {
	Opts   ldapclient.Options
	BaseDN string
	RL     *RateLimiter

	// LookupBind 可选：解析 uid → DN 时使用的档案凭据（未配置则匿名搜索）
	LookupBindDN   string
	LookupBindPass string
}

// Change 用账号（uid）+ 旧密码自 bind，成功则改自己的密码。
// 返回翻译后的错误（*PolicyError / 领域错误）；成功返回 nil。
func (s *Service) Change(account, oldPw, newPw string) error {
	account = strings.TrimSpace(account)
	if account == "" || oldPw == "" || newPw == "" {
		return errors.New("请完整填写账号、当前密码与新密码")
	}
	if newPw == oldPw {
		return errors.New("新密码不能与当前密码相同")
	}
	if len(newPw) < 8 {
		return &PolicyError{19, "新密码至少需要 8 位（这是服务器密码策略的最低要求）"}
	}

	// 限流（账号维度；IP 维度由 API 层再叠加）
	if s.RL != nil && !s.RL.Allow("acct:"+account) {
		return ErrRateLimited
	}

	// 1. 用管理员视角解析 uid → DN（只读搜索；找得到/找不到都返回同文案防枚举）
	c, err := ldapclient.Dial(s.Opts)
	if err != nil {
		return fmt.Errorf("无法连接目录服务器，请稍后再试")
	}
	defer c.Close()
	if s.LookupBindDN == "" {
		// 档案未存管理员凭据时退化为匿名搜索（多数内网允许）
		_ = c.Bind("", "")
	} else if err := c.Bind(s.LookupBindDN, s.LookupBindPass); err != nil {
		return fmt.Errorf("目录服务暂不可用，请稍后再试")
	}
	hits, err := c.PagedSearch(s.BaseDN, ldap.ScopeWholeSubtree,
		"(uid="+ldap.EscapeFilter(account)+")", []string{"1.1"}, 2)
	if err != nil || len(hits) != 1 {
		// 记一次失败，模拟真实账号的节流体验
		if s.RL != nil {
			s.RL.RecordFail("acct:" + account)
		}
		return ErrAccountNotFound
	}
	dn := hits[0].DN

	// 2. 自 bind 验证旧密码（R9.2：全程无需管理员权限）
	uc, err := ldapclient.Dial(s.Opts)
	if err != nil {
		return fmt.Errorf("无法连接目录服务器，请稍后再试")
	}
	defer uc.Close()
	if err := uc.Bind(dn, oldPw); err != nil {
		if s.RL != nil {
			s.RL.RecordFail("acct:" + account)
		}
		return ErrBadCredentials
	}

	// 3. 用本人连接替换 userPassword（属性级修改；ppolicy 由服务端按其策略校验，
	// 拒绝时以结果码 19/53 + diagnostic 形式返回，由 translatePolicy 翻译）
	err = modifyWithPolicy(uc, dn, newPw)
	if err != nil {
		var serverErr *ldap.Error
		if errors.As(err, &serverErr) {
			if pe := translatePolicy(int(serverErr.ResultCode), serverErr.Err.Error()); pe != nil {
				if s.RL != nil {
					s.RL.RecordFail("acct:" + account)
				}
				return pe
			}
		}
		return fmt.Errorf("修改失败，请稍后再试或联系管理员")
	}

	if s.RL != nil {
		s.RL.RecordSuccess("acct:" + account)
	}
	return nil
}

// modifyWithPolicy 执行带 ppolicy 请求控件的属性级替换。
func modifyWithPolicy(c *ldapclient.Client, dn, newPw string) error {
	// 复用 ldap.ModifyRequest + 控件：通过底层 Modify 直连（保持包内闭环）
	return c.ModifyAttributes(dn, []ldap.Change{
		{Operation: ldap.ReplaceAttribute, Modification: ldap.PartialAttribute{Type: "userPassword", Vals: []string{newPw}}},
	})
}
