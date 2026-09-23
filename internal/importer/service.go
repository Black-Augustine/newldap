// Package importer 实现 Excel 导入导出的计划构建与执行（R7.2~R7.6，M3）。
// 计划：部门按"全路径"匹配目录已有 OU；人员按工号（其次 uid）匹配已有人员
// → create/update/error。执行：部门自底向上创建；人员创建经 people 领域层
// （收集初始密码供管理员转交）；更新做属性级 diff + 部门移动（modrdn）。
package importer

import (
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"strings"

	ldap "github.com/go-ldap/ldap/v3"

	"newldap/internal/directory"
	"newldap/internal/excel"
	"newldap/internal/ldapclient"
	"newldap/internal/people"
)

// Item 与前端预览表对应的计划项。
type Item struct {
	Sheet  string `json:"sheet"`
	Row    int    `json:"row"`
	Kind   string `json:"kind"`   // dept | person
	Action string `json:"action"` // create | update | error
	Name   string `json:"name"`
	Key    string `json:"key"`
	Detail string `json:"detail"`

	// 执行期数据（随计划 JSON 原样往返：parse → 前端预览确认 → execute）
	DeptPath  string           `json:"deptPath,omitempty"`
	PersonRow *excel.PersonRow `json:"person,omitempty"`
}

// Summary 汇总计数。
type Summary struct {
	Create int `json:"create"`
	Update int `json:"update"`
	Dept   int `json:"dept"`
	Error  int `json:"error"`
}

type existingPerson struct {
	DN, UID, EmpNo, Mail, Mobile, Title, CN string
}

// Service 依赖一个已认证连接与根 DN。
type Service struct {
	C      *ldapclient.Client
	BaseDN string
}

// ParsePlan 解析上传的 Excel 并对账目录，生成执行计划（不写目录）。
func (s *Service) ParsePlan(fh *multipart.FileHeader) (file string, items []Item, summary Summary, err error) {
	f, err := fh.Open()
	if err != nil {
		return "", nil, summary, err
	}
	defer f.Close()
	tmp, err := os.CreateTemp("", "newldap-import-*.xlsx")
	if err != nil {
		return "", nil, summary, err
	}
	defer os.Remove(tmp.Name())
	if _, err := io.Copy(tmp, f); err != nil {
		return "", nil, summary, err
	}
	tmp.Close()

	wb, err := excel.ParseFile(tmp.Name())
	if err != nil {
		return "", nil, summary, err
	}
	file = fh.Filename

	// 目录现状
	ouPaths, err := s.existingOUPaths()
	if err != nil {
		return "", nil, summary, err
	}
	persons, err := s.existingPersons()
	if err != nil {
		return "", nil, summary, err
	}
	byEmp, byUID := map[string]*existingPerson{}, map[string]*existingPerson{}
	for i := range persons {
		if persons[i].EmpNo != "" {
			byEmp[persons[i].EmpNo] = &persons[i]
		}
		byUID[strings.ToLower(persons[i].UID)] = &persons[i]
	}

	// 部门计划：路径不存在于目录 → create
	for _, d := range wb.Depts {
		if _, ok := ouPaths[d.Path]; ok {
			continue // 已存在，无需处理（属性不覆盖）
		}
		items = append(items, Item{
			Sheet: excel.SheetDept, Row: d.Row, Kind: "dept", Action: "create",
			Name: d.Name, Key: d.Path, Detail: "新建部门 " + d.Path, DeptPath: d.Path,
		})
		ouPaths[d.Path] = "planned"
	}

	// 人员计划
	for _, p := range wb.Persons {
		it := Item{Sheet: excel.SheetPerson, Row: p.Row, Kind: "person", Name: p.Name, Key: p.EmpNo, PersonRow: &p}
		// 部门路径必须存在于目录或本次计划
		if _, ok := ouPaths[p.DeptPath]; !ok {
			it.Action = "error"
			it.Detail = fmt.Sprintf("部门路径不存在：%s（也不在本文件的部门表中）", p.DeptPath)
			items = append(items, it)
			continue
		}
		var match *existingPerson
		if p.EmpNo != "" {
			match = byEmp[p.EmpNo]
		}
		if match == nil {
			match = byUID[strings.ToLower(p.UID)]
		}
		if match != nil {
			it.Action = "update"
			it.Detail = "更新已有人员 " + match.UID
		} else {
			it.Action = "create"
			it.Detail = "新建人员 uid=" + p.UID
		}
		items = append(items, it)
	}

	// 解析层错误行
	for _, e := range wb.Errors {
		items = append(items, Item{Sheet: e.Sheet, Row: e.Row, Kind: "person", Action: "error", Name: "—", Detail: e.Err})
	}

	for _, it := range items {
		switch {
		case it.Action == "error":
			summary.Error++
		case it.Kind == "dept":
			summary.Dept++
		case it.Action == "create":
			summary.Create++
		case it.Action == "update":
			summary.Update++
		}
	}
	return file, items, summary, nil
}

// existingOUPaths 返回 部门全路径 → DN 映射（如 技术研发部/平台组 → ou=平台组,ou=技术研发部,base）。
func (s *Service) existingOUPaths() (map[string]string, error) {
	entries, err := s.C.PagedSearch(s.BaseDN, ldap.ScopeWholeSubtree,
		"(objectClass=organizationalUnit)", []string{"ou"}, 2000)
	if err != nil {
		return nil, err
	}
	out := map[string]string{}
	for _, e := range entries {
		dn := e.DN
		parts := strings.Split(strings.ToLower(dn), ",")
		var names []string
		for _, p := range parts {
			p = strings.TrimSpace(p)
			if strings.HasPrefix(p, "ou=") {
				names = append([]string{e.GetAttributeValue("ou")}, names...) // 简化：取每级 ou 值（同名场景少）
			}
		}
		if len(names) > 0 {
			out[strings.Join(names, "/")] = dn
		}
	}
	return out, nil
}

func (s *Service) existingPersons() ([]existingPerson, error) {
	entries, err := s.C.PagedSearch(s.BaseDN, ldap.ScopeWholeSubtree,
		"(objectClass=inetOrgPerson)",
		[]string{"cn", "uid", "employeeNumber", "mail", "mobile", "title"}, 5000)
	if err != nil {
		return nil, err
	}
	out := make([]existingPerson, 0, len(entries))
	for _, e := range entries {
		out = append(out, existingPerson{
			DN: e.DN, CN: e.GetAttributeValue("cn"), UID: e.GetAttributeValue("uid"),
			EmpNo: e.GetAttributeValue("employeeNumber"), Mail: e.GetAttributeValue("mail"),
			Mobile: e.GetAttributeValue("mobile"), Title: e.GetAttributeValue("title"),
		})
	}
	return out, nil
}

// ResultItem 执行结果行。
type ResultItem struct {
	Sheet  string `json:"sheet"`
	Row    int    `json:"row"`
	Name   string `json:"name"`
	Action string `json:"action"`
	Status string `json:"status"` // ok | fail | skip
	Detail string `json:"detail"`
}

// InitialPassword 新建账号的初始密码（仅本次响应携带，供管理员转交）。
type InitialPassword struct {
	DN       string `json:"dn"`
	UID      string `json:"uid"`
	Password string `json:"password"`
}

// Execute 执行计划（逐条；失败不中断，错误行跳过）。
func (s *Service) Execute(items []Item) (results []ResultItem, initials []InitialPassword) {
	ds := directory.New(s.C)
	ps := people.New(s.C, s.BaseDN)

	// 已建部门缓存：路径 → DN（执行中新建的也计入，供后续人员引用）
	ouPaths, _ := s.existingOUPaths()

	for _, it := range items {
		res := ResultItem{Sheet: it.Sheet, Row: it.Row, Name: it.Name, Action: it.Action, Detail: it.Detail}
		switch {
		case it.Action == "error":
			res.Status = "skip"
		case it.Kind == "dept":
			dn, err := s.ensureDeptChain(ouPaths, it.DeptPath)
			if err != nil {
				res.Status = "fail"
				res.Detail = "新建部门失败：" + err.Error()
			} else {
				res.Status = "ok"
				res.Detail = "已创建 " + dn
			}
		case it.Kind == "person" && it.Action == "create":
			p := *it.PersonRow
			ouDN := ouPaths[p.DeptPath]
			dn, pw, err := ps.Create(people.CreateRequest{
				CN: p.Name, UID: p.UID, OU: ouDN, Mail: p.Mail,
				Mobile: p.Mobile, EmpNo: p.EmpNo, Title: p.Title,
			})
			if err != nil {
				res.Status = "fail"
				res.Detail = "新建失败：" + err.Error()
			} else {
				res.Status = "ok"
				res.Detail = "已创建 " + dn
				initials = append(initials, InitialPassword{DN: dn, UID: p.UID, Password: pw})
			}
		case it.Kind == "person" && it.Action == "update":
			p := it.PersonRow
			okDetail, err := s.updatePerson(ps, ds, ouPaths, *p)
			if err != nil {
				res.Status = "fail"
				res.Detail = "更新失败：" + err.Error()
			} else {
				res.Status = "ok"
				res.Detail = okDetail
			}
		default:
			res.Status = "skip"
		}
		results = append(results, res)
	}
	return results, initials
}

// ensureDeptChain 自底向上确保路径各级 OU 存在，返回最深层 DN。
func (s *Service) ensureDeptChain(ouPaths map[string]string, path string) (string, error) {
	segs := strings.Split(path, "/")
	cur := s.BaseDN
	for _, seg := range segs {
		name := strings.TrimSpace(seg)
		if name == "" {
			continue
		}
		next := "ou=" + name + "," + cur
		if _, err := s.C.ReadEntry(next, []string{"1.1"}); err != nil {
			if err := directory.New(s.C).Create(directory.CreateRequest{
				DN: next, Classes: []string{"top", "organizationalUnit"},
				Attrs: map[string][]string{"ou": {name}},
			}); err != nil {
				return "", fmt.Errorf("%s: %w", next, err)
			}
		}
		cur = next
	}
	// 记入缓存供人员使用
	ouPaths[path] = cur
	return cur, nil
}

// updatePerson 属性级 diff 更新 + 部门移动。
func (s *Service) updatePerson(ps *people.Service, ds *directory.Service, ouPaths map[string]string, p excel.PersonRow) (string, error) {
	e, err := s.C.ReadEntry("uid="+p.UID+","+s.parentOf(p, ouPaths), nil)
	_ = e
	// 直接按 uid 全局再定位一次（避免路径拼写差异）
	hits, err := s.C.PagedSearch(s.BaseDN, ldap.ScopeWholeSubtree,
		"(uid="+ldap.EscapeFilter(p.UID)+")", []string{"cn", "mail", "mobile", "employeeNumber", "title"}, 2)
	if err != nil || len(hits) == 0 {
		return "", fmt.Errorf("未找到 uid=%s 的已有人员", p.UID)
	}
	cur := hits[0]

	var changes []directory.Change
	addDiff := func(attr, want string) {
		got := cur.GetAttributeValue(attr)
		if want != "" && want != got {
			changes = append(changes, directory.Change{Op: "replace", Attr: attr, Vals: []string{want}})
		}
	}
	addDiff("cn", p.Name)
	addDiff("mail", p.Mail)
	addDiff("mobile", p.Mobile)
	addDiff("employeeNumber", p.EmpNo)
	addDiff("title", p.Title)

	detail := "无变化"
	if len(changes) > 0 {
		var names []string
		for _, c := range changes {
			names = append(names, c.Attr)
		}
		detail = "更新属性：" + strings.Join(names, ", ")
		if _, err := ds.Update(cur.DN, "", changes); err != nil {
			return "", err
		}
	}

	// 部门变化 → 移动
	if targetOU, ok := ouPaths[p.DeptPath]; ok && targetOU != "" {
		if newParent := parentDN(cur.DN); !strings.EqualFold(newParent, targetOU) {
			if _, err := ps.SetDept(cur.DN, targetOU); err != nil {
				return detail, fmt.Errorf("部门调动失败: %w", err)
			}
			detail += "；已调动部门 → " + targetOU
		}
	}
	return detail, nil
}

func (s *Service) parentOf(p excel.PersonRow, ouPaths map[string]string) string {
	return ouPaths[p.DeptPath]
}

func parentDN(dn string) string {
	if i := strings.Index(dn, ","); i >= 0 {
		return strings.TrimSpace(dn[i+1:])
	}
	return dn
}
