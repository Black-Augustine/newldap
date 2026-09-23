// Package config 加载服务配置：YAML 文件 + 环境变量（环境变量优先，供 compose 伴随形态零配置）。
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
)

// Profile 是一个目录连接档案。
// M0 说明：BindPassword 目前明文保存于配置文件，仅限开发/演示；
// 按《技术架构设计》D7，M1 里程碑将改为 AES-GCM 加密落盘（密钥来自 NEWLDAP_SECRET_KEY）。
type Profile struct {
	Name               string `yaml:"name" json:"name"`
	URL                string `yaml:"url" json:"url"` // ldap://host:port 或 ldaps://host:port
	StartTLS           bool   `yaml:"start_tls" json:"start_tls"`
	BindDN             string `yaml:"bind_dn" json:"bind_dn"`
	BindPassword       string `yaml:"bind_password" json:"-"`
	BaseDN             string `yaml:"base_dn" json:"base_dn"`
	InsecureSkipVerify bool   `yaml:"insecure_skip_verify" json:"insecure_skip_verify"`
}

type Server struct {
	Addr       string        `yaml:"addr"`
	ForceHTTPS bool          `yaml:"force_https"` // 改密入口强制 HTTPS（生产建议开启）
	Mock       bool          `yaml:"mock"`        // 内置测试目录（开发/演示用，进程内 LDAP 替身）
	AuditFile  string        `yaml:"audit_file"`
	SessionTTL time.Duration `yaml:"session_ttl"`
	PLDAURL    string        `yaml:"plda_url"` // 伴随部署的 phpLDAPadmin 地址（设置后界面显示跳转按钮，供交叉验证）
}

type Config struct {
	Server  Server  `yaml:"server"`
	Profile Profile `yaml:"ldap"`
	// Source 表示本次配置来源：env（环境变量生效）/ file（配置文件）/ default（内置默认）
	Source string `yaml:"-"`
}

// DefaultConfigFile 是配置文件的默认路径（连接向导保存档案时写入）。
const DefaultConfigFile = "data/config.yaml"

// ActiveConfigFile 返回当前生效的配置文件路径：优先 NEWLDAP_CONFIG，
// 与 Load 的读取路径保持一致（向导保存与启动加载必须指向同一文件）。
func ActiveConfigFile() string {
	if v := os.Getenv("NEWLDAP_CONFIG"); v != "" {
		return v
	}
	return DefaultConfigFile
}

func defaultConfig() *Config {
	return &Config{
		Server: Server{
			Addr:       ":8080",
			AuditFile:  "data/audit.jsonl",
			SessionTTL: 30 * time.Minute,
		},
		Profile: Profile{
			Name: "默认连接",
			URL:  "ldap://127.0.0.1:389",
		},
	}
}

// Load 按 默认值 → 配置文件（NEWLDAP_CONFIG 或 data/config.yaml）→ 环境变量
// 的优先级装配配置。配置文件中的 enc:v1: 密文会用 NEWLDAP_SECRET_KEY 解密。
func Load() (*Config, error) {
	cfg := defaultConfig()

	file := os.Getenv("NEWLDAP_CONFIG")
	if file == "" {
		file = DefaultConfigFile // 不存在则静默跳过（首次运行）
	}
	if _, err := os.Stat(file); err == nil {
		b, err := os.ReadFile(file)
		if err != nil {
			return nil, fmt.Errorf("读取配置文件 %s: %w", file, err)
		}
		if err := yaml.Unmarshal(b, cfg); err != nil {
			return nil, fmt.Errorf("解析配置文件 %s: %w", file, err)
		}
		cfg.Source = "file"
		// 解密已保存的 bind 密码
		if IsEncrypted(cfg.Profile.BindPassword) {
			key := SecretKey()
			if key == nil {
				return nil, fmt.Errorf("配置文件 %s 中的密码已加密，但未设置 NEWLDAP_SECRET_KEY", file)
			}
			plain, err := DecryptString(key, cfg.Profile.BindPassword)
			if err != nil {
				return nil, fmt.Errorf("%s（配置文件 %s）", err.Error(), file)
			}
			cfg.Profile.BindPassword = plain
		}
	}
	if cfg.Source == "" {
		cfg.Source = "default"
	}
	if v := os.Getenv("LDAP_URL"); v != "" {
		cfg.Profile.URL = v
		cfg.Source = "env"
	}
	if v := os.Getenv("LDAP_STARTTLS"); v == "1" || v == "true" || v == "yes" {
		cfg.Profile.StartTLS = true
	}
	if v := os.Getenv("LDAP_BIND_DN"); v != "" {
		cfg.Profile.BindDN = v
	}
	if v := os.Getenv("LDAP_BIND_PASSWORD"); v != "" {
		cfg.Profile.BindPassword = v
	}
	if v := os.Getenv("LDAP_BASE_DN"); v != "" {
		cfg.Profile.BaseDN = v
	}
	if v := os.Getenv("LDAP_PROFILE_NAME"); v != "" {
		cfg.Profile.Name = v
	}
	if v := os.Getenv("LDAP_INSECURE_SKIP_VERIFY"); v == "1" || v == "true" {
		cfg.Profile.InsecureSkipVerify = true
	}
	if v := os.Getenv("NEWLDAP_ADDR"); v != "" {
		cfg.Server.Addr = v
	}
	if v := os.Getenv("NEWLDAP_AUDIT_FILE"); v != "" {
		cfg.Server.AuditFile = v
	}
	if v := os.Getenv("NEWLDAP_FORCE_HTTPS"); v == "1" || v == "true" {
		cfg.Server.ForceHTTPS = true
	}
	if v := os.Getenv("NEWLDAP_MOCK"); v == "1" || v == "true" {
		cfg.Server.Mock = true
	}
	if v := os.Getenv("NEWLDAP_PLDA_URL"); v != "" {
		cfg.Server.PLDAURL = v
	}
	return cfg, nil
}

// SaveToFile 将配置写入 YAML 文件。
// BindPassword 优先加密落盘（设置 NEWLDAP_SECRET_KEY 时）；未设置密钥则
// 明文写入并在文件头注释中警告。返回是否以明文保存。
func (c *Config) SaveToFile(path string) (plaintextSaved bool, err error) {
	out := &Config{Server: c.Server, Profile: c.Profile}
	out.Server.Addr = c.Server.Addr
	out.Profile.BindPassword = ""
	if c.Profile.BindPassword != "" {
		key := SecretKey()
		if key != nil {
			enc, err := EncryptString(key, c.Profile.BindPassword)
			if err != nil {
				return false, err
			}
			out.Profile.BindPassword = enc
		} else {
			out.Profile.BindPassword = c.Profile.BindPassword
			plaintextSaved = true
		}
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return plaintextSaved, err
	}
	b, err := yaml.Marshal(out)
	if err != nil {
		return plaintextSaved, err
	}
	header := "# NewLDAP 配置文件（连接向导生成）\n"
	if plaintextSaved {
		header += "# 警告：bind 密码以明文保存！请设置 NEWLDAP_SECRET_KEY 环境变量后重新保存。\n"
	}
	return plaintextSaved, os.WriteFile(path, append([]byte(header), b...), 0o600)
}
