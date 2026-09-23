// M3 端点：Excel 导入（parse/execute/template）、导出、审计查询。
package api

import (
	"fmt"
	"net/http"
	"os"
	"strings"

	"newldap/internal/audit"
	"newldap/internal/auth"
	"newldap/internal/excel"
	"newldap/internal/importer"
)

func (d *Deps) mountM3(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/v1/import/parse", d.withSession(d.handleImportParse))
	mux.HandleFunc("POST /api/v1/import/execute", d.withSession(d.handleImportExecute))
	mux.HandleFunc("GET /api/v1/import/template", d.withSession(d.handleImportTemplate))
	mux.HandleFunc("GET /api/v1/export/people", d.withSession(d.handleExportPeople))
	mux.HandleFunc("GET /api/v1/audit", d.withSession(d.handleAudit))
	mux.HandleFunc("GET /api/v1/audit/export", d.withSession(d.handleAuditExport))
}

const maxImportSize = 10 << 20 // 10MB

func (d *Deps) handleImportParse(w http.ResponseWriter, r *http.Request, s *auth.Session) {
	r.Body = http.MaxBytesReader(w, r.Body, maxImportSize)
	_, hdr, err := r.FormFile("file")
	if err != nil {
		writeErr(w, http.StatusBadRequest, "请上传 .xlsx 文件（≤10MB）")
		return
	}
	svc := importer.Service{C: s.Conn, BaseDN: d.curProfile().BaseDN}
	file, items, summary, err := svc.ParsePlan(hdr)
	if err != nil {
		writeErr(w, http.StatusBadRequest, "解析失败："+err.Error())
		return
	}
	d.Audit.Log(audit.Event{Actor: s.BindDN, IP: clientIP(r), Op: "import", Result: "ok",
		Detail: fmt.Sprintf("解析 %s：新建%d 更新%d 部门%d 错误%d", file, summary.Create, summary.Update, summary.Dept, summary.Error)})
	writeJSON(w, http.StatusOK, map[string]any{"file": file, "summary": summary, "items": items})
}

func (d *Deps) handleImportExecute(w http.ResponseWriter, r *http.Request, s *auth.Session) {
	var req struct {
		Items []importer.Item `json:"items"`
	}
	if err := readJSON(r, &req); err != nil || len(req.Items) == 0 {
		writeErr(w, http.StatusBadRequest, "请求体不合法（需要 items）")
		return
	}
	svc := importer.Service{C: s.Conn, BaseDN: d.curProfile().BaseDN}
	results, initials := svc.Execute(req.Items)
	var okN, failN, skipN int
	for _, res := range results {
		switch res.Status {
		case "ok":
			okN++
		case "fail":
			failN++
		default:
			skipN++
		}
	}
	d.Audit.Log(audit.Event{Actor: s.BindDN, IP: clientIP(r), Op: "import", Result: func() string {
		if failN > 0 {
			return "fail"
		}
		return "ok"
	}(),
		Detail: fmt.Sprintf("执行：成功%d 失败%d 跳过%d", okN, failN, skipN)})
	writeJSON(w, http.StatusOK, map[string]any{
		"summary": map[string]int{"ok": okN, "fail": failN, "skip": skipN},
		"results": results, "initialPasswords": initials,
	})
}

func (d *Deps) handleImportTemplate(w http.ResponseWriter, r *http.Request, s *auth.Session) {
	tmp, err := os.CreateTemp("", "newldap-tpl-*.xlsx")
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "生成模板失败")
		return
	}
	name := tmp.Name()
	tmp.Close()
	defer os.Remove(name)
	if err := excel.WriteTemplate(name); err != nil {
		writeErr(w, http.StatusInternalServerError, "生成模板失败："+err.Error())
		return
	}
	b, err := os.ReadFile(name)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "读取模板失败")
		return
	}
	w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	w.Header().Set("Content-Disposition", `attachment; filename="newldap-导入模板.xlsx"`)
	w.Write(b) //nolint:errcheck
}

func (d *Deps) handleExportPeople(w http.ResponseWriter, r *http.Request, s *auth.Session) {
	svc := importer.Service{C: s.Conn, BaseDN: d.curProfile().BaseDN}
	b, err := svc.ExportPeople()
	if err != nil {
		if !writeLdapErr(w, err, "导出") {
			writeErr(w, http.StatusBadGateway, "导出失败："+err.Error())
		}
		return
	}
	d.Audit.Log(audit.Event{Actor: s.BindDN, IP: clientIP(r), Op: "export", Result: "ok",
		Detail: "导出部门+人员（不含密码）"})
	w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	w.Header().Set("Content-Disposition", `attachment; filename="newldap-组织与人员.xlsx"`)
	w.Write(b) //nolint:errcheck
}

func (d *Deps) handleAudit(w http.ResponseWriter, r *http.Request, s *auth.Session) {
	limit := atoiDefault(r.URL.Query().Get("limit"), 50)
	if limit < 1 || limit > 500 {
		limit = 50
	}
	events := audit.ReadLast(d.Cfg.Server.AuditFile, limit)
	writeJSON(w, http.StatusOK, map[string]any{"events": events, "total": len(events)})
}

// handleAuditExport 审计导出（CSV，Excel 可直接打开；R11.3/M5）。
func (d *Deps) handleAuditExport(w http.ResponseWriter, r *http.Request, s *auth.Session) {
	limit := atoiDefault(r.URL.Query().Get("limit"), 1000)
	if limit < 1 || limit > 10000 {
		limit = 1000
	}
	events := audit.ReadLast(d.Cfg.Server.AuditFile, limit)
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="newldap-audit.csv"`)
	// BOM：让 Excel 正确识别 UTF-8
	w.Write([]byte{0xEF, 0xBB, 0xBF})              //nolint:errcheck
	w.Write([]byte("时间,操作人,来源IP,操作,目标DN,详情,结果\n")) //nolint:errcheck
	for _, e := range events {
		row := []string{e.TS.Format("2006-01-02 15:04:05"), e.Actor, e.IP, e.Op, e.DN, e.Detail, e.Result}
		for i, v := range row {
			if strings.ContainsAny(v, ",\"\n") {
				row[i] = "\"" + strings.ReplaceAll(v, "\"", "\"\"") + "\""
			}
		}
		w.Write([]byte(strings.Join(row, ",") + "\n")) //nolint:errcheck
	}
}
