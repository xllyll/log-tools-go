package config

import (
	"encoding/json"
	"fmt"
	"io/ioutil"

	"github.com/spf13/viper"
)

type Config struct {
	Server    ServerConfig      `mapstructure:"server"`
	Storage   StorageConfig     `mapstructure:"storage"`
	LogLevels map[string]string `mapstructure:"log_levels"`
	Filters   FilterConfig      `mapstructure:"filters"`
	AiConfig  AiConfig          `mapstructure:"ai"`
}

type ServerConfig struct {
	Port int    `mapstructure:"port"`
	Host string `mapstructure:"host"`
}

type StorageConfig struct {
	UploadDir    string `mapstructure:"upload_dir"`
	LogDir       string `mapstructure:"log_dir"`
	MaxFileSize  int64  `mapstructure:"max_file_size"`
	DatabasePath string `mapstructure:"database_path"`
}

type FilterConfig struct {
	DefaultLevels   []string `mapstructure:"default_levels"`
	ExcludePatterns []string `mapstructure:"exclude_patterns"`
	IncludePatterns []string `mapstructure:"include_patterns"`
}

type AiConfig struct {
	ApiKey string `mapstructure:"api_key"`
	Model  string `mapstructure:"model"`
}

// 新增：项目规则结构体
type LogParseRule struct {
	Timestamp       string `json:"timestamp"`
	TimestampFormat string `json:"timestamp_format"`
	Level           string `json:"level"`
	Thread          string `json:"thread"`
	Class           string `json:"class"`
	ClassLine       string `json:"class_line"`
	Message         string `json:"message"`
}

type LogProjectRule struct {
	ProjectName string       `json:"project_name"`
	Rule        LogParseRule `json:"rule"`
}

var AppConfig *Config
var ProjectRules []LogProjectRule

func LoadConfig() error {
	viper.SetConfigName("conf")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("./config")
	viper.AddConfigPath(".")

	if err := viper.ReadInConfig(); err != nil {
		return fmt.Errorf("读取配置文件失败: %w", err)
	}

	AppConfig = &Config{}
	if err := viper.Unmarshal(AppConfig); err != nil {
		return fmt.Errorf("解析配置文件失败: %w", err)
	}

	return nil
}

func GetConfig() *Config {
	return AppConfig
}

// 新增：加载项目规则
func LoadProjectRules(path string) error {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, &ProjectRules)
}

func GetRuleByProjectName(name string) *LogParseRule {
	for _, pr := range ProjectRules {
		if pr.ProjectName == name {
			return &pr.Rule
		}
	}
	return nil
}
