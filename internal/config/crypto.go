// 凭据加密：AES-256-GCM（架构 D7）。
// 密钥来源：环境变量 NEWLDAP_SECRET_KEY（64 位 hex 直接用；任意字符串则 SHA-256 派生）。
// 密文格式：enc:v1:<base64(nonce|ciphertext)>。
package config

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
)

const encPrefix = "enc:v1:"

// SecretKey 返回当前生效的加密密钥（32 字节）；未配置时返回 nil。
func SecretKey() []byte {
	s := secretKeyEnv()
	if s == "" {
		return nil
	}
	return deriveKey(s)
}

// 默认直接读环境变量；测试可用 SetSecretKeySource 注入/清除。
var secretKeyEnv = func() string { return os.Getenv("NEWLDAP_SECRET_KEY") }

// SetSecretKeySource 替换密钥来源（测试注入用；传空串表示无密钥）。
func SetSecretKeySource(v string) {
	secretKeyEnv = func() string { return v }
}

func deriveKey(s string) []byte {
	if len(s) == 64 {
		if b, err := hex.DecodeString(s); err == nil && len(b) == 32 {
			return b
		}
	}
	h := sha256.Sum256([]byte(s))
	return h[:]
}

// EncryptString 用 key 加密明文。
func EncryptString(key []byte, plain string) (string, error) {
	if len(key) != 32 {
		return "", errors.New("密钥长度必须为 32 字节")
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	ct := gcm.Seal(nil, nonce, []byte(plain), nil)
	return encPrefix + base64.StdEncoding.EncodeToString(append(nonce, ct...)), nil
}

// DecryptString 解密 EncryptString 的输出；密钥错误返回明确错误（而非崩溃）。
func DecryptString(key []byte, enc string) (string, error) {
	if !strings.HasPrefix(enc, encPrefix) {
		return "", fmt.Errorf("密文格式不正确（缺少 %s 前缀）", encPrefix)
	}
	raw, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(enc, encPrefix))
	if err != nil {
		return "", err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	ns := gcm.NonceSize()
	if len(raw) < ns+1 {
		return "", errors.New("密文长度不合法")
	}
	plain, err := gcm.Open(nil, raw[:ns], raw[ns:], nil)
	if err != nil {
		return "", errors.New("解密失败：NEWLDAP_SECRET_KEY 与加密时不一致")
	}
	return string(plain), nil
}

// IsEncrypted 判断字符串是否为本包密文格式。
func IsEncrypted(s string) bool { return strings.HasPrefix(s, encPrefix) }

// MaskSecret 输出给前端的密码形态。
func MaskSecret(s string) string {
	if s == "" {
		return ""
	}
	return "********"
}
