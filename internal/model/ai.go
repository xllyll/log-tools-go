package model

type AnalysisLogRequest struct {
	FileID string `json:"fileId"`
	Logs   string `json:"logs"`
}

type GenerateLogRoleRequest struct {
	Log string `json:"log"`
	Msg string `json:"msg"`
}
