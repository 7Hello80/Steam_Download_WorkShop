package config

import (
	"log"
	"os"
	"strconv"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Port                string `yaml:"port" json:"port"`
	JWTSecret           string `yaml:"jwt_secret" json:"-"`
	GitHubClientID      string `yaml:"github_client_id" json:"-"`
	GitHubClientSecret  string `yaml:"github_client_secret" json:"-"`
	GitHubRedirectURL   string `yaml:"github_redirect_url" json:"-"`
	FrontendURL         string `yaml:"frontend_url" json:"-"`
	PaymentMode         string `yaml:"payment_mode" json:"payment_mode"`
	MaxWorkers          int    `yaml:"max_workers" json:"max_workers"`
	FileTTLHours        int    `yaml:"file_ttl_hours" json:"file_ttl_hours"`
	DatabasePath        string `yaml:"database_path" json:"-"`
	DepotDownloaderPath string `yaml:"depot_downloader_path" json:"-"`
	OutputDir           string `yaml:"output_dir" json:"-"`
	StaticDir           string `yaml:"static_dir" json:"-"`

	// Encryption
	AESKey string `yaml:"aes_key" json:"-"`

	// Payment
	AlipayAppID      string `yaml:"alipay_app_id" json:"-"`
	AlipayPrivateKey string `yaml:"alipay_private_key" json:"-"`
	WechatAppID      string `yaml:"wechat_app_id" json:"-"`
	WechatMchID      string `yaml:"wechat_mch_id" json:"-"`
	WechatAPIKey     string `yaml:"wechat_api_key" json:"-"`

	// SMTP Email
	SMTPHost     string `yaml:"smtp_host" json:"-"`
	SMTPPort     string `yaml:"smtp_port" json:"-"`
	SMTPUsername string `yaml:"smtp_username" json:"-"`
	SMTPPassword string `yaml:"smtp_password" json:"-"`
	SMTPFrom     string `yaml:"smtp_from" json:"-"`
	SMTPFromName string `yaml:"smtp_from_name" json:"-"`

	// Pricing
	PricePerDownload float64 `yaml:"price_per_download" json:"price_per_download"`
}

func defaultConfig() *Config {
	return &Config{
		Port:                "8086",
		JWTSecret:           "steam-download-tool-jwt-secret-change-me",
		GitHubRedirectURL:   "http://localhost:8086/api/auth/github/callback",
		FrontendURL:         "http://localhost:8086",
		PaymentMode:         "free",
		MaxWorkers:          2,
		FileTTLHours:        72,
		DatabasePath:        "./data/steam-download.db",
		DepotDownloaderPath: "./DepotDownloader",
		OutputDir:           "./output",
		StaticDir:           "./static",
		AESKey:              "",
		SMTPPort:            "587",
		SMTPFromName:        "Steam Download Tool",
		PricePerDownload:    0,
	}
}

func Load() *Config {
	cfg := defaultConfig()

	// Determine config file path
	configPath := os.Getenv("CONFIG_PATH")
	if configPath == "" {
		configPath = "config.yaml"
	}

	// Read YAML file
	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			log.Printf("Config file %s not found, using defaults (and env overrides)", configPath)
		} else {
			log.Printf("Warning: failed to read config file %s: %v", configPath, err)
		}
	} else {
		if err := yaml.Unmarshal(data, cfg); err != nil {
			log.Printf("Warning: failed to parse config file %s: %v", configPath, err)
		} else {
			log.Printf("Config loaded from %s", configPath)
		}
	}

	// Environment variable overrides (takes precedence over config file)
	applyEnvOverrides(cfg)

	return cfg
}

func applyEnvOverrides(cfg *Config) {
	if v := os.Getenv("PORT"); v != "" {
		cfg.Port = v
	}
	if v := os.Getenv("JWT_SECRET"); v != "" {
		cfg.JWTSecret = v
	}
	if v := os.Getenv("GITHUB_CLIENT_ID"); v != "" {
		cfg.GitHubClientID = v
	}
	if v := os.Getenv("GITHUB_CLIENT_SECRET"); v != "" {
		cfg.GitHubClientSecret = v
	}
	if v := os.Getenv("GITHUB_REDIRECT_URL"); v != "" {
		cfg.GitHubRedirectURL = v
	}
	if v := os.Getenv("FRONTEND_URL"); v != "" {
		cfg.FrontendURL = v
	}
	if v := os.Getenv("PAYMENT_MODE"); v != "" {
		cfg.PaymentMode = v
	}
	if v := os.Getenv("MAX_WORKERS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			cfg.MaxWorkers = n
		} else {
			log.Printf("Warning: invalid MAX_WORKERS value '%s', using default", v)
		}
	}
	if v := os.Getenv("FILE_TTL_HOURS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			cfg.FileTTLHours = n
		} else {
			log.Printf("Warning: invalid FILE_TTL_HOURS value '%s', using default", v)
		}
	}
	if v := os.Getenv("DATABASE_PATH"); v != "" {
		cfg.DatabasePath = v
	}
	if v := os.Getenv("DEPOT_DOWNLOADER_PATH"); v != "" {
		cfg.DepotDownloaderPath = v
	}
	if v := os.Getenv("OUTPUT_DIR"); v != "" {
		cfg.OutputDir = v
	}
	if v := os.Getenv("STATIC_DIR"); v != "" {
		cfg.StaticDir = v
	}
	if v := os.Getenv("AES_KEY"); v != "" {
		cfg.AESKey = v
	}
	if v := os.Getenv("SMTP_HOST"); v != "" {
		cfg.SMTPHost = v
	}
	if v := os.Getenv("SMTP_PORT"); v != "" {
		cfg.SMTPPort = v
	}
	if v := os.Getenv("SMTP_USERNAME"); v != "" {
		cfg.SMTPUsername = v
	}
	if v := os.Getenv("SMTP_PASSWORD"); v != "" {
		cfg.SMTPPassword = v
	}
	if v := os.Getenv("SMTP_FROM"); v != "" {
		cfg.SMTPFrom = v
	}
	if v := os.Getenv("SMTP_FROM_NAME"); v != "" {
		cfg.SMTPFromName = v
	}
}

// GenerateDefaultConfig writes the default config.yaml to disk.
func GenerateDefaultConfig(path string) error {
	cfg := defaultConfig()
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}
