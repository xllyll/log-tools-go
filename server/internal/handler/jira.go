package handler

import (
	"net/http"

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
	var req jiraImportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.APIResponse{Success: false, Error: err.Error()})
		return
	}
	if len(req.Attachments) == 0 {
		c.JSON(http.StatusBadRequest, model.APIResponse{Success: false, Error: "请选择至少一个附件"})
		return
	}
	ctx := c.Request.Context()
	fileIDs, err := h.importAttachments(ctx, deviceID, req.Attachments, func(JiraProgressEvent) {})
	if err != nil {
		c.JSON(http.StatusBadGateway, model.UploadResponse{Success: false, Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, model.UploadResponse{
		Success: true,
		Message: "jira 附件已导入",
		FileIDs: fileIDs,
	})
}
