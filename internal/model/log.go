package model

import (
	"time"
)

type LogEntry struct {
	ID        string    `json:"id"`         // 日志ID
	LogTime   time.Time `json:"log_time"`   // 日志时间
	SaveTime  time.Time `json:"save_time"`  // 日志保存时间
	Level     string    `json:"level"`      // 日志级别
	Module    string    `json:"module"`     // 模块名称
	Process   *string   `json:"process"`    // 进程名称
	Thread    *string   `json:"thread"`     // 线程名称
	Class     *string   `json:"class"`      // 类名
	ClassLine *string   `json:"class_line"` // 类行号
	Tag       *string   `json:"tag"`        // 标签
	Message   string    `json:"message"`    // 日志消息
	Content   string    `json:"content"`    // 日志内容[原始内容]
	Source    string    `json:"source"`     // 日志来源
	Line      int       `json:"line"`       // 日志行号
	Color     string    `json:"color"`      // 日志颜色
}

type LogFile struct {
	ID       string     `json:"id"`
	Name     string     `json:"name"`
	Size     int64      `json:"size"`
	UploadAt time.Time  `json:"upload_at"`
	Entries  []LogEntry `json:"entries"`
	Total    int        `json:"total"`
}

type LogFilter struct {
	Levels    []string   `json:"levels"`
	Module    string     `json:"module"`
	Keywords  []string   `json:"keywords"`
	StartTime *time.Time `json:"start_time"`
	EndTime   *time.Time `json:"end_time"`
	Source    string     `json:"source"`
	Limit     int        `json:"limit"`
	Offset    int        `json:"offset"`
}

type LogStats struct {
	TotalEntries int            `json:"total_entries"`
	LevelCounts  map[string]int `json:"level_counts"`
	TimeRange    struct {
		Start time.Time `json:"start"`
		End   time.Time `json:"end"`
	} `json:"time_range"`
}

type UploadResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	FileID  string `json:"file_id,omitempty"`
	Error   string `json:"error,omitempty"`
}

type LogResponse struct {
	Success bool       `json:"success"`
	Data    []LogEntry `json:"data,omitempty"`
	Stats   LogStats   `json:"stats,omitempty"`
	Error   string     `json:"error,omitempty"`
}
type R struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
}
