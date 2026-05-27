package handler

import (
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

	var fileIDs []string
	for _, att := range req.Attachments {
		data, err := h.client.DownloadAttachment(att.ContentURL, att.ID)
		if err != nil {
			continue
		}
		dir := h.cfg.Storage.UploadDir
		_ = os.MkdirAll(dir, 0o755)
		finalPath := filepath.Join(dir, "jira_"+att.Filename)
		if err := os.WriteFile(finalPath, data, 0o644); err != nil {
			continue
		}
		orig := model.OriginalBaseName(att.Filename)
		lf, err := h.storage.RegisterUpload(deviceID, service.UploadFileMeta{
			Path:         finalPath,
			OriginalName: orig,
			FileFormat:   model.FileFormatFromName(orig),
		})
		if err != nil {
			_ = os.Remove(finalPath)
			continue
		}
		fileIDs = append(fileIDs, lf.ID)
	}
	if len(fileIDs) == 0 {
		c.JSON(http.StatusBadGateway, model.UploadResponse{Success: false, Error: "未能导入任何附件"})
		return
	}
	c.JSON(http.StatusOK, model.UploadResponse{
		Success: true,
		Message: "jira 附件已上传",
		FileIDs: fileIDs,
	})
}
