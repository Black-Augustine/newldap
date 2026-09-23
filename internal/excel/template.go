// Package excel 实现中文模板的生成与解析（R7.1~R7.4 的基础，spike③）。
// M0 交付：固定列模板（部门 / 人员两个工作表）+ 行级校验错误定位。
package excel

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/xuri/excelize/v2"
)

// 模板列（中文表头）
const (
	SheetDept   = "部门"
	SheetPerson = "人员"
	SheetGuide  = "填写说明"
)

var (
	deptHeader   = []string{"部门全路径", "上级部门", "部门名称", "描述"}
	personHeader = []string{"姓名", "账号(uid)", "工号", "邮箱", "手机", "部门全路径", "职务"}
)

// DeptRow 是「部门」表的一行。
type DeptRow struct {
	Row    int    `json:"row"` // Excel 行号（1 起，含表头）
	Path   string // 如 技术研发部/平台研发组
	Parent string
	Name   string
	Desc   string
}

// PersonRow 是「人员」表的一行。
type PersonRow struct {
	Row      int    `json:"row"`
	Name     string
	UID      string
	EmpNo    string
	Mail     string
	Mobile   string
	DeptPath string
	Title    string
}

// RowError 是一条行级错误（"第 N 行 / 原因"，R7.5）。
type RowError struct {
	Sheet string `json:"sheet"`
	Row   int    `json:"row"`
	Err   string `json:"err"`
}

// Workbook 是一次解析的结果。
type Workbook struct {
	Depts   []DeptRow
	Persons []PersonRow
	Errors  []RowError
}

// WriteTemplate 生成导入模板到 path。
func WriteTemplate(path string) error {
	f := excelize.NewFile()
	defer f.Close()

	// 部门
	idx, _ := f.NewSheet(SheetDept)
	f.SetActiveSheet(idx)
	f.DeleteSheet("Sheet1")
	for i, h := range deptHeader {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		f.SetCellValue(SheetDept, cell, h)
	}
	examples := [][]interface{}{
		{"技术研发部", "", "技术研发部", "负责产品研发"},
		{"技术研发部/平台研发组", "技术研发部", "平台研发组", ""},
	}
	for r, row := range examples {
		for c, v := range row {
			cell, _ := excelize.CoordinatesToCellName(c+1, r+2)
			f.SetCellValue(SheetDept, cell, v)
		}
	}

	// 人员
	if _, err := f.NewSheet(SheetPerson); err != nil {
		return err
	}
	for i, h := range personHeader {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		f.SetCellValue(SheetPerson, cell, h)
	}
	pexamples := [][]interface{}{
		{"张伟", "zhangwei", "E1001", "zhangwei@example.cn", "13800000001", "技术研发部", "平台研发工程师"},
	}
	for r, row := range pexamples {
		for c, v := range row {
			cell, _ := excelize.CoordinatesToCellName(c+1, r+2)
			f.SetCellValue(SheetPerson, cell, v)
		}
	}

	// 说明
	if _, err := f.NewSheet(SheetGuide); err != nil {
		return err
	}
	guide := []string{
		"1. 【部门】按层级填写「部门全路径」，用 / 分隔，如：技术研发部/平台研发组。",
		"2. 【人员】必填：姓名、账号(uid)、部门全路径；工号是导入匹配的唯一标识，重复视为错误。",
		"3. 已存在的人员（按工号匹配）默认更新其姓名/邮箱/手机/部门/职务。",
		"4. 不需要填密码：新账号首次登录须自助改密。",
		"5. 部门路径不存在时，导入向导会提示并（经你确认后）自动创建。",
	}
	for i, line := range guide {
		cell, _ := excelize.CoordinatesToCellName(1, i+1)
		f.SetCellValue(SheetGuide, cell, line)
	}

	for sheet, cols := range map[string]float64{SheetDept: 34, SheetPerson: 26} {
		for c := 1; c <= len(deptHeader)+len(personHeader); c++ {
			name, _ := excelize.ColumnNumberToName(c)
			f.SetColWidth(sheet, name, name, cols)
		}
	}
	if err := f.SaveAs(path); err != nil {
		return fmt.Errorf("保存模板失败: %w", err)
	}
	return nil
}

// ParseFile 解析填写后的工作簿：结构读入 + 行级校验。
// 校验规则（M0）：姓名/账号/部门路径必填；工号重复报错；邮箱格式粗检。
func ParseFile(path string) (*Workbook, error) {
	f, err := excelize.OpenFile(path)
	if err != nil {
		return nil, fmt.Errorf("打开 %s 失败: %w", filepath.Base(path), err)
	}
	defer f.Close()
	wb := &Workbook{}

	// 部门
	if rows, err := f.GetRows(SheetDept); err == nil {
		seenPath := map[string]int{}
		for i, row := range rows {
			no := i + 1
			if no == 1 || allEmpty(row) {
				continue
			}
			pad := padRow(row, 4)
			d := DeptRow{Row: no, Path: strings.TrimSpace(pad[0]), Parent: strings.TrimSpace(pad[1]), Name: strings.TrimSpace(pad[2]), Desc: strings.TrimSpace(pad[3])}
			if d.Path == "" {
				wb.Errors = append(wb.Errors, RowError{SheetDept, no, "部门全路径为空"})
				continue
			}
			if prev, dup := seenPath[d.Path]; dup {
				wb.Errors = append(wb.Errors, RowError{SheetDept, no, fmt.Sprintf("部门路径与第 %d 行重复：%s", prev, d.Path)})
				continue
			}
			seenPath[d.Path] = no
			wb.Depts = append(wb.Depts, d)
		}
	}

	// 人员
	if rows, err := f.GetRows(SheetPerson); err == nil {
		seenEmp := map[string]int{}
		for i, row := range rows {
			no := i + 1
			if no == 1 || allEmpty(row) {
				continue
			}
			pad := padRow(row, 7)
			p := PersonRow{Row: no, Name: strings.TrimSpace(pad[0]), UID: strings.TrimSpace(pad[1]), EmpNo: strings.TrimSpace(pad[2]), Mail: strings.TrimSpace(pad[3]), Mobile: strings.TrimSpace(pad[4]), DeptPath: strings.TrimSpace(pad[5]), Title: strings.TrimSpace(pad[6])}
			switch {
			case p.Name == "":
				wb.Errors = append(wb.Errors, RowError{SheetPerson, no, "姓名为空"})
				continue
			case p.UID == "":
				wb.Errors = append(wb.Errors, RowError{SheetPerson, no, "账号(uid)为空"})
				continue
			case p.DeptPath == "":
				wb.Errors = append(wb.Errors, RowError{SheetPerson, no, "部门全路径为空"})
				continue
			}
			if p.EmpNo != "" {
				if prev, dup := seenEmp[p.EmpNo]; dup {
					wb.Errors = append(wb.Errors, RowError{SheetPerson, no, fmt.Sprintf("工号 %s 与第 %d 行重复", p.EmpNo, prev)})
					continue
				}
				seenEmp[p.EmpNo] = no
			}
			if p.Mail != "" && !strings.Contains(p.Mail, "@") {
				wb.Errors = append(wb.Errors, RowError{SheetPerson, no, fmt.Sprintf("邮箱格式不正确：%s", p.Mail)})
				continue
			}
			wb.Persons = append(wb.Persons, p)
		}
	} else {
		return nil, fmt.Errorf("缺少「%s」工作表", SheetPerson)
	}
	return wb, nil
}

func allEmpty(row []string) bool {
	for _, c := range row {
		if strings.TrimSpace(c) != "" {
			return false
		}
	}
	return true
}

func padRow(row []string, n int) []string {
	out := make([]string, n)
	copy(out, row)
	return out
}
