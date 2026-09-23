package selfservice

import (
	"errors"
	"strings"
	"testing"
	"time"

	"newldap/internal/ldapclient"
	"newldap/internal/ldaptest"
)

const root = "dc=example,dc=cn"

func newService(t *testing.T) (*Service, *ldaptest.Server) {
	t.Helper()
	srv, err := ldaptest.Start(true)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(srv.Stop)
	return &Service{
		Opts:   ldapclient.Options{URL: srv.Addr()},
		BaseDN: root,
		RL:     NewRateLimiter(3, 10*time.Minute),
	}, srv
}

func TestChangePasswordHappyPath(t *testing.T) {
	s, srv := newService(t)
	if err := s.Change("zhangwei", "Passw0rd!", "NewStrong#88x"); err != nil {
		t.Fatalf("改密失败: %v", err)
	}
	// 旧密码失效、新密码可 bind
	c, _ := ldapclient.Dial(ldapclient.Options{URL: srv.Addr()})
	defer c.Close()
	if err := c.Bind("uid=zhangwei,ou=tech,"+root, "Passw0rd!"); err == nil {
		t.Fatal("旧密码仍然有效！")
	}
	if err := c.Bind("uid=zhangwei,ou=tech,"+root, "NewStrong#88x"); err != nil {
		t.Fatalf("新密码 bind 失败: %v", err)
	}
}

func TestBadOldPasswordAndAntiEnum(t *testing.T) {
	s, _ := newService(t)
	err := s.Change("zhangwei", "错误旧密码", "NewStrong#88x")
	if !errors.Is(err, ErrBadCredentials) {
		t.Fatalf("应报旧密码错误，得到: %v", err)
	}
	// 不存在的账号与密码错误返回相同文案（防枚举）
	err2 := s.Change("no_such_user_xyz", "随便", "NewStrong#88x")
	if !errors.Is(err2, ErrAccountNotFound) || err2.Error() != err.Error() {
		t.Errorf("防枚举要求两种失败文案一致: %q vs %q", err2.Error(), err.Error())
	}
}

func TestWeakNewPassword(t *testing.T) {
	s, _ := newService(t)
	if err := s.Change("zhangwei", "Passw0rd!", "short"); err == nil || !strings.Contains(err.Error(), "8 位") {
		t.Fatalf("弱密码应被拒并提示 8 位: %v", err)
	}
	if err := s.Change("zhangwei", "Passw0rd!", "Passw0rd!"); err == nil || !strings.Contains(err.Error(), "相同") {
		t.Fatalf("新旧相同应被拒: %v", err)
	}
}

func TestRateLimitPerAccount(t *testing.T) {
	s, _ := newService(t)
	// 连续 3 次旧密码错误（max=3）→ 第 4 次被限流（即使这次密码是对的）
	for i := 0; i < 3; i++ {
		_ = s.Change("lina", "错的", "NewStrong#88x")
	}
	err := s.Change("lina", "Passw0rd!", "NewStrong#88x")
	if !errors.Is(err, ErrRateLimited) {
		t.Fatalf("第 4 次应被限流，得到: %v", err)
	}
	// 其他账号不受影响
	if err := s.Change("chenjing", "Passw0rd!", "Another#77q"); err != nil {
		t.Fatalf("无关账号被误伤: %v", err)
	}
	// 成功后清零计数
	s.RL.RecordSuccess("acct:lina")
	if err := s.Change("lina", "Passw0rd!", "NewStrong#88x"); err != nil {
		t.Fatalf("清零后应成功: %v", err)
	}
}

func TestTranslatePolicy(t *testing.T) {
	cases := []struct {
		code int
		diag string
		want string
	}{
		{19, "password in history", "重复"},
		{19, "Password fails quality checking", "强度"},
		{19, "password too short", "长度"},
		{19, "password age too young", "太近"},
		{53, "account locked", "锁定"},
		{0, "", ""}, // 非策略 → nil
	}
	for _, c := range cases {
		pe := translatePolicy(c.code, c.diag)
		if c.want == "" {
			if pe != nil {
				t.Errorf("[%d %q] 不应翻译", c.code, c.diag)
			}
			continue
		}
		if pe == nil || !strings.Contains(pe.Friendly, c.want) {
			t.Errorf("[%d %q] 翻译应含 %q: %+v", c.code, c.diag, c.want, pe)
		}
	}
}
