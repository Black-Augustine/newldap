package people

import (
	"strings"
	"testing"

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
	c, err := ldapclient.Dial(ldapclient.Options{URL: srv.Addr()})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(c.Close)
	if err := c.Bind("cn=admin,"+root, "admin123"); err != nil {
		t.Fatal(err)
	}
	return New(c, root), srv
}

func TestCreateAndDetail(t *testing.T) {
	s, srv := newService(t)

	// 缺必填
	if _, _, err := s.Create(CreateRequest{CN: "无账号"}); err == nil {
		t.Fatal("缺 uid 应报错")
	}
	// 非法字符
	if _, _, err := s.Create(CreateRequest{CN: "坏", UID: "a,b"}); err == nil {
		t.Fatal("uid 含逗号应报错")
	}

	dn, pw, err := s.Create(CreateRequest{
		CN: "许倩", UID: "xuqian", OU: "ou=tech," + root,
		Mail: "xuqian@example.cn", Mobile: "139-0000-0001", EmpNo: "E1021", Title: "算法工程师",
	})
	if err != nil {
		t.Fatal(err)
	}
	if dn != "uid=xuqian,ou=tech,"+root {
		t.Errorf("dn = %q", dn)
	}
	if len(pw) != 12 {
		t.Errorf("初始密码长度 = %d", len(pw))
	}
	// 初始密码可 bind（真实 bind 验证）
	c2, err := ldapclient.Dial(ldapclient.Options{URL: srv.Addr()})
	if err != nil {
		t.Fatal(err)
	}
	defer c2.Close()
	if err := c2.Bind(dn, pw); err != nil {
		t.Fatalf("初始密码 bind 失败: %v", err)
	}
	// 重复 uid 被拦
	if _, _, err := s.Create(CreateRequest{CN: "许倩2", UID: "xuqian", OU: "ou=tech," + root}); err == nil {
		t.Fatal("重复 uid 应被拦截")
	}

	// 详情 + 所属组反查（种子 vpn-access 含张伟）
	d, err := s.Detail("uid=zhangwei,ou=tech," + root)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, g := range d.Groups {
		if g.Name == "vpn-access" && g.Type == "权限组" {
			found = true
		}
	}
	if !found {
		t.Errorf("张伟应属于 vpn-access: %+v", d.Groups)
	}
	// 详情不得返回 userPassword（与 API 搜索同规）
	if _, ok := d.Attrs["userPassword"]; ok {
		t.Error("详情返回了 userPassword")
	}
}

func TestResetDisableEnable(t *testing.T) {
	s, srv := newService(t)
	dn := "uid=lina,ou=tech," + root

	// 指定密码重置
	if _, err := s.ResetPassword(dn, "NewPass123!"); err != nil {
		t.Fatal(err)
	}
	c2, _ := ldapclient.Dial(ldapclient.Options{URL: srv.Addr()})
	defer c2.Close()
	if err := c2.Bind(dn, "NewPass123!"); err != nil {
		t.Fatalf("重置后新密码 bind 失败: %v", err)
	}
	// 随机重置
	pw, err := s.ResetPassword(dn, "")
	if err != nil || len(pw) != 12 {
		t.Fatalf("随机重置: %v %q", err, pw)
	}

	// 禁用后 bind 失败（替身按字符串比对，前缀必不匹配）
	if err := s.Disable(dn); err != nil {
		t.Fatal(err)
	}
	if err := c2.Bind(dn, pw); err == nil {
		t.Fatal("禁用后仍能 bind！")
	} else if !strings.Contains(err.Error(), "49") && !strings.Contains(err.Error(), "invalid") && !strings.Contains(err.Error(), "账号或密码") {
		t.Logf("禁用后的 bind 错误（信息）: %v", err)
	}
	// 重复禁用报错
	if err := s.Disable(dn); err == nil {
		t.Fatal("重复禁用应报错")
	}
	// 启用后恢复原密码
	if err := s.Enable(dn); err != nil {
		t.Fatal(err)
	}
	if err := c2.Bind(dn, pw); err != nil {
		t.Fatalf("启用后原密码应恢复: %v", err)
	}
}

func TestSetDept(t *testing.T) {
	s, srv := newService(t)
	dn := "uid=chenjing,ou=tech," + root
	newDN, err := s.SetDept(dn, "ou=product,"+root)
	if err != nil {
		t.Fatal(err)
	}
	if newDN != "uid=chenjing,ou=product,"+root {
		t.Errorf("newDN = %q", newDN)
	}
	e := srv.Entry(newDN)
	if e == nil {
		t.Fatal("调动后新位置无条目")
	}
	if len(e.Get("mail")) == 0 {
		t.Error("调动后属性丢失")
	}
}
