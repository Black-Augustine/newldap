// Package e2e 是 M0 冒烟链路验收：登录（bind）→ 树浏览 → 读取条目 →
// 属性级修改（含冲突拦截）→ 移动（modrdn），全程走真实 HTTP API +
// 进程内 LDAP 替身（BER 线协议真实收发）。
package e2e

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"newldap/internal/api"
	"newldap/internal/audit"
	"newldap/internal/auth"
	"newldap/internal/config"
	"newldap/internal/ldaptest"
)

const root = "dc=example,dc=cn"

var auditFile string

func TestMain(m *testing.M) {
	auditFile = filepath.Join(os.TempDir(), fmt.Sprintf("newldap-e2e-audit-%d.jsonl", time.Now().UnixNano()))
	os.Exit(m.Run())
}

type env struct {
	ts      *httptest.Server
	session string // nlsid cookie 值
	addr    string // 本 env 专属的替身目录地址（用例间数据隔离）
}

// newEnvWithCfg 启动一套独立的替身目录 + HTTP 服务（每个用例自己的数据副本）。
func newEnvWithCfg(t *testing.T, cfg *config.Config) *env {
	t.Helper()
	srv, err := ldaptest.Start(true)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(srv.Stop)
	aud, err := audit.New(auditFile)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { aud.Close() })
	deps := &api.Deps{Cfg: cfg, Sessions: auth.NewStore(cfg.Server.SessionTTL), Audit: aud}
	ts := httptest.NewServer(api.New(deps))
	t.Cleanup(ts.Close)
	return &env{ts: ts, addr: srv.Addr()}
}

func newEnv(t *testing.T) *env {
	t.Helper()
	cfg := &config.Config{}
	cfg.Server.SessionTTL = time.Minute
	cfg.Profile.BaseDN = root
	cfg.Server.AuditFile = auditFile
	e := newEnvWithCfg(t, cfg)
	cfg.Profile.URL = e.addr // 已配置环境：直接指向本用例的目录
	return e
}

func (e *env) do(t *testing.T, method, path string, body any, wantStatus int) map[string]any {
	t.Helper()
	var rd io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			t.Fatal(err)
		}
		rd = bytes.NewReader(b)
	}
	req, err := http.NewRequest(method, e.ts.URL+path, rd)
	if err != nil {
		t.Fatal(err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if e.session != "" {
		req.Header.Set("Cookie", "nlsid="+e.session)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	for _, c := range resp.Cookies() {
		if c.Name == "nlsid" {
			e.session = c.Value
		}
	}
	b, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != wantStatus {
		t.Fatalf("%s %s → %d（期望 %d）: %s", method, path, resp.StatusCode, wantStatus, b)
	}
	var out map[string]any
	if len(b) > 0 {
		json.Unmarshal(b, &out) //nolint:errcheck
	}
	return out
}

// doWithOrigin 带指定 Origin 头发起写请求（CSRF 用例）。
func (e *env) doWithOrigin(t *testing.T, method, path string, body any, origin string, wantStatus int) map[string]any {
	t.Helper()
	var rd io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			t.Fatal(err)
		}
		rd = bytes.NewReader(b)
	}
	req, err := http.NewRequest(method, e.ts.URL+path, rd)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Origin", origin)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != wantStatus {
		t.Fatalf("%s %s (Origin=%s) → %d（期望 %d）: %s", method, path, origin, resp.StatusCode, wantStatus, b)
	}
	var out map[string]any
	if len(b) > 0 {
		json.Unmarshal(b, &out) //nolint:errcheck
	}
	return out
}

// TestSmokeFullChain 是 M0 验收用例：完整冒烟链路。
func TestSmokeFullChain(t *testing.T) {
	e := newEnv(t)

	// 0) 健康检查（api 会独立拨号替身目录）
	h := e.do(t, "GET", "/healthz", nil, 200)
	if h["ldap"] != "up" {
		t.Fatalf("healthz.ldap = %v（detail=%v）", h["ldap"], h["detail"])
	}

	// 1) 未登录访问受保护接口 → 401
	e.do(t, "GET", "/api/v1/tree?base="+root, nil, 401)

	// 2) 登录：错误密码 → 401（不区分账号/密码错误）；正确 → 200 + 会话
	e.do(t, "POST", "/api/v1/auth/login", map[string]string{"bindDN": "cn=admin," + root, "password": "错的"}, 401)
	e.do(t, "POST", "/api/v1/auth/login", map[string]string{"bindDN": "cn=admin," + root, "password": "admin123"}, 200)
	if e.session == "" {
		t.Fatal("登录未返回会话 Cookie")
	}

	// 3) 树浏览：根的直接子节点 ≥5（tech/product/market/hr/groups）
	tree := e.do(t, "GET", "/api/v1/tree?base="+root, nil, 200)
	kids, _ := tree["children"].([]any)
	if len(kids) < 5 {
		t.Fatalf("根子节点 = %d，应 ≥5: %+v", len(kids), kids)
	}

	// 4) 读取条目（含 entryCSN 与属性）
	dn := "uid=zhangwei,ou=tech," + root
	entry := e.do(t, "GET", "/api/v1/entry?dn="+dn, nil, 200)
	csn, _ := entry["entryCSN"].(string)
	if csn == "" {
		t.Fatal("条目缺 entryCSN")
	}
	attrs, _ := entry["attrs"].(map[string]any)
	if attrs["mail"] == nil {
		t.Fatal("条目缺 mail 属性")
	}

	// 5) 过期 CSN 提交 → 409 冲突（与 LAM 并存的乐观并发控制）
	e.do(t, "PUT", "/api/v1/entry", map[string]any{
		"dn": dn, "expectedCSN": "stale-csn",
		"changes": []map[string]any{{"op": "replace", "attr": "mobile", "vals": []string{"139-0000-0001"}}},
	}, 409)

	// 6) 正确 CSN 提交 → 200 且 CSN 变化
	updated := e.do(t, "PUT", "/api/v1/entry", map[string]any{
		"dn": dn, "expectedCSN": csn,
		"changes": []map[string]any{{"op": "replace", "attr": "mobile", "vals": []string{"139-0000-0001"}}},
	}, 200)
	if updated["entryCSN"] == csn {
		t.Error("修改后 entryCSN 应变化")
	}

	// 7) 搜索（分页路径）
	res := e.do(t, "GET", "/api/v1/search?base="+root+"&filter=(objectClass=inetOrgPerson)&attr=cn&attr=uid", nil, 200)
	if total, _ := res["total"].(float64); total < 5 {
		t.Errorf("人员搜索 total = %v，应 ≥5", res["total"])
	}

	// 8) 移动（modrdn）：李娜 调到产品设计部
	moved := e.do(t, "POST", "/api/v1/entry/move", map[string]any{
		"dn": "uid=lina,ou=tech," + root, "newParent": "ou=product," + root,
	}, 200)
	if moved["dn"] != "uid=lina,ou=product,"+root {
		t.Errorf("移动结果 = %v", moved["dn"])
	}

	// 9) schema 可读
	sm := e.do(t, "GET", "/api/v1/schema", nil, 200)
	if _, ok := sm["objectClasses"]; !ok {
		t.Error("schema 响应缺 objectClasses")
	}

	// 10) 审计落盘：登录（成功+失败）/修改/移动/冲突都在，且无密码泄漏
	auditContent(t)
}

func auditContent(t *testing.T) {
	t.Helper()
	b, err := os.ReadFile(auditFile)
	if err != nil {
		t.Fatal(err)
	}
	s := string(b)
	for _, want := range []string{`"op":"login"`, `"op":"modify"`, `"op":"move"`, `"result":"fail"`} {
		if !strings.Contains(s, want) {
			t.Errorf("审计日志缺少 %s", want)
		}
	}
	if strings.Contains(s, "admin123") || strings.Contains(s, "错的") {
		t.Error("审计日志泄漏了密码明文！")
	}
}

// TestIntegrationBindAccount 帮助中心 · 对接指引（R12）：
// 只读账号创建（默认自动建 ou=services / 指定已有位置 / 重名 / 非法输入）+ 凭据自测。
func TestIntegrationBindAccount(t *testing.T) {
	e := newEnv(t)
	e.do(t, "POST", "/api/v1/auth/login", map[string]string{"bindDN": "cn=admin," + root, "password": "admin123"}, 200)

	// 1) 默认位置：自动创建 ou=services，账号落 cn=<name>,ou=services,<base>
	out := e.do(t, "POST", "/api/v1/integration/bindacct", map[string]string{"name": "jumphub"}, 201)
	if out["dn"] != "cn=jumphub,ou=services,"+root {
		t.Fatalf("默认位置 DN = %v", out["dn"])
	}
	pw, _ := out["password"].(string)
	if len(pw) < 12 {
		t.Fatalf("初始密码过短: %q", pw)
	}

	// 2) 条目真实存在（树接口可见）且不含 userPassword 泄漏
	tree := e.do(t, "GET", "/api/v1/tree?containers=1", nil, 200)
	if !strings.Contains(fmt.Sprint(tree), "ou=services") {
		t.Error("自动创建的 ou=services 未出现在目录树")
	}
	ent := e.do(t, "GET", "/api/v1/entry?dn="+urlQueryEscape("cn=jumphub,ou=services,"+root), nil, 200)
	if strings.Contains(fmt.Sprint(ent), "userPassword") || strings.Contains(fmt.Sprint(ent), pw) {
		t.Error("对接账号条目泄漏了 userPassword")
	}

	// 3) 凭据自测：正确密码 ok=true；错误密码 ok=false（HTTP 仍 200）
	r1 := e.do(t, "POST", "/api/v1/integration/bindtest", map[string]string{"dn": "cn=jumphub,ou=services," + root, "password": pw}, 200)
	if r1["ok"] != true {
		t.Fatalf("正确密码自测失败: %v", r1)
	}
	r2 := e.do(t, "POST", "/api/v1/integration/bindtest", map[string]string{"dn": "cn=jumphub,ou=services," + root, "password": "wrong-pass"}, 200)
	if r2["ok"] != false {
		t.Fatalf("错误密码自测应 ok=false: %v", r2)
	}

	// 4) 重名 → 409
	e.do(t, "POST", "/api/v1/integration/bindacct", map[string]string{"name": "jumphub"}, 409)

	// 5) 指定已有位置（业务 OU）
	out5 := e.do(t, "POST", "/api/v1/integration/bindacct", map[string]string{"name": "reader", "parent": "ou=tech," + root}, 201)
	if out5["dn"] != "cn=reader,ou=tech,"+root {
		t.Fatalf("指定位置 DN = %v", out5["dn"])
	}
	// 二次创建不再触发自动建 OU（services 已存在），同名不同位置成功
	out5b := e.do(t, "POST", "/api/v1/integration/bindacct", map[string]string{"name": "jumphub", "parent": "ou=tech," + root}, 201)
	if out5b["dn"] != "cn=jumphub,ou=tech,"+root {
		t.Fatalf("同名不同位置 DN = %v", out5b["dn"])
	}

	// 6) 位置不存在 → 404/400；Base DN 之外 → 400；非法名 → 400
	e.do(t, "POST", "/api/v1/integration/bindacct", map[string]string{"name": "ghost", "parent": "ou=nop," + root}, 404)
	e.do(t, "POST", "/api/v1/integration/bindacct", map[string]string{"name": "outside", "parent": "ou=x,dc=other,dc=com"}, 400)
	for _, bad := range []string{"1bad", "bad name", "剑", ""} {
		e.do(t, "POST", "/api/v1/integration/bindacct", map[string]string{"name": bad}, 400)
	}

	// 7) 审计：创建与自测均有记录，且不落密码
	b, _ := os.ReadFile(auditFile)
	s := string(b)
	if !strings.Contains(s, `"op":"create-bindacct"`) || !strings.Contains(s, `"op":"bind-test"`) {
		t.Error("审计日志缺少 create-bindacct / bind-test")
	}
	if strings.Contains(s, pw) {
		t.Error("审计日志泄漏了对接账号密码！")
	}
}

func urlQueryEscape(s string) string {
	return strings.ReplaceAll(s, " ", "%20")
}
