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
	ID           string   `json:"id"`
	DeviceID     string   `json:"device_id"`
	EntryType    string   `json:"entry_type"` // file | folder
	Name         string   `json:"name"`       // stored file name on disk (may include dedup timestamp)
	OriginalName string   `json:"original_name"`
	FileFormat   string   `json:"file_format"`
	ParentID     string   `json:"parent_id,omitempty"`
	FolderPath   []string `json:"folder_path,omitempty"`
	Size           int64      `json:"size"`
	UploadAt    time.Time  `json:"upload_at"`
	Total       int        `json:"total"`
	ParsedLines int        `json:"parsed_lines"`
	Progress    int        `json:"progress"`
	Status      string     `json:"status"` // uploaded | parsing | inserting | ready | failed
	StatusMsg   string     `json:"status_msg,omitempty"`
	SourcePath     string     `json:"source_path,omitempty"`
	ChildFileCount int        `json:"child_file_count,omitempty"` // 文件夹列表：直属文件数
	Entries        []LogEntry `json:"-"`
}

// SceneKeywordFilter 场景查询关键字（与场景配置一致）
type SceneKeywordFilter struct {
	Keyword       string `json:"keyword"`
	Mode          int    `json:"mode"`           // 0 关键字 1 正则
	CaseSensitive int    `json:"case_sensitive"` // 0 不区分 1 区分，默认 0
}

type LogFilter struct {
	DeviceID      string
	FileID        string
	FileIDs       []string
	Keywords              []string
	SceneKeywords         []SceneKeywordFilter
	UseRegex              bool
	KeywordCaseSensitive  bool
	Limit         int
	Offset        int
	LineNumber    int
	Before        int
	After         int
}

type LogStats struct {
	TotalEntries int `json:"total_entries"`
}

type FileListData struct {
	Items []LogFile `json:"items"`
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
	Keyword       string `json:"keyword"`
	Desc          string `json:"desc"`
	Mode          int    `json:"mode"`           // 0 关键字 1 正则
	CaseSensitive int    `json:"case_sensitive"` // 0 不区分 1 区分
	Color         string `json:"color"`
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
