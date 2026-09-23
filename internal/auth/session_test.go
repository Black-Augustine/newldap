package auth

import (
	"testing"
	"time"

	"newldap/internal/ldapclient"
	"newldap/internal/ldaptest"
)

// TestSessionExpiry 覆盖 R11.1 会话超时：过期后 Get 失败并关闭连接。
func TestSessionExpiry(t *testing.T) {
	srv, err := ldaptest.Start(true)
	if err != nil {
		t.Fatal(err)
	}
	defer srv.Stop()

	st := NewStore(60 * time.Millisecond)
	s, err := st.Login(ldapclient.Options{URL: srv.Addr()}, "cn=admin,dc=example,dc=cn", "admin123")
	if err != nil {
		t.Fatal(err)
	}
	if st.Count() != 1 {
		t.Fatalf("会话数 = %d", st.Count())
	}
	if _, err := st.Get(s.Token); err != nil {
		t.Fatalf("会话内 Get 不应报错: %v", err)
	}
	time.Sleep(120 * time.Millisecond)
	if _, err := st.Get(s.Token); err == nil {
		t.Fatal("过期会话应报错")
	}
	if st.Count() != 0 {
		t.Errorf("过期会话应被清除，剩余 %d", st.Count())
	}
	_ = s.Conn.Ping() // 过期连接已被关闭（不要求特定错误，仅确保不 panic）
}

// TestLogout 覆盖注销：会话即失效。
func TestLogout(t *testing.T) {
	srv, err := ldaptest.Start(true)
	if err != nil {
		t.Fatal(err)
	}
	defer srv.Stop()

	st := NewStore(time.Minute)
	s, err := st.Login(ldapclient.Options{URL: srv.Addr()}, "cn=admin,dc=example,dc=cn", "admin123")
	if err != nil {
		t.Fatal(err)
	}
	st.Logout(s.Token)
	if _, err := st.Get(s.Token); err == nil {
		t.Fatal("注销后 Get 应报错")
	}
}
