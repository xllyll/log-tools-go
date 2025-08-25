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

// LogParseRule 新增：项目规则结构体
type LogParseRule struct {
	Timestamp       string `json:"timestamp"`        // 时间正则表达式
	TimestampFormat string `json:"timestamp_format"` // 时间格式
	Process         string `json:"process"`          // 进程正则表达式
	Thread          string `json:"thread"`           // 线程正则表达式
	Level           string `json:"level"`            // 日志级别正则表达式
	Module          string `json:"module"`           // 模块名正则表达式
	Class           string `json:"class"`            // 类名正则表达式
	ClassLine       string `json:"class_line"`       // 类方法行号正则表达式
	Tag             string `json:"tag"`              // 标签正则表达式
	Message         string `json:"message"`          // 日志内容正则表达式
}
type LogProjectKeyword struct {
	Keyword string `json:"keyword"`
	Desc    string `json:"desc"`
	Mode    string `json:"mode"`
	Color   string `json:"color"`
}
type LogProjectScene struct {
	Name     string              `json:"name"`
	Keywords []LogProjectKeyword `json:"keywords"`
}
type LogProjectModule struct {
	Name   string            `json:"name"`
	Scenes []LogProjectScene `json:"scenes"`
}

type LogProjectRule struct {
	ProjectName string             `json:"project_name"`
	Rule        LogParseRule       `json:"rule"`
	Modules     []LogProjectModule `json:"modules"`
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
