package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"
)

// Config stores the runtime settings for the Go backend.
type Config struct {
	HTTPAddr           string   `mapstructure:"HTTP_ADDR"`
	DatabasePath       string   `mapstructure:"DATABASE_PATH"`
	DatabaseType       string   `mapstructure:"DATABASE_TYPE"` // sqlite, mysql, postgres
	DatabaseHost       string   `mapstructure:"DATABASE_HOST"`
	DatabasePort       string   `mapstructure:"DATABASE_PORT"`
	DatabaseName       string   `mapstructure:"DATABASE_NAME"`
	DatabaseUser       string   `mapstructure:"DATABASE_USER"`
	DatabasePassword   string   `mapstructure:"DATABASE_PASSWORD"`
	StoragePath        string   `mapstructure:"STORAGE_PATH"`
	PublicBaseURL      string   `mapstructure:"PUBLIC_BASE_URL"`
	LegacyDSN          string   `mapstructure:"LEGACY_DSN"`
	FrontendDist       string   `mapstructure:"FRONTEND_DIST"`
	AllowRegistration  bool     `mapstructure:"ALLOW_REGISTRATION"`
	CORSAllowedOrigins []string `mapstructure:"CORS_ALLOWED_ORIGINS"`
	TrustedProxies     []string `mapstructure:"TRUSTED_PROXIES"`
	DemoMode           bool     `mapstructure:"DEMO_MODE"`
	// 默认不显示的管理端设置项（设置键，逗号分隔）
	HiddenSettings     []string `mapstructure:"HIDDEN_SETTINGS"`
	// 演示站配置
	SiteName         string `mapstructure:"SITE_NAME"`
	AdminUsername     string `mapstructure:"ADMIN_USERNAME"`
	AdminEmail        string `mapstructure:"ADMIN_EMAIL"`
	AdminPassword     string `mapstructure:"ADMIN_PASSWORD"`
	DemoUserUsername  string `mapstructure:"DEMO_USER_USERNAME"`
	DemoUserEmail     string `mapstructure:"DEMO_USER_EMAIL"`
	DemoUserPassword  string `mapstructure:"DEMO_USER_PASSWORD"`
	SkipInstall       bool   `mapstructure:"SKIP_INSTALL"`
}

// Load reads configuration from env variables and optional .env/.yaml files.
func Load() (Config, error) {
	viper.SetConfigName(".env")
	viper.SetConfigType("env")
	viper.AddConfigPath(".")

	// 设置默认值
	setDefaults()

	// 启用自动环境变量读取
	viper.AutomaticEnv()

	// 显式绑定所有需要的环境变量
	viper.BindEnv("HTTP_ADDR")
	viper.BindEnv("DATABASE_PATH")
	viper.BindEnv("DATABASE_TYPE")
	viper.BindEnv("DATABASE_HOST")
	viper.BindEnv("DATABASE_PORT")
	viper.BindEnv("DATABASE_NAME")
	viper.BindEnv("DATABASE_USER")
	viper.BindEnv("DATABASE_PASSWORD")
	viper.BindEnv("STORAGE_PATH")
	viper.BindEnv("PUBLIC_BASE_URL")
	viper.BindEnv("LEGACY_DSN")
	viper.BindEnv("FRONTEND_DIST")
	viper.BindEnv("ALLOW_REGISTRATION")
	viper.BindEnv("CORS_ALLOWED_ORIGINS")
	viper.BindEnv("TRUSTED_PROXIES")
	viper.BindEnv("DEMO_MODE")
	viper.BindEnv("HIDDEN_SETTINGS")
	// 演示站配置
	viper.BindEnv("SITE_NAME")
	viper.BindEnv("ADMIN_USERNAME")
	viper.BindEnv("ADMIN_EMAIL")
	viper.BindEnv("ADMIN_PASSWORD")
	viper.BindEnv("DEMO_USER_USERNAME")
	viper.BindEnv("DEMO_USER_EMAIL")
	viper.BindEnv("DEMO_USER_PASSWORD")
	viper.BindEnv("SKIP_INSTALL")

	_ = viper.ReadInConfig() // best-effort optional .env

	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return Config{}, fmt.Errorf("unmarshal config: %w", err)
	}

	cfg.CORSAllowedOrigins = parseCSVEnv(viper.GetString("CORS_ALLOWED_ORIGINS"))
	cfg.TrustedProxies = parseCSVEnv(viper.GetString("TRUSTED_PROXIES"))
	cfg.HiddenSettings = parseCSVEnv(viper.GetString("HIDDEN_SETTINGS"))

	if err := ensurePaths(&cfg); err != nil {
		return Config{}, err
	}

	return cfg, nil
}

func MustLoad() Config {
	cfg, err := Load()
	if err != nil {
		panic(err)
	}
	return cfg
}

func setDefaults() {
	viper.SetDefault("HTTP_ADDR", ":8080")
	viper.SetDefault("DATABASE_TYPE", "")
	viper.SetDefault("DATABASE_PATH", "")
	viper.SetDefault("DATABASE_HOST", "")
	viper.SetDefault("DATABASE_PORT", "")
	viper.SetDefault("DATABASE_NAME", "")
	viper.SetDefault("DATABASE_USER", "")
	viper.SetDefault("DATABASE_PASSWORD", "")
	viper.SetDefault("STORAGE_PATH", filepath.Join("storage", "uploads"))
	viper.SetDefault("PUBLIC_BASE_URL", "http://localhost:8080")
	viper.SetDefault("ALLOW_REGISTRATION", true)
	viper.SetDefault("CORS_ALLOWED_ORIGINS", "")
	viper.SetDefault("TRUSTED_PROXIES", "")
	viper.SetDefault("DEMO_MODE", false)
	viper.SetDefault("HIDDEN_SETTINGS", "")
	// 演示站配置默认值
	viper.SetDefault("SITE_NAME", "SkyImage Demo")
	viper.SetDefault("ADMIN_USERNAME", "demo_admin")
	viper.SetDefault("ADMIN_EMAIL", "demo@example.com")
	viper.SetDefault("ADMIN_PASSWORD", "DemoPass123!")
	viper.SetDefault("DEMO_USER_USERNAME", "demo_user")
	viper.SetDefault("DEMO_USER_EMAIL", "user@example.com")
	viper.SetDefault("DEMO_USER_PASSWORD", "UserPass123!")
	viper.SetDefault("SKIP_INSTALL", false)
}

func parseCSVEnv(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	items := make([]string, 0, len(parts))
	for _, part := range parts {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			items = append(items, trimmed)
		}
	}
	return items
}

func ensurePaths(cfg *Config) error {
	// 只有在明确选择 SQLite 且指定了路径后才创建数据库目录，避免安装前就生成数据库文件
	if cfg.DatabaseType == "sqlite" && cfg.DatabasePath != "" {
		if err := os.MkdirAll(filepath.Dir(cfg.DatabasePath), 0o755); err != nil {
			return fmt.Errorf("create database dir: %w", err)
		}
	}
	if err := os.MkdirAll(cfg.StoragePath, 0o755); err != nil {
		return fmt.Errorf("create storage dir: %w", err)
	}
	return nil
}

// SaveDatabaseEnv persists database connection settings into the local .env file.
// Empty values are written as empty strings so switching DB types clears stale fields.
func SaveDatabaseEnv(cfg Config) error {
	envPath := ".env"
	content := ""
	if existingContent, err := os.ReadFile(envPath); err == nil {
		content = string(existingContent)
	}

	updates := map[string]string{
		"DATABASE_TYPE":     strings.TrimSpace(cfg.DatabaseType),
		"DATABASE_PATH":     cfg.DatabasePath,
		"DATABASE_HOST":     cfg.DatabaseHost,
		"DATABASE_PORT":     cfg.DatabasePort,
		"DATABASE_NAME":     cfg.DatabaseName,
		"DATABASE_USER":     cfg.DatabaseUser,
		"DATABASE_PASSWORD": cfg.DatabasePassword,
	}

	for key, value := range updates {
		sanitized, err := sanitizeEnvValue(value)
		if err != nil {
			return fmt.Errorf("invalid %s: %w", key, err)
		}
		pattern := key + "="
		lineValue := quoteEnvValue(sanitized)
		if strings.Contains(content, pattern) {
			lines := strings.Split(content, "\n")
			for i, line := range lines {
				if strings.HasPrefix(strings.TrimSpace(line), pattern) {
					lines[i] = fmt.Sprintf("%s=%s", key, lineValue)
				}
			}
			content = strings.Join(lines, "\n")
		} else {
			if content != "" && !strings.HasSuffix(content, "\n") {
				content += "\n"
			}
			content += fmt.Sprintf("%s=%s\n", key, lineValue)
		}
	}

	if err := os.WriteFile(envPath, []byte(content), 0o644); err != nil {
		return fmt.Errorf("write %s: %w (若在 Docker 中部署，请确认宿主机 .env 是文件而不是目录)", envPath, err)
	}
	return nil
}

func sanitizeEnvValue(raw string) (string, error) {
	if strings.Contains(raw, "\x00") {
		return "", fmt.Errorf("contains null byte")
	}
	if strings.Contains(raw, "\n") || strings.Contains(raw, "\r") {
		return "", fmt.Errorf("contains newline")
	}
	return raw, nil
}

func quoteEnvValue(raw string) string {
	escaped := strings.ReplaceAll(raw, `\`, `\\`)
	escaped = strings.ReplaceAll(escaped, `"`, `\"`)
	return `"` + escaped + `"`
}
