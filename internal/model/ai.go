package model

type AnalysisLogRequest struct {
	FileID string `json:"fileId"`
	Logs   string `json:"logs"`
}
