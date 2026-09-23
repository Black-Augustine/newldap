// Package planexec 是 Excel 导入与 v2 同步共用的「计划引擎」内核
//（《技术架构设计》D5）。M0 交付：diff 计算与行级计划项；执行器在 M3 落地。
package planexec

import (
	"sort"
	"strings"

	"newldap/internal/excel"
)

// Action 是计划项的执行动作。
type Action string

const (
	Create Action = "create"
	Update Action = "update"
	Error  Action = "error"
)

// Kind 是计划对象类别。
type Kind string

const (
	Dept   Kind = "dept"
	Person Kind = "person"
)

// Item 是一条计划项（预览表的一行）。
type Item struct {
	Sheet  string `json:"sheet"`
	Row    int    `json:"row"`
	Kind   Kind   `json:"kind"`
	Action Action `json:"action"`
	Name   string `json:"name"`
	Key    string `json:"key"` // 工号或部门路径
	Detail string `json:"detail"`
}

// ExistingPerson 描述目录里已存在的人员（匹配键：工号，其次 uid）。
type ExistingPerson struct {
	UID    string
	EmpNo  string
	DN     string
	DeptOU string
}

// Plan 计算部门 + 人员的导入计划：
//   - 部门：全部列为 create（路径去重已在解析层完成）；
//   - 人员：按工号（无工号按 uid）匹配已存在者 → update，否则 create；
//   - 解析层已标出的错误行原样带入计划（Action=error，执行时跳过）。
func Plan(wb *excel.Workbook, existing []ExistingPerson) []Item {
	var items []Item

	for _, d := range wb.Depts {
		items = append(items, Item{
			Sheet: excel.SheetDept, Row: d.Row, Kind: Dept, Action: Create,
			Name: d.Name, Key: d.Path, Detail: "新建部门 " + d.Path,
		})
	}

	byEmp := map[string]ExistingPerson{}
	byUID := map[string]ExistingPerson{}
	for _, p := range existing {
		if p.EmpNo != "" {
			byEmp[p.EmpNo] = p
		}
		if p.UID != "" {
			byUID[strings.ToLower(p.UID)] = p
		}
	}

	for _, p := range wb.Persons {
		match, ok := byEmp[p.EmpNo]
		if !ok && p.EmpNo == "" {
			match, ok = byUID[strings.ToLower(p.UID)]
		}
		it := Item{Sheet: excel.SheetPerson, Row: p.Row, Kind: Person, Name: p.Name, Key: p.EmpNo}
		if ok {
			it.Action = Update
			it.Detail = "更新已有人员 " + match.UID
		} else {
			it.Action = Create
			it.Detail = "新建人员 uid=" + p.UID
		}
		items = append(items, it)
	}

	for _, e := range wb.Errors {
		items = append(items, Item{Sheet: e.Sheet, Row: e.Row, Kind: Person, Action: Error, Name: "—", Detail: e.Err})
	}

	// 稳定排序：按 sheet（部门在前）+ 行号
	sort.SliceStable(items, func(i, j int) bool {
		if items[i].Sheet != items[j].Sheet {
			return items[i].Sheet == excel.SheetDept
		}
		return items[i].Row < items[j].Row
	})
	return items
}

// Summary 汇总计划项数量（预览页的四个统计卡）。
type Summary struct {
	Create int `json:"create"`
	Update int `json:"update"`
	Dept   int `json:"dept"`
	Error  int `json:"error"`
}

func Summarize(items []Item) Summary {
	var s Summary
	for _, it := range items {
		switch {
		case it.Action == Error:
			s.Error++
		case it.Kind == Dept:
			s.Dept++
		case it.Action == Create:
			s.Create++
		case it.Action == Update:
			s.Update++
		}
	}
	return s
}
