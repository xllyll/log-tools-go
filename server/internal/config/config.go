package config

import (
	"fmt"

	"github.com/spf13/viper"
)

type Config struct {
	Server   ServerConfig   `mapstructure:"server"`
	Database DatabaseConfig `mapstructure:"database"`
	Storage  StorageConfig  `mapstructure:"storage"`
	Ingest   IngestConfig   `mapstructure:"ingest"`
	Jira     JiraConfig     `mapstructure:"jira"`
}

type ServerConfig struct {
	Host string `mapstructure:"host"`
	Port int    `mapstructure:"port"`
}

type DatabaseConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	User     string `mapstructure:"user"`
	Password string `mapstructure:"password"`
	DBName   string `mapstructure:"dbname"`
	SSLMode  string `mapstructure:"sslmode"`
}

type StorageConfig struct {
	UploadDir            string `mapstructure:"upload_dir"`
	MaxFileSize          int64  `mapstructure:"max_file_size"`
	RetentionDays        int    `mapstructure:"retention_days"`
	CleanupIntervalHours int    `mapstructure:"cleanup_interval_hours"`
}

type IngestConfig struct {
	BatchSize   int `mapstructure:"batch_size"`
	WorkerCount int `mapstructure:"worker_count"`
}

type JiraConfig struct {
	Enabled  bool   `mapstructure:"enabled"`
	BaseURL  string `mapstructure:"base_url"`
	Email    string `mapstructure:"email"`
	APIToken string `mapstructure:"api_token"`
}

var global *Config

func Load(path string) error {
	v := viper.New()
	v.SetConfigFile(path)
	if err := v.ReadInConfig(); err != nil {
		return fmt.Errorf("read config: %w", err)
	}
	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return fmt.Errorf("unmarshal config: %w", err)
	}
	if cfg.Ingest.BatchSize <= 0 {
		cfg.Ingest.BatchSize = 500
	}
	if cfg.Ingest.WorkerCount <= 0 {
		cfg.Ingest.WorkerCount = 4
	}
	if !v.IsSet("storage.retention_days") {
		cfg.Storage.RetentionDays = 30
	}
	if cfg.Storage.CleanupIntervalHours <= 0 {
		cfg.Storage.CleanupIntervalHours = 24
	}
	global = &cfg
	return nil
}

func Get() *Config {
	return global
}

func (d DatabaseConfig) DSN() string {
	return fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		d.Host, d.Port, d.User, d.Password, d.DBName, d.SSLMode,
	)
}
