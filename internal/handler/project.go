package handler

import (
	"github.com/gin-gonic/gin"
	"log-tools-go/internal/config"
	"log-tools-go/internal/service"
	"net/http"
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

// 保存项目配置
func (h *ProjectHandler) SaveProjects(c *gin.Context) {
	var projects []config.LogProjectRule
	if err := c.ShouldBindJSON(&projects); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "请求参数错误: " + err.Error()})
		return
	}

	if err := h.project.SaveProjects(projects); err != nil {
		c.JSON(500, gin.H{"success": false, "message": "保存项目配置失败: " + err.Error()})
		return
	}

	c.JSON(200, gin.H{"success": true, "message": "项目配置保存成功"})
}
