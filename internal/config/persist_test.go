package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSaveLoadEncryptedRoundTrip(t *testing.T) {
	SetSecretKeySource("unit-test-key")
	t.Cleanup(func() { SetSecretKeySource("") })

	dir := t.TempDir()
	f := filepath.Join(dir, "config.yaml")

	orig := defaultConfig()
	orig.Profile = Profile{
		Name: "示例公司", URL: "ldap://127.0.0.1:3891", BindDN: "cn=admin,dc=example,dc=cn",
		BindPassword: "SuperSecret123!", BaseDN: "dc=example,dc=cn",
	}
	plaintext, err := orig.SaveToFile(f)
	if err != nil {
		t.Fatal(err)
	}
	if plaintext {
		t.Fatal("应走加密保存（密钥已注入）")
	}
	// 落盘内容不含明文密码
	b, _ := os.ReadFile(f)
	if strings.Contains(string(b), "SuperSecret123!") {
		t.Fatal("配置文件泄漏明文密码！")
	}
	if !strings.Contains(string(b), "enc:v1:") {
		t.Fatal("未使用加密格式")
	}

	// 重新加载并解密
	t.Setenv("NEWLDAP_CONFIG", f)
	got, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if got.Profile.BindPassword != "SuperSecret123!" {
		t.Errorf("解密后密码 = %q", got.Profile.BindPassword)
	}
	if got.Profile.Name != "示例公司" || got.Profile.BaseDN != "dc=example,dc=cn" {
		t.Errorf("档案字段回读不一致: %+v", got.Profile)
	}
}

func TestSavePlaintextFallbackWithoutKey(t *testing.T) {
	SetSecretKeySource("")
	t.Cleanup(func() { SetSecretKeySource("") })

	dir := t.TempDir()
	f := filepath.Join(dir, "config.yaml")
	orig := defaultConfig()
	orig.Profile.BindPassword = "plain-pass"
	plaintext, err := orig.SaveToFile(f)
	if err != nil {
		t.Fatal(err)
	}
	if !plaintext {
		t.Fatal("无密钥时应明文保存并警告")
	}
	b, _ := os.ReadFile(f)
	if !strings.Contains(string(b), "警告") {
		t.Error("明文保存应写入警告注释")
	}
}
