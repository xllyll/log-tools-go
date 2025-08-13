package handler

import (
	"fmt"
	"log-tools-go/internal/config"
	"log-tools-go/internal/model"
	"log-tools-go/internal/service"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
)

type UploadHandler struct {
	config  *config.Config
	storage *service.StorageService
	parser  *service.LogParser
}

func NewUploadHandler(cfg *config.Config, storage *service.StorageService, parser *service.LogParser) *UploadHandler {
	return &UploadHandler{
		config:  cfg,
		storage: storage,
		parser:  parser,
	}
}

func (h *UploadHandler) UploadFile(c *gin.Context) {
	// 获取上传的文件
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, model.UploadResponse{
			Success: false,
			Error:   "获取上传文件失败: " + err.Error(),
		})
		return
	}
	defer file.Close()

	// 创建临时文件
	tempFile, err := os.CreateTemp("", "upload_*")
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.UploadResponse{
			Success: false,
			Error:   "创建临时文件失败: " + err.Error(),
		})
		return
	}
	defer os.Remove(tempFile.Name())
	defer tempFile.Close()

	// 将上传的文件内容复制到临时文件
	if _, err := tempFile.ReadFrom(file); err != nil {
		c.JSON(http.StatusInternalServerError, model.UploadResponse{
			Success: false,
			Error:   "保存临时文件失败: " + err.Error(),
		})
		return
	}

	// 验证文件
	if err := h.storage.ValidateFile(tempFile, header.Filename); err != nil {
		c.JSON(http.StatusBadRequest, model.UploadResponse{
			Success: false,
			Error:   "文件验证失败: " + err.Error(),
		})
		return
	}

	// 保存上传的文件
	savedPath, err := h.storage.SaveUploadedFile(tempFile, header.Filename)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.UploadResponse{
			Success: false,
			Error:   "保存文件失败: " + err.Error(),
		})
		return
	}

	// 处理不同类型的文件
	var processedFiles []string
	ext := strings.ToLower(filepath.Ext(header.Filename))

	if ext == ".zip" {
		// 解压zip文件
		extractedFiles, err := h.storage.ExtractZipFile(savedPath)
		if err != nil {
			c.JSON(http.StatusInternalServerError, model.UploadResponse{
				Success: false,
				Error:   "解压文件失败: " + err.Error(),
			})
			return
		}
		processedFiles = extractedFiles
	} else {
		// 直接处理单个文件
		processedFiles = []string{savedPath}
	}

	// 解析所有文件
	projectName := c.PostForm("project_name")
	rule := config.GetRuleByProjectName(projectName)
	if rule == nil {
		c.JSON(http.StatusBadRequest, model.UploadResponse{
			Success: false,
			Error:   "项目不存在",
		})
		return
	}
	parser := service.NewLogParserWithRule(h.config, rule)

	var allLogFiles []*model.LogFile
	for _, filePath := range processedFiles {
		logFile, err := parser.ParseLogFile(filePath)
		if err != nil {
			// 记录错误但继续处理其他文件
			fmt.Printf("解析文件 %s 失败: %v\n", filePath, err)
			continue
		}
		// 保存解析结果
		if err := h.storage.SaveParsedLogs(logFile); err != nil {
			fmt.Printf("保存解析结果失败: %v\n", err)
		}

		allLogFiles = append(allLogFiles, logFile)
	}

	if len(allLogFiles) == 0 {
		c.JSON(http.StatusBadRequest, model.UploadResponse{
			Success: false,
			Error:   "没有成功解析任何日志文件",
		})
		return
	}

	// 返回成功响应
	fileIDs := make([]string, len(allLogFiles))
	for i, logFile := range allLogFiles {
		fileIDs[i] = logFile.ID
	}

	c.JSON(http.StatusOK, model.UploadResponse{
		Success: true,
		Message: fmt.Sprintf("成功上传并解析了 %d 个文件", len(allLogFiles)),
		FileID:  strings.Join(fileIDs, ","),
	})
}

func (h *UploadHandler) GetUploadedFiles(c *gin.Context) {
	files, err := h.storage.GetUploadedFiles()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "获取上传文件列表失败: " + err.Error(),
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    files,
	})
}

func (h *UploadHandler) DeleteFile(c *gin.Context) {
	fileID := c.Param("id")
	if fileID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "文件ID不能为空",
		})
		return
	}

	if err := h.storage.DeleteFile(fileID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "删除文件失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "文件删除成功",
	})
}

func (h *UploadHandler) GetProjects(c *gin.Context) {
	var names []string
	for _, pr := range config.ProjectRules {
		names = append(names, pr.ProjectName)
	}
	c.JSON(200, gin.H{"success": true, "data": names})
}
