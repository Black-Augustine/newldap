package api

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	ldap "github.com/go-ldap/ldap/v3"
)

func TestWriteLdapErr(t *testing.T) {
	cases := []struct {
		name   string
		err    error
		action string
		status int
		want   string
	}{
		{"ACL 拒绝 → 403", &ldap.Error{ResultCode: ldap.LDAPResultInsufficientAccessRights, Err: errors.New("no write access")}, "修改条目", http.StatusForbidden, "无权修改条目"},
		{"需强认证 → 403", &ldap.Error{ResultCode: 8, Err: errors.New("")}, "读取条目", http.StatusForbidden, "无权"},
		{"不存在 → 404", &ldap.Error{ResultCode: ldap.LDAPResultNoSuchObject, Err: errors.New("no such object")}, "删除条目", http.StatusNotFound, "不存在"},
		{"已存在 → 409", &ldap.Error{ResultCode: ldap.LDAPResultEntryAlreadyExists, Err: errors.New("entry exists")}, "新建条目", http.StatusConflict, "已存在"},
		{"非叶子 → 409", &ldap.Error{ResultCode: ldap.LDAPResultNotAllowedOnNonLeaf, Err: errors.New("not allowed")}, "删除条目", http.StatusConflict, "子条目"},
		{"schema 违例 → 400", &ldap.Error{ResultCode: ldap.LDAPResultObjectClassViolation, Err: errors.New("missing attribute")}, "新建条目", http.StatusBadRequest, "必填属性"},
		{"其他 → 502 原文", &ldap.Error{ResultCode: 80, Err: errors.New("other")}, "操作", http.StatusBadGateway, "other"},
		{"非 LDAP 错误 → 未处理", errors.New("业务错误"), "操作", 0, ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			handled := writeLdapErr(w, c.err, c.action)
			if c.status == 0 {
				if handled {
					t.Fatal("非 LDAP 错误不应被处理")
				}
				return
			}
			if !handled {
				t.Fatal("应被处理")
			}
			if w.Code != c.status {
				t.Errorf("状态 = %d，应为 %d", w.Code, c.status)
			}
			body := w.Body.String()
			if !containsStr(body, c.want) {
				t.Errorf("响应 %q 应含 %q", body, c.want)
			}
		})
	}
}

func containsStr(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
