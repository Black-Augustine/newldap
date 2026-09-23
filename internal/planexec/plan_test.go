package planexec

import (
	"testing"

	"newldap/internal/excel"
)

func TestPlanDiff(t *testing.T) {
	wb := &excel.Workbook{
		Depts: []excel.DeptRow{
			{Row: 3, Path: "数据智能部", Name: "数据智能部"},
			{Row: 4, Path: "数据智能部/算法组", Name: "算法组"},
		},
		Persons: []excel.PersonRow{
			{Row: 7, Name: "许倩", UID: "xuqian", EmpNo: "E1021", DeptPath: "数据智能部"},
			{Row: 8, Name: "张伟", UID: "zhangwei", EmpNo: "E1001", DeptPath: "技术研发部"},
			{Row: 9, Name: "无名", UID: "noname", DeptPath: "数据智能部"}, // 无工号新人
		},
		Errors: []excel.RowError{
			{Sheet: excel.SheetPerson, Row: 10, Err: "邮箱格式不正确"},
		},
	}
	existing := []ExistingPerson{
		{UID: "zhangwei", EmpNo: "E1001", DN: "uid=zhangwei,ou=tech,dc=example,dc=cn"},
	}
	items := Plan(wb, existing)
	s := Summarize(items)
	if s.Create != 2 || s.Update != 1 || s.Dept != 2 || s.Error != 1 {
		t.Fatalf("汇总 = %+v，应为 {Create:2 Update:1 Dept:2 Error:1}", s)
	}
	// 无工号者按 uid 匹配：noname 不在 existing → create（已计入）
	// 张伟按工号匹配 → update
	var zhang, xu *Item
	for i := range items {
		switch items[i].Name {
		case "张伟":
			zhang = &items[i]
		case "许倩":
			xu = &items[i]
		}
	}
	if zhang == nil || zhang.Action != Update {
		t.Errorf("张伟应按工号匹配为 update: %+v", zhang)
	}
	if xu == nil || xu.Action != Create {
		t.Errorf("许倩应为 create: %+v", xu)
	}
	// 排序：部门项在前，行号升序
	if items[0].Kind != Dept || items[1].Kind != Dept {
		t.Errorf("部门项应排在最前: %+v", items[:2])
	}
	if items[2].Row > items[3].Row {
		t.Errorf("人员项应按行号升序")
	}
}
