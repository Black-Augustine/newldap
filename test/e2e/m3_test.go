// M3 验收 e2e：模板下载 → 填写 → 解析计划（含错误行）→ 执行 → 目录核验 → 导出。
package e2e

import (
	"bytes"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xuri/excelize/v2"
)

func decodeBody(t *testing.T, resp *http.Response) map[string]any {
	t.Helper()
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatalf("响应不是 JSON: %.200s", b)
	}
	return m
}

func fmtJSON(v any) string {
	b, _ := json.Marshal(v)
	return string(b)
}

// 下载模板（带会话），返回落盘路径。
func downloadTemplate(t *testing.T, e *env) string {
	t.Helper()
	req, _ := http.NewRequest("GET", e.ts.URL+"/api/v1/import/template", nil)
	req.Header.Set("Cookie", "nlsid="+e.session)
	resp, err := http.DefaultClient.Do(req)
	if err != nil || resp.StatusCode != 200 {
		t.Fatalf("模板下载: %v %s", err, resp.Status)
	}
	defer resp.Body.Close()
	if !strings.Contains(resp.Header.Get("Content-Type"), "spreadsheet") {
		t.Fatalf("模板 Content-Type = %s", resp.Header.Get("Content-Type"))
	}
	dir := t.TempDir()
	p := filepath.Join(dir, "tpl.xlsx")
	b, _ := io.ReadAll(resp.Body)
	if err := os.WriteFile(p, b, 0o600); err != nil {
		t.Fatal(err)
	}
	return p
}

// 在模板上追加：2 部门 + 3 人（新建/邮箱错/更新+调动）。
func buildFilledWorkbook(t *testing.T, e *env) []byte {
	t.Helper()
	f, err := excelize.OpenFile(downloadTemplate(t, e))
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
	// 清掉模板自带的示例行（真实用户会替换为自己的数据）
	write("部门", 2, []interface{}{"", "", "", ""})
	write("部门", 3, []interface{}{"", "", "", ""})
	write("人员", 2, []interface{}{"", "", "", "", "", "", ""})
	write("部门", 4, []interface{}{"数据智能部", "", "数据智能部", ""})
	write("部门", 5, []interface{}{"数据智能部/算法组", "数据智能部", "算法组", ""})
	write("人员", 3, []interface{}{"许倩", "xuqian", "E1021", "xuqian@example.cn", "13800000021", "数据智能部", "算法工程师"})
	write("人员", 4, []interface{}{"邓超", "dengchao", "E1022", "dengchao#example.cn", "13800000022", "数据智能部/算法组", ""})     // 邮箱格式错 → 错误行
	write("人员", 5, []interface{}{"张伟改", "zhangwei", "E1001", "zhangwei@example.cn", "139-0000-9999", "数据智能部", "平台架构师"}) // 更新 + 调动
	buf := &bytes.Buffer{}
	if err := f.Write(buf); err != nil {
		t.Fatal(err)
	}
	f.Close()
	return buf.Bytes()
}

func uploadParse(t *testing.T, e *env, wb []byte) map[string]any {
	t.Helper()
	body := &bytes.Buffer{}
	mw := multipart.NewWriter(body)
	fw, _ := mw.CreateFormFile("file", "组织与人员_测试.xlsx")
	fw.Write(wb)
	mw.Close()
	req, _ := http.NewRequest("POST", e.ts.URL+"/api/v1/import/parse", body)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	req.Header.Set("Cookie", "nlsid="+e.session)
	resp, err := http.DefaultClient.Do(req)
	if err != nil || resp.StatusCode != 200 {
		t.Fatalf("parse: %v %s", err, resp.Status)
	}
	return decodeBody(t, resp)
}

func TestImportExportFlow(t *testing.T) {
	e := newEnv(t)
	e.do(t, "POST", "/api/v1/auth/login", map[string]string{"bindDN": "cn=admin," + root, "password": "admin123"}, 200)

	// 1-2. 模板下载 + 填写 + 解析
	parseRes := uploadParse(t, e, buildFilledWorkbook(t, e))
	sum := parseRes["summary"].(map[string]any)
	if sum["dept"].(float64) != 2 || sum["create"].(float64) != 1 || sum["update"].(float64) != 1 || sum["error"].(float64) != 1 {
		t.Fatalf("计划汇总 = %+v（应 部门2 新建1 更新1 错误1）", sum)
	}

	// 3. 执行（跳过错误行）
	execRes := e.do(t, "POST", "/api/v1/import/execute", map[string]any{"items": parseRes["items"]}, 200)
	es := execRes["summary"].(map[string]any)
	for _, r := range execRes["results"].([]any) {
		t.Logf("result: %v", r)
	}
	if es["ok"].(float64) != 4 || es["skip"].(float64) != 1 || es["fail"].(float64) != 0 {
		t.Fatalf("执行汇总 = %+v（应 ok=4 skip=1 fail=0）", es)
	}

	// 初始密码真实可 bind（仅新建者获得）
	initials, _ := execRes["initialPasswords"].([]any)
	if len(initials) != 1 {
		t.Fatalf("初始密码数 = %d，应为 1（仅新建的许倩）", len(initials))
	}
	ip := initials[0].(map[string]any)
	if ip["uid"] != "xuqian" {
		t.Errorf("初始密码对应 uid = %v", ip["uid"])
	}
	e.do(t, "POST", "/api/v1/auth/login", map[string]string{"bindDN": ip["dn"].(string), "password": ip["password"].(string)}, 200)
	e.do(t, "POST", "/api/v1/auth/login", map[string]string{"bindDN": "cn=admin," + root, "password": "admin123"}, 200)

	// 4. 目录核验
	tree := e.do(t, "GET", "/api/v1/tree?base="+root, nil, 200)
	if !strings.Contains(fmtJSON(tree), "数据智能部") {
		t.Error("根下未见 数据智能部")
	}
	zd := e.do(t, "GET", "/api/v1/people?dn=uid=xuqian,ou=数据智能部,"+root, nil, 200)
	zattrs := zd["attrs"].(map[string]any)
	if zattrs["title"] == nil || zattrs["mail"] == nil {
		t.Errorf("许倩属性不完整: %+v", zattrs)
	}
	moved := e.do(t, "GET", "/api/v1/people?dn=uid=zhangwei,ou=数据智能部,"+root, nil, 200)
	mattrs := moved["attrs"].(map[string]any)
	if mattrs["cn"].([]any)[0] != "张伟改" || mattrs["mobile"] == nil {
		t.Errorf("张伟更新/调动未生效: %+v", mattrs)
	}
	if sres := e.do(t, "GET", "/api/v1/search?q=dengchao", nil, 200); sres["total"] != float64(0) {
		t.Errorf("错误行 dengchao 不应被创建，total=%v", sres["total"])
	}

	// 5. 导出：可打开、含新人、绝无密码字段
	req2, _ := http.NewRequest("GET", e.ts.URL+"/api/v1/export/people", nil)
	req2.Header.Set("Cookie", "nlsid="+e.session)
	resp2, err := http.DefaultClient.Do(req2)
	if err != nil || resp2.StatusCode != 200 {
		t.Fatalf("export: %v %s", err, resp2.Status)
	}
	out, _ := io.ReadAll(resp2.Body)
	resp2.Body.Close()
	f, err := excelize.OpenReader(bytes.NewReader(out))
	if err != nil {
		t.Fatalf("导出文件无法打开: %v", err)
	}
	rows, _ := f.GetRows("人员")
	found := false
	for _, r := range rows {
		joined := strings.ToLower(strings.Join(r, ","))
		if strings.Contains(joined, "xuqian") {
			found = true
		}
		if strings.Contains(joined, "password") || strings.Contains(joined, "passw0rd") {
			t.Error("导出包含密码字段/值！")
		}
	}
	if !found {
		t.Error("导出缺少 xuqian")
	}

	// 6. 审计有 import 事件
	auditHas(t, `"op":"import"`)
}
