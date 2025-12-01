package model

type AnalysisLogRequest struct {
	FileID string `json:"fileId"`
	Logs   string `json:"logs"`
}

type GenerateLogRoleRequest struct {
	Log string `json:"log"`
	Msg string `json:"msg"`
}
type AiChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}
type AiChatCompletions struct {
	Model    string           `json:"model"`    // 模型名称
	Messages []*AiChatMessage `json:"messages"` // 消息列表s
}
