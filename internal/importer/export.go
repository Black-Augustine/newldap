package importer

import (
	"bytes"
	"fmt"

	"github.com/xuri/excelize/v2"

	"newldap/internal/excel"
	ldap0 "newldap/internal/ldapclient"
)

// ExportPeople 导出部门 + 人员到 Excel（R7.6：绝不包含密码与哈希）。
func (s *Service) ExportPeople() ([]byte, error) {
	f := excelize.NewFile()
	idx, _ := f.NewSheet(excel.SheetDept)
	f.SetActiveSheet(idx)
	f.DeleteSheet("Sheet1")

	// 部门表
	for i, h := range []string{"部门全路径", "上级部门", "部门名称", "描述"} {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		_ = f.SetCellValue(excel.SheetDept, cell, h)
	}
	ouPaths, err := s.existingOUPaths()
	if err != nil {
		return nil, err
	}
	dnDesc, _ := s.ouDescriptions()
	row := 2
	for path, dn := range ouPaths {
		segs := splitPath(path)
		name := segs[len(segs)-1]
		parent := ""
		if len(segs) > 1 {
			parent = joinPath(segs[:len(segs)-1])
		}
		for c, v := range []interface{}{path, parent, name, dnDesc[dn]} {
			cell, _ := excelize.CoordinatesToCellName(c+1, row)
			_ = f.SetCellValue(excel.SheetDept, cell, v)
		}
		row++
	}

	// 人员表
	if _, err := f.NewSheet(excel.SheetPerson); err != nil {
		return nil, err
	}
	for i, h := range []string{"姓名", "账号(uid)", "工号", "邮箱", "手机", "部门全路径", "职务"} {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		_ = f.SetCellValue(excel.SheetPerson, cell, h)
	}
	persons, err := s.existingPersons()
	if err != nil {
		return nil, err
	}
	dnToPath := map[string]string{}
	for p, dn := range ouPaths {
		dnToPath[dn] = p
	}
	row = 2
	for _, p := range persons {
		dept := dnToPath[parentDN(p.DN)]
		for c, v := range []interface{}{p.CN, p.UID, p.EmpNo, p.Mail, p.Mobile, dept, p.Title} {
			cell, _ := excelize.CoordinatesToCellName(c+1, row)
			_ = f.SetCellValue(excel.SheetPerson, cell, v)
		}
		row++
	}

	var buf bytes.Buffer
	if err := f.Write(&buf); err != nil {
		return nil, fmt.Errorf("生成导出文件失败: %w", err)
	}
	return buf.Bytes(), nil
}

func (s *Service) ouDescriptions() (map[string]string, error) {
	entries, err := s.C.PagedSearch(s.BaseDN, 2, "(objectClass=organizationalUnit)", []string{"ou", "description"}, 2000) //nolint:mnd
	if err != nil {
		return nil, err
	}
	out := map[string]string{}
	for _, e := range entries {
		out[e.DN] = e.GetAttributeValue("description")
	}
	return out, nil
}

func splitPath(p string) []string {
	var segs []string
	for _, s := range bytes.Split([]byte(p), []byte("/")) {
		if len(s) > 0 {
			segs = append(segs, string(s))
		}
	}
	return segs
}

func joinPath(segs []string) string {
	out := ""
	for i, s := range segs {
		if i > 0 {
			out += "/"
		}
		out += s
	}
	return out
}

var _ = ldap0.Options{} // 保持 import（客户端类型在接口中使用）
