package handler

import (
	"github.com/gin-gonic/gin"
	"log-tools-go/internal/config"
	"log-tools-go/internal/service"
)

type ProjectHandler struct {
	config  *config.Config
	storage *service.StorageService
	parser  *service.LogParser
	project *service.ProjectService
}

func NewProjectHandler(cfg *config.Config, storage *service.StorageService, parser *service.LogParser, projectService *service.ProjectService) *ProjectHandler {
	return &ProjectHandler{
		config:  cfg,
		storage: storage,
		parser:  parser,
		project: projectService,
	}
}

/**
 * 获取项目列表
 */
func (h *ProjectHandler) GetProjects(c *gin.Context) {
	projects, err := h.project.GetAllProjects()
	if err != nil {
		c.JSON(500, gin.H{"success": false, "message": "获取项目列表失败"})
		return
	}
	c.JSON(200, gin.H{"success": true, "data": projects})
}
