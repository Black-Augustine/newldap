package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLoadDefaults(t *testing.T) {
	for _, k := range []string{"LDAP_URL", "LDAP_BIND_DN", "LDAP_BASE_DN", "NEWLDAP_CONFIG", "NEWLDAP_ADDR"} {
		os.Unsetenv(k)
	}
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Profile.URL != "ldap://127.0.0.1:389" {
		t.Errorf("默认 URL = %q", cfg.Profile.URL)
	}
	if cfg.Server.SessionTTL != 30*time.Minute {
		t.Errorf("默认会话 TTL = %v", cfg.Server.SessionTTL)
	}
}

func TestEnvOverrides(t *testing.T) {
	t.Setenv("LDAP_URL", "ldaps://ldap.example.cn:636")
	t.Setenv("LDAP_BIND_DN", "cn=admin,dc=example,dc=cn")
	t.Setenv("LDAP_BIND_PASSWORD", "s3cret")
	t.Setenv("LDAP_BASE_DN", "dc=example,dc=cn")
	t.Setenv("NEWLDAP_ADDR", ":9090")
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Profile.URL != "ldaps://ldap.example.cn:636" {
		t.Errorf("URL = %q", cfg.Profile.URL)
	}
	if cfg.Profile.BindDN != "cn=admin,dc=example,dc=cn" {
		t.Errorf("BindDN = %q", cfg.Profile.BindDN)
	}
	if cfg.Server.Addr != ":9090" {
		t.Errorf("Addr = %q", cfg.Server.Addr)
	}
}

func TestYamlThenEnvPrecedence(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "config.yaml")
	yaml := `
server:
  addr: ":7070"
ldap:
  name: 演示环境
  url: ldap://from-yaml:389
  bind_dn: cn=admin,dc=yaml,dc=cn
  bind_password: yamlpass
  base_dn: dc=yaml,dc=cn
`
	if err := os.WriteFile(f, []byte(yaml), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("NEWLDAP_CONFIG", f)
	t.Setenv("LDAP_URL", "ldap://from-env:389") // 环境变量应压过 YAML

	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Profile.Name != "演示环境" {
		t.Errorf("Name = %q", cfg.Profile.Name)
	}
	if cfg.Profile.URL != "ldap://from-env:389" {
		t.Errorf("URL 应为环境变量值，得到 %q", cfg.Profile.URL)
	}
	if cfg.Profile.BindPassword != "yamlpass" {
		t.Errorf("BindPassword = %q", cfg.Profile.BindPassword)
	}
	if cfg.Server.Addr != ":7070" {
		t.Errorf("Addr = %q", cfg.Server.Addr)
	}
}
