// Package group 用户组领域层（R4.1~R4.3）：
// 场景化建组（权限组 groupOfNames / 登录组 posixGroup + gidNumber 自动分配）、
// 成员维护（属性级修改 member/memberUid）、列表与计数。
package group

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	ldap "github.com/go-ldap/ldap/v3"

	"newldap/internal/ldapclient"
)

// GID 范围（可被环境覆盖前先固定常量；v1.x 做成配置）
const (
	gidMin = 5000
	gidMax = 5999
)

type Service struct {
	C      *ldapclient.Client
	BaseDN string // 组搜索范围（通常是根）
}

func New(c *ldapclient.Client, baseDN string) *Service { return &Service{C: c, BaseDN: baseDN} }

// Group 列表项。
type Group struct {
	DN          string   `json:"dn"`
	Name        string   `json:"name"`
	Scenario    string   `json:"scenario"` // 权限组 | 登录组
	Classes     []string `json:"objectClass"`
	Description string   `json:"description,omitempty"`
	GidNumber   string   `json:"gidNumber,omitempty"`
	Members     []string `json:"members"` // 权限组: member(DN)；登录组: memberUid
	MemberCount int      `json:"memberCount"`
}

// List 列出全部用户组（两类合并，R4 列表页）。
func (s *Service) List() ([]Group, error) {
	var out []Group
	hits, err := s.C.PagedSearch(s.BaseDN, ldap.ScopeWholeSubtree,
		"(|(objectClass=groupOfNames)(objectClass=posixGroup))",
		[]string{"cn", "description", "gidNumber", "member", "memberUid", "objectClass"}, 500)
	if err != nil {
		return nil, err
	}
	for _, e := range hits {
		g := Group{
			DN: e.DN, Name: e.GetAttributeValue("cn"),
			Description: e.GetAttributeValue("description"),
			GidNumber:   e.GetAttributeValue("gidNumber"),
			Classes:     e.GetAttributeValues("objectClass"),
		}
		if hasClass(g.Classes, "posixGroup") {
			g.Scenario = "登录组"
			g.Members = e.GetAttributeValues("memberUid")
		} else {
			g.Scenario = "权限组"
			g.Members = e.GetAttributeValues("member")
		}
		g.MemberCount = len(g.Members)
		out = append(out, g)
	}
	return out, nil
}

func hasClass(classes []string, want string) bool {
	for _, c := range classes {
		if strings.EqualFold(c, want) {
			return true
		}
	}
	return false
}

// CreateRequest 场景化建组（R4.1）：用白话选场景，不直接暴露 objectClass。
type CreateRequest struct {
	Name        string   `json:"name"`     // 组名（cn，必填）
	Scenario    string   `json:"scenario"` // 权限组 | 登录组
	Description string   `json:"description"`
	GroupOU     string   `json:"groupOU"`    // 建组位置（默认 ou=groups,<base>）
	Members     []string `json:"members"`    // 权限组：成员 DN（至少 1，schema MUST）
	MemberUids  []string `json:"memberUids"` // 登录组：成员 uid（可空）
	GidNumber   int      `json:"gidNumber"`  // 登录组：缺省自动分配
}

// Create 建组。权限组要求至少一名成员（groupOfNames 的 MUST member），
// 界面应引导"先选首名成员"；登录组自动从 5000-5999 分配空闲 gidNumber。
func (s *Service) Create(req CreateRequest) (string, error) {
	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" {
		return "", errors.New("组名为必填")
	}
	ou := strings.TrimSpace(req.GroupOU)
	if ou == "" {
		ou = "ou=groups," + s.BaseDN
	}
	dn := "cn=" + req.Name + "," + ou
	attrs := map[string][]string{"cn": {req.Name}}
	if req.Description != "" {
		attrs["description"] = []string{req.Description}
	}
	switch req.Scenario {
	case "权限组":
		if len(req.Members) == 0 {
			return "", errors.New("权限组至少需要一名成员（目录要求 member 必填），建组后再继续添加")
		}
		attrs["objectClass"] = []string{"top", "groupOfNames"}
		attrs["member"] = req.Members
	case "登录组":
		gid := req.GidNumber
		if gid == 0 {
			var err error
			gid, err = s.NextGID()
			if err != nil {
				return "", err
			}
		}
		attrs["objectClass"] = []string{"top", "posixGroup"}
		attrs["gidNumber"] = []string{strconv.Itoa(gid)}
		if len(req.MemberUids) > 0 {
			attrs["memberUid"] = req.MemberUids
		}
	default:
		return "", fmt.Errorf("未知场景 %q（应为 权限组 或 登录组）", req.Scenario)
	}
	if err := s.C.AddEntry(dn, attrs); err != nil {
		return "", err
	}
	return dn, nil
}

// NextGID 扫描现有 posixGroup 的 gidNumber，返回范围内第一个空闲值。
func (s *Service) NextGID() (int, error) {
	used := map[int]bool{}
	hits, err := s.C.PagedSearch(s.BaseDN, ldap.ScopeWholeSubtree,
		"(objectClass=posixGroup)", []string{"gidNumber"}, 1000)
	if err != nil {
		return 0, err
	}
	for _, e := range hits {
		for _, v := range e.GetAttributeValues("gidNumber") {
			if n, err := strconv.Atoi(strings.TrimSpace(v)); err == nil {
				used[n] = true
			}
		}
	}
	for gid := gidMin; gid <= gidMax; gid++ {
		if !used[gid] {
			return gid, nil
		}
	}
	return 0, fmt.Errorf("gidNumber 范围 %d-%d 已耗尽", gidMin, gidMax)
}

// MemberChanges 成员变更（R4.2）。
type MemberChanges struct {
	Add    []string `json:"add"`
	Remove []string `json:"remove"`
}

// UpdateMembers 属性级修改成员：权限组改 member(DN)，登录组改 memberUid。
func (s *Service) UpdateMembers(dn string, ch MemberChanges) error {
	e, err := s.C.ReadEntry(dn, []string{"objectClass"})
	if err != nil {
		return err
	}
	classes := e.GetAttributeValues("objectClass")
	attr := "member"
	if hasClass(classes, "posixGroup") {
		attr = "memberUid"
	}
	var changes []ldap.Change
	for _, m := range ch.Add {
		changes = append(changes, ldap.Change{Operation: ldap.AddAttribute,
			Modification: ldap.PartialAttribute{Type: attr, Vals: []string{m}}})
	}
	for _, m := range ch.Remove {
		changes = append(changes, ldap.Change{Operation: ldap.DeleteAttribute,
			Modification: ldap.PartialAttribute{Type: attr, Vals: []string{m}}})
	}
	if len(changes) == 0 {
		return errors.New("没有成员变更")
	}
	// groupOfNames 的 member 是 MUST：删除最后一名成员会被服务端拒绝（合理行为）
	return s.C.ModifyAttributes(dn, changes)
}

// UpdateDescription 修改组描述。
func (s *Service) UpdateDescription(dn, desc string) error {
	return s.C.ModifyAttributes(dn, []ldap.Change{
		{Operation: ldap.ReplaceAttribute, Modification: ldap.PartialAttribute{Type: "description", Vals: []string{desc}}},
	})
}
