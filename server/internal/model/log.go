package model

import "time"

type LogEntry struct {
	ID        string    `json:"id"`
	FileID    string    `json:"file_id,omitempty"`
	LogTime   time.Time `json:"log_time"`
	Content   string    `json:"content"`
	Line      int       `json:"line"`
	Level     string    `json:"level"`
	Module    string    `json:"module"`
	Message   string    `json:"message"`
	Color     string    `json:"color"`
	SceneDesc string    `json:"scene_desc,omitempty"`
}

type LogFile struct {
	ID         string     `json:"id"`
	DeviceID   string     `json:"device_id"`
	Name       string     `json:"name"`
	Size       int64      `json:"size"`
	UploadAt   time.Time  `json:"upload_at"`
	Total      int        `json:"total"`
	Status     string     `json:"status"` // parsing | ready | failed
	StatusMsg  string     `json:"status_msg,omitempty"`
	Entries    []LogEntry `json:"-"`
}

type LogFilter struct {
	DeviceID      string
	FileID        string
	FileIDs       []string
	Keywords      []string
	SceneKeywords []string
	UseRegex      bool
	Limit         int
	Offset        int
	LineNumber    int
	Before        int
	After         int
}

type LogStats struct {
	TotalEntries int `json:"total_entries"`
}

type UploadResponse struct {
	Success bool     `json:"success"`
	Message string   `json:"message"`
	FileIDs []string `json:"file_ids,omitempty"`
	Error   string   `json:"error,omitempty"`
}

type APIResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Message string      `json:"message,omitempty"`
	Error   string      `json:"error,omitempty"`
}

type SceneConfig struct {
	Modules []SceneModule `json:"modules"`
}

type SceneModule struct {
	Name   string       `json:"name"`
	Scenes []SceneGroup `json:"scenes"`
}

type SceneGroup struct {
	Name     string         `json:"name"`
	Keywords []SceneKeyword `json:"keywords"`
}

type SceneKeyword struct {
	Keyword string `json:"keyword"`
	Desc    string `json:"desc"`
	Mode    string `json:"mode"` // word | regex
	Color   string `json:"color"`
}

type JiraConfig struct {
	BaseURL  string `json:"base_url"`
	Email    string `json:"email"`
	APIToken string `json:"api_token"`
}

type JiraAttachment struct {
	ID         string `json:"id"`
	Filename   string `json:"filename"`
	Size       int64  `json:"size"`
	MimeType   string `json:"mime_type"`
	ContentURL string `json:"content_url"`
}
