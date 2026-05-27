package handler

import (
	"log"
	"net/http"
	"os"
	"path/filepath"

	"log-tools/server/internal/config"
	"log-tools/server/internal/model"
	"log-tools/server/internal/service"
	"log-tools/server/pkg/jira"

	"github.com/gin-gonic/gin"
)

type JiraHandler struct {
	cfg     *config.Config
	storage *service.StorageService
	client  *jira.Client
}

func NewJiraHandler(cfg *config.Config, storage *service.StorageService) *JiraHandler {
	return &JiraHandler{
		cfg:     cfg,
		storage: storage,
		client:  jira.NewClient(cfg.Jira),
	}
}

func (h *JiraHandler) ListAttachments(c *gin.Context) {
	issueKey := c.Param("key")
	list, err := h.client.ListLogAttachments(issueKey)
	if err != nil {
		c.JSON(http.StatusBadGateway, model.APIResponse{Success: false, Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, model.APIResponse{Success: true, Data: list})
}

func (h *JiraHandler) Import(c *gin.Context) {
	deviceID := GetDeviceID(c)
	var req struct {
		IssueKey    string `json:"issue_key"`
		Attachments []struct {
			ID         string `json:"id"`
			Filename   string `json:"filename"`
			ContentURL string `json:"content_url"`
		} `json:"attachments"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.APIResponse{Success: false, Error: err.Error()})
		return
	}
	if len(req.Attachments) == 0 {
		c.JSON(http.StatusBadRequest, model.APIResponse{Success: false, Error: "请选择至少一个附件"})
		return
	}

	dir := h.cfg.Storage.UploadDir
	if err := os.MkdirAll(dir, 0o755); err != nil {
		c.JSON(http.StatusInternalServerError, model.UploadResponse{Success: false, Error: err.Error()})
		return
	}

	ctx := c.Request.Context()
	var fileIDs []string
	var lastErr string
	for _, att := range req.Attachments {
		data, err := h.client.DownloadAttachment(att.ContentURL, att.ID)
		if err != nil {
			lastErr = err.Error()
			log.Printf("[jira] download %s: %v", att.Filename, err)
			continue
		}
		if err := h.storage.ValidateFile(int64(len(data)), att.Filename); err != nil {
			lastErr = err.Error()
			log.Printf("[jira] skip %s: %v", att.Filename, err)
			continue
		}
		finalPath := filepath.Join(dir, "jira_"+filepath.Base(att.Filename))
		if err := os.WriteFile(finalPath, data, 0o644); err != nil {
			lastErr = err.Error()
			continue
		}
		ids, err := h.storage.ImportSavedFile(ctx, deviceID, finalPath, att.Filename)
		if err != nil {
			lastErr = err.Error()
			_ = os.Remove(finalPath)
			log.Printf("[jira] import %s: %v", att.Filename, err)
			continue
		}
		if len(ids) == 0 {
			lastErr = "压缩包内未解析到可导入的日志文件"
			log.Printf("[jira] no logs from %s", att.Filename)
			continue
		}
		fileIDs = append(fileIDs, ids...)
	}
	if len(fileIDs) == 0 {
		msg := "未能导入任何附件"
		if lastErr != "" {
			msg = msg + ": " + lastErr
		}
		c.JSON(http.StatusBadGateway, model.UploadResponse{Success: false, Error: msg})
		return
	}
	c.JSON(http.StatusOK, model.UploadResponse{
		Success: true,
		Message: "jira 附件已导入",
		FileIDs: fileIDs,
	})
}
