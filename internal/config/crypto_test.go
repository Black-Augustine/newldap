package config

import (
	"strings"
	"testing"
)

func TestEncryptDecryptRoundTrip(t *testing.T) {
	key := SecretKeyForTest("my-secret-key")
	enc, err := EncryptString(key, "P@ssw0rd!中文密码")
	if err != nil {
		t.Fatal(err)
	}
	if !IsEncrypted(enc) {
		t.Fatalf("密文缺少前缀: %s", enc)
	}
	if strings.Contains(enc, "P@ssw0rd") {
		t.Fatal("密文包含明文！")
	}
	plain, err := DecryptString(key, enc)
	if err != nil {
		t.Fatal(err)
	}
	if plain != "P@ssw0rd!中文密码" {
		t.Errorf("解密结果 = %q", plain)
	}
}

func TestDecryptWrongKey(t *testing.T) {
	enc, _ := EncryptString(SecretKeyForTest("key-a"), "secret")
	if _, err := DecryptString(SecretKeyForTest("key-b"), enc); err == nil {
		t.Fatal("错误密钥解密应失败")
	} else if !strings.Contains(err.Error(), "SECRET_KEY") {
		t.Errorf("错误信息应提示密钥不一致: %v", err)
	}
}

func TestDecryptGarbage(t *testing.T) {
	key := SecretKeyForTest("k")
	for _, bad := range []string{"", "plaintext", "enc:v1:!!!not-base64!!!", "enc:v1:QUJD"} {
		if _, err := DecryptString(key, bad); err == nil {
			t.Errorf("垃圾输入 %q 不应解密成功", bad)
		}
	}
}

// SecretKeyForTest 用任意字符串派生 32 字节密钥（测试与内部共用）。
func SecretKeyForTest(s string) []byte { return deriveKey(s) }
