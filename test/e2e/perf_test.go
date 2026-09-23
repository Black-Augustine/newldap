// M5 性能验证（NFR2）：5000 条目目录可正常使用；1000 行 Excel 导入 ≤ 2 分钟。
// 对进程内替身测量（真实 OpenLDAP 的加速能力只多不少）；-short 时跳过。
package e2e

import (
	"bytes"
	"fmt"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/xuri/excelize/v2"

	"newldap/internal/ldapclient"
)

func TestPerformanceBudgets(t *testing.T) {
	if testing.Short() {
		t.Skip("-short 跳过性能验证")
	}
	e := newEnv(t)
	e.do(t, "POST", "/api/v1/auth/login", map[string]string{"bindDN": "cn=admin," + root, "password": "admin123"}, 200)

	// ── 造 5000 条目：50 个部门 × 每部门 99 人（+种子）
	admin, err := ldapclient.Dial(ldapclient.Options{URL: e.addr})
	if err != nil {
		t.Fatal(err)
	}
	defer admin.Close()
	if err := admin.Bind("cn=admin,"+root, "admin123"); err != nil {
		t.Fatal(err)
	}
	seedStart := time.Now()
	for d := 0; d < 50; d++ {
		ou := fmt.Sprintf("ou=perf%02d," + root, d)
		if err := admin.AddEntry(ou, map[string][]string{
			"objectClass": {"top", "organizationalUnit"}, "ou": {fmt.Sprintf("perf%02d", d)},
		}); err != nil {
			t.Fatalf("建部门 %d: %v", d, err)
		}
		for u := 0; u < 99; u++ {
			uid := fmt.Sprintf("perf%02du%02d", d, u)
			_ = admin.AddEntry(fmt.Sprintf("uid=%s,%s", uid, ou), map[string][]string{
				"objectClass":  {"top", "person", "organizationalPerson", "inetOrgPerson"},
				"uid":          {uid},
				"cn":           {"性能用户" + uid},
				"sn":           {"用户"},
				"mail":         {uid + "@example.cn"},
				"employeeNumber": {fmt.Sprintf("P%04d", d*100+u)},
				"userPassword": {"Perf1234!"},
			})
		}
	}
	t.Logf("造数完成：5000 条目，耗时 %v", time.Since(seedStart))

	// ── 检查 1：分页搜索全量人员（页 200）应流畅
	t0 := time.Now()
	res := e.do(t, "GET", "/api/v1/search?filter=(objectClass=inetOrgPerson)&attr=cn&attr=uid&limit=200", nil, 200)
	searchDur := time.Since(t0)
	if total, _ := res["total"].(float64); total < 4950 {
		t.Fatalf("全量人员 = %v，应为 ≥4950", total)
	}
	if searchDur > 5*time.Second {
		t.Errorf("全量分页搜索 %v 超出预算 5s", searchDur)
	}
	t.Logf("全量搜索 %d 条：%v", res["total"], searchDur)

	// ── 检查 2：树一层浏览（根下 50+ 节点含计数）
	t0 = time.Now()
	tree := e.do(t, "GET", "/api/v1/tree?base="+root, nil, 200)
	treeDur := time.Since(t0)
	kids, _ := tree["children"].([]any)
	if len(kids) < 50 {
		t.Fatalf("根子节点 = %d，应 ≥50", len(kids))
	}
	if treeDur > 3*time.Second {
		t.Errorf("树一层浏览+计数 %v 超出预算 3s", treeDur)
	}
	t.Logf("树浏览（%d 子节点，含逐节点计数）：%v", len(kids), treeDur)

	// ── 检查 3：1000 行 Excel 导入 ≤ 2 分钟（NFR2 硬指标）
	wb := excelize.NewFile()
	defer wb.Close()
	idx, _ := wb.NewSheet("部门")
	wb.SetActiveSheet(idx)
	wb.DeleteSheet("Sheet1")
	for i, h := range []string{"部门全路径", "上级部门", "部门名称", "描述"} {
		c, _ := excelize.CoordinatesToCellName(i+1, 1)
		_ = wb.SetCellValue("部门", c, h)
	}
	if _, err := wb.NewSheet("人员"); err != nil {
		t.Fatal(err)
	}
	for i, h := range []string{"姓名", "账号(uid)", "工号", "邮箱", "手机", "部门全路径", "职务"} {
		c, _ := excelize.CoordinatesToCellName(i+1, 1)
		_ = wb.SetCellValue("人员", c, h)
	}
	for r := 0; r < 10; r++ {
		c1, _ := excelize.CoordinatesToCellName(1, r+2)
		c2, _ := excelize.CoordinatesToCellName(3, r+2)
		_ = wb.SetCellValue("部门", c1, fmt.Sprintf("perf导入%d", r))
		_ = wb.SetCellValue("部门", c2, fmt.Sprintf("perf导入%d", r))
	}
	for r := 0; r < 1000; r++ {
		dept := fmt.Sprintf("perf导入%d", r%10)
		row := []interface{}{fmt.Sprintf("导入用户%04d", r), fmt.Sprintf("imp%04d", r),
			fmt.Sprintf("I%04d", r), fmt.Sprintf("imp%04d@example.cn", r), "13800000000", dept, "导入测试"}
		for c, v := range row {
			cell, _ := excelize.CoordinatesToCellName(c+1, r+2)
			_ = wb.SetCellValue("人员", cell, v)
		}
	}
	tmp := filepath.Join(t.TempDir(), "perf.xlsx")
	if err := wb.SaveAs(tmp); err != nil {
		t.Fatal(err)
	}
	info, _ := os.Stat(tmp)
	t.Logf("导入文件：%d 行人员 + 10 部门（%.1f KB）", 1000, float64(info.Size())/1024)

	body := &bytes.Buffer{}
	mw := multipart.NewWriter(body)
	fw, _ := mw.CreateFormFile("file", "perf.xlsx")
	b, _ := os.ReadFile(tmp)
	fw.Write(b)
	mw.Close()
	t0 = time.Now()
	req, _ := http.NewRequest("POST", e.ts.URL+"/api/v1/import/parse", body)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	req.Header.Set("Cookie", "nlsid="+e.session)
	resp, err := http.DefaultClient.Do(req)
	if err != nil || resp.StatusCode != 200 {
		t.Fatalf("parse: %v %s", err, resp.Status)
	}
	parseRes := decodeBody(t, resp)
	sum := parseRes["summary"].(map[string]any)
	if sum["create"].(float64) != 1000 || sum["dept"].(float64) != 10 {
		t.Fatalf("计划汇总异常: %+v", sum)
	}
	execRes := e.do(t, "POST", "/api/v1/import/execute", map[string]any{"items": parseRes["items"]}, 200)
	es := execRes["summary"].(map[string]any)
	importDur := time.Since(t0)
	if es["fail"].(float64) != 0 || es["ok"].(float64) != 1010 {
		t.Fatalf("执行结果异常: %+v", es)
	}
	if importDur > 2*time.Minute {
		t.Errorf("1000 行导入 %v 超出 NFR2 预算 2 分钟", importDur)
	}
	t.Logf("1000 行导入（解析+计划+执行 1010 项）：%v（预算 ≤ 2m）", importDur)

	// ── 检查 4：全局搜索在 6000+ 条目目录中应秒回
	t0 = time.Now()
	sr := e.do(t, "GET", "/api/v1/search?q=imp0999", nil, 200)
	if sr["total"] != float64(1) {
		t.Errorf("精确搜索命中 = %v", sr["total"])
	}
	t.Logf("6000+ 条目目录中关键字搜索：%v", time.Since(t0))
}
