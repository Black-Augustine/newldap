// NewLDAP 目录管理台服务入口。
// 用法：
//
//	newldap-server [--addr :8080] [--mock]
//
// --mock：内置示例目录（进程内 LDAP 替身），无需任何外部依赖即可演示。
// 正式模式：通过环境变量 / NEWLDAP_CONFIG 指定目录连接（见 internal/config）。
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"newldap/internal/api"
	"newldap/internal/audit"
	"newldap/internal/auth"
	"newldap/internal/config"
	"newldap/internal/ldapclient"
	"newldap/internal/ldaptest"
)

func main() {
	var (
		addr = flag.String("addr", "", "监听地址，默认取 NEWLDAP_ADDR 或 :8080")
		mock = flag.Bool("mock", false, "使用内置示例目录（演示模式）")
	)
	flag.Parse()

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("加载配置失败: %v", err)
	}
	if *addr != "" {
		cfg.Server.Addr = *addr
	}
	if *mock {
		cfg.Server.Mock = true
	}

	// 演示模式：启动进程内目录替身并指向它
	var mockConn *ldapclient.Client
	if cfg.Server.Mock {
		// NEWLDAP_MOCK_ADDR 固定内置目录监听地址（如 127.0.0.1:3890），
		// 便于外部 LDAP 客户端（phpLDAPadmin 等）交叉验证；缺省随机端口。
		srv, err := ldaptest.StartAt(true, os.Getenv("NEWLDAP_MOCK_ADDR"))
		if err != nil {
			log.Fatalf("启动内置目录失败: %v", err)
		}
		cfg.Profile.URL = srv.Addr()
		cfg.Profile.BaseDN = "dc=example,dc=cn"
		log.Printf("[mock] 内置目录已启动：%s（管理员 cn=admin,dc=example,dc=cn / admin123）", srv.Addr())
		mockConn, err = ldapclient.Dial(ldapclient.Options{URL: srv.Addr()})
		if err != nil {
			log.Fatalf("连接内置目录失败: %v", err)
		}
		_ = mockConn.Bind("cn=admin,dc=example,dc=cn", "admin123")
	}

	aud, err := audit.New(cfg.Server.AuditFile)
	if err != nil {
		log.Fatalf("初始化审计日志失败: %v", err)
	}
	defer aud.Close()
	aud.Log(audit.Event{Op: "startup", Result: "ok", Detail: "addr=" + cfg.Server.Addr + " url=" + cfg.Profile.URL})

	sessions := auth.NewStore(cfg.Server.SessionTTL)

	deps := &api.Deps{
		Cfg: cfg, Sessions: sessions, Audit: aud, MockLDAP: mockConn,
	}
	apiHandler := api.New(deps)

	mux := http.NewServeMux()
	mux.Handle("/api/", apiHandler)
	mux.Handle("/healthz", apiHandler)
	mux.Handle("/", spaHandler())

	srv := &http.Server{
		Addr:              cfg.Server.Addr,
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
	}

	go func() {
		fmt.Printf("\n  NewLDAP 目录管理台\n  ─────────────────────────────\n")
		fmt.Printf("  监听地址   http://localhost%s\n", displayAddr(cfg.Server.Addr))
		fmt.Printf("  目录服务器 %s\n", cfg.Profile.URL)
		if cfg.Server.Mock {
			fmt.Printf("  演示账号   cn=admin,dc=example,dc=cn / admin123\n")
		}
		fmt.Printf("  审计日志   %s\n\n", cfg.Server.AuditFile)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("HTTP 服务退出: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop
	fmt.Println("\n正在退出…")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = srv.Shutdown(ctx)
}

func displayAddr(addr string) string {
	if addr == "" || addr[0] == ':' {
		return addr
	}
	return addr
}
