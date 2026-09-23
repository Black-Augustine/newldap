package excel

import (
	"path/filepath"
	"testing"

	"github.com/xuri/excelize/v2"
)

// TestTemplateRoundTrip 覆盖：模板生成 → 按模板填写（含脏数据）→ 解析 →
// 行级校验定位。这是 M0 spike③ 的验收测试。
func TestTemplateRoundTrip(t *testing.T) {
	dir := t.TempDir()
	tpl := filepath.Join(dir, "template.xlsx")
	if err := WriteTemplate(tpl); err != nil {
		t.Fatal(err)
	}

	// 在模板副本上追加数据：2 个部门 + 4 个人（含 3 行脏数据）
	f, err := excelize.OpenFile(tpl)
	if err != nil {
		t.Fatal(err)
	}
	write := func(sheet string, rowNo int, vals []interface{}) {
		for c, v := range vals {
			cell, _ := excelize.CoordinatesToCellName(c+1, rowNo)
			if err := f.SetCellValue(sheet, cell, v); err != nil {
				t.Fatal(err)
			}
		}
	}
	write(SheetDept, 4, []interface{}{"数据智能部", "", "数据智能部", ""})
	write(SheetDept, 5, []interface{}{"数据智能部/算法组", "数据智能部", "算法组", ""})
	write(SheetDept, 6, []interface{}{"数据智能部", "", "数据智能部", "重复路径"}) // 第6行：路径重复

	write(SheetPerson, 3, []interface{}{"许倩", "xuqian", "E1021", "xuqian@example.cn", "13800000021", "数据智能部", "算法工程师"})
	write(SheetPerson, 4, []interface{}{"邓超", "dengchao", "E1022", "dengchao#example.cn", "13800000022", "数据智能部/算法组", ""}) // 邮箱格式错
	write(SheetPerson, 5, []interface{}{"", "nobody", "E1023", "", "", "数据智能部", ""})                                            // 姓名为空
	write(SheetPerson, 6, []interface{}{"崔敏", "cuimin", "E1021", "", "", "数据智能部", ""})                                          // 工号重复

	filled := filepath.Join(dir, "filled.xlsx")
	if err := f.SaveAs(filled); err != nil {
		t.Fatal(err)
	}
	f.Close()

	wb, err := ParseFile(filled)
	if err != nil {
		t.Fatal(err)
	}

	// 示例行也在模板里，属于合法数据
	if len(wb.Depts) != 4 { // 2 模板示例 + 2 新增（第 6 行重复被拒）
		t.Errorf("部门行 = %d（%+v），应为 4", len(wb.Depts), wb.Depts)
	}
	if len(wb.Persons) != 2 { // 1 示例 + 1 新增合法
		t.Errorf("人员行 = %d（%+v），应为 2", len(wb.Persons), wb.Persons)
	}
	if len(wb.Errors) != 4 {
		t.Fatalf("错误行 = %d，应为 4: %+v", len(wb.Errors), wb.Errors)
	}
	// 逐一断言错误行号与原因
	type want struct{ sheet string; row int; contains string }
	wants := []want{
		{SheetDept, 6, "重复"},
		{SheetPerson, 4, "邮箱"},
		{SheetPerson, 5, "姓名"},
		{SheetPerson, 6, "工号"},
	}
	for _, w := range wants {
		found := false
		for _, e := range wb.Errors {
			if e.Sheet == w.sheet && e.Row == w.row && contains(e.Err, w.contains) {
				found = true
			}
		}
		if !found {
			t.Errorf("缺少错误：sheet=%s 行=%d 含 %q，实际 %+v", w.sheet, w.row, w.contains, wb.Errors)
		}
	}
}

func contains(s, sub string) bool {
	return len(sub) == 0 || (len(s) >= len(sub) && indexOf(s, sub) >= 0)
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}

func TestChineseHeaders(t *testing.T) {
	dir := t.TempDir()
	tpl := filepath.Join(dir, "t.xlsx")
	if err := WriteTemplate(tpl); err != nil {
		t.Fatal(err)
	}
	f, err := excelize.OpenFile(tpl)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	got, err := f.GetRows(SheetPerson)
	if err != nil {
		t.Fatal(err)
	}
	for i, h := range personHeader {
		if got[0][i] != h {
			t.Errorf("人员表第 %d 列表头 = %q，应为 %q", i+1, got[0][i], h)
		}
	}
}
