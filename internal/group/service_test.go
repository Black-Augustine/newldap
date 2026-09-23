package group

import (
	"strings"
	"testing"

	"newldap/internal/ldapclient"
	"newldap/internal/ldaptest"
)

const root = "dc=example,dc=cn"

func newService(t *testing.T) *Service {
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
	return New(c, root)
}

func TestScenarioCreateAndList(t *testing.T) {
	s := newService(t)

	// 权限组必须带首名成员
	if _, err := s.Create(CreateRequest{Name: "空权限组", Scenario: "权限组"}); err == nil {
		t.Fatal("无成员权限组应报错")
	}
	dn, err := s.Create(CreateRequest{
		Name: "代码管理员", Scenario: "权限组", Description: "Git 仓库管理权限",
		Members: []string{"uid=zhangwei,ou=tech," + root, "uid=lina,ou=tech," + root},
	})
	if err != nil {
		t.Fatal(err)
	}
	if dn != "cn=代码管理员,ou=groups,"+root {
		t.Errorf("dn = %q", dn)
	}

	// 登录组自动分配 gidNumber（种子 dev-login 不存在 → 5000）
	gdn, err := s.Create(CreateRequest{
		Name: "开发机登录", Scenario: "登录组", MemberUids: []string{"zhangwei", "lina"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(gdn, "cn=开发机登录") {
		t.Errorf("gdn = %q", gdn)
	}

	// 未知场景
	if _, err := s.Create(CreateRequest{Name: "x", Scenario: "神秘组"}); err == nil {
		t.Fatal("未知场景应报错")
	}

	// 列表：两类都在，计数正确
	list, err := s.List()
	if err != nil {
		t.Fatal(err)
	}
	var auth, posix *Group
	for i := range list {
		if list[i].Name == "代码管理员" {
			auth = &list[i]
		}
		if list[i].Name == "开发机登录" {
			posix = &list[i]
		}
	}
	if auth == nil || auth.MemberCount != 2 || auth.Scenario != "权限组" {
		t.Errorf("权限组列表项 = %+v", auth)
	}
	if posix == nil || posix.GidNumber != "5000" || posix.MemberCount != 2 {
		t.Errorf("登录组列表项 = %+v", posix)
	}
	// 种子组 vpn-access 也在
	var vpn *Group
	for i := range list {
		if list[i].Name == "vpn-access" {
			vpn = &list[i]
		}
	}
	if vpn == nil {
		t.Error("列表缺少种子组 vpn-access")
	}
}

func TestGIDAllocation(t *testing.T) {
	s := newService(t)
	// 占用 5000/5001
	for _, gid := range []int{5000, 5001} {
		if _, err := s.Create(CreateRequest{Name: "占位" + string(rune('A'+gid-5000)), Scenario: "登录组", GidNumber: gid}); err != nil {
			t.Fatal(err)
		}
	}
	next, err := s.NextGID()
	if err != nil {
		t.Fatal(err)
	}
	if next != 5002 {
		t.Errorf("NextGID = %d，应为 5002", next)
	}
}

func TestMemberUpdates(t *testing.T) {
	s := newService(t)
	dn := "cn=vpn-access,ou=groups," + root

	// 添加 + 移除成员（属性级）
	if err := s.UpdateMembers(dn, MemberChanges{Add: []string{"uid=lina,ou=tech," + root}}); err != nil {
		t.Fatal(err)
	}
	g, err := s.List()
	if err != nil {
		t.Fatal(err)
	}
	for _, grp := range g {
		if grp.Name == "vpn-access" && grp.MemberCount != 4 {
			t.Errorf("添加后成员数 = %d，应为 4（种子3+1）", grp.MemberCount)
		}
	}
	// 移除刚加的
	if err := s.UpdateMembers(dn, MemberChanges{Remove: []string{"uid=lina,ou=tech," + root}}); err != nil {
		t.Fatal(err)
	}
	// 空变更报错
	if err := s.UpdateMembers(dn, MemberChanges{}); err == nil {
		t.Fatal("空变更应报错")
	}
	// 登录组改 memberUid
	posix, err := s.Create(CreateRequest{Name: "临时登录组", Scenario: "登录组"})
	if err != nil {
		t.Fatal(err)
	}
	if err := s.UpdateMembers(posix, MemberChanges{Add: []string{"zhangwei"}}); err != nil {
		t.Fatal(err)
	}
	list, _ := s.List()
	for _, grp := range list {
		if grp.DN == posix && !contains(grp.Members, "zhangwei") {
			t.Errorf("登录组成员 = %v", grp.Members)
		}
	}
}

func TestUpdateDescription(t *testing.T) {
	s := newService(t)
	dn := "cn=vpn-access,ou=groups," + root
	if err := s.UpdateDescription(dn, "新的描述"); err != nil {
		t.Fatal(err)
	}
	list, _ := s.List()
	for _, g := range list {
		if g.DN == dn && g.Description != "新的描述" {
			t.Errorf("描述 = %q", g.Description)
		}
	}
}

func contains(list []string, v string) bool {
	for _, x := range list {
		if x == v {
			return true
		}
	}
	return false
}
