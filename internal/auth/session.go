// Package auth 实现管理台会话：登录 = 用输入的 DN/密码对目录做一次真实
// bind；bind 成功后保留该连接作为会话连接，后续所有操作经由它执行，
// 因此会话权限 === 服务端 ACL（R11.4 的实现基础）。
package auth

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"sync"
	"time"

	"newldap/internal/ldapclient"
)

// Session 是一个登录会话，持有独立的已 bind 连接。
type Session struct {
	Token   string
	BindDN  string
	Conn    *ldapclient.Client
	Expires time.Time
}

// Store 管理内存会话表。
type Store struct {
	mu       sync.Mutex
	sessions map[string]*Session
	ttl      time.Duration
}

func NewStore(ttl time.Duration) *Store {
	if ttl <= 0 {
		ttl = 30 * time.Minute
	}
	return &Store{sessions: map[string]*Session{}, ttl: ttl}
}

// Login 用 dn/password 对目录 bind；成功则建立会话。
// 返回的错误原样透传（含 LDAP invalidCredentials），供 API 层翻译文案。
func (st *Store) Login(opts ldapclient.Options, dn, password string) (*Session, error) {
	if dn == "" || password == "" {
		return nil, errors.New("请输入账号与密码")
	}
	c, err := ldapclient.Dial(opts)
	if err != nil {
		return nil, err
	}
	if err := c.Bind(dn, password); err != nil {
		c.Close()
		return nil, err
	}
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		c.Close()
		return nil, err
	}
	s := &Session{
		Token:   hex.EncodeToString(buf),
		BindDN:  dn,
		Conn:    c,
		Expires: time.Now().Add(st.ttl),
	}
	st.mu.Lock()
	st.sessions[s.Token] = s
	st.mu.Unlock()
	return s, nil
}

// Get 取会话（滑动续期）；不存在或已过期返回错误。
func (st *Store) Get(token string) (*Session, error) {
	st.mu.Lock()
	defer st.mu.Unlock()
	s, ok := st.sessions[token]
	if !ok {
		return nil, errors.New("会话不存在或已过期")
	}
	if time.Now().After(s.Expires) {
		s.Conn.Close()
		delete(st.sessions, token)
		return nil, errors.New("会话不存在或已过期")
	}
	s.Expires = time.Now().Add(st.ttl)
	return s, nil
}

// Logout 注销会话并关闭连接。
func (st *Store) Logout(token string) {
	st.mu.Lock()
	defer st.mu.Unlock()
	if s, ok := st.sessions[token]; ok {
		s.Conn.Close()
		delete(st.sessions, token)
	}
}

// Count 返回活跃会话数（测试/监控用）。
func (st *Store) Count() int {
	st.mu.Lock()
	defer st.mu.Unlock()
	return len(st.sessions)
}
