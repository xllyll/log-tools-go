package handler

import (
	"log-tools-go/internal/config"
	"log-tools-go/internal/model"
	"log-tools-go/internal/service"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

type LogHandler struct {
	config  *config.Config
	storage *service.StorageService
	parser  *service.LogParser
}

func NewLogHandler(cfg *config.Config, storage *service.StorageService, parser *service.LogParser) *LogHandler {
	return &LogHandler{
		config:  cfg,
		storage: storage,
		parser:  parser,
	}
}

// LogQueryRequest 定义日志查询请求的JSON结构
type LogQueryRequest struct {
	FileID    string   `json:"file_id"`  // 文件ID
	FileIDs   string   `json:"file_ids"` // 多个文件ID
	Levels    []string `json:"levels"`   // 日志级别
	Keywords  []string `json:"keywords"` // 关键词ss
	StartTime *string  `json:"start_time"`
	EndTime   *string  `json:"end_time"`
	Source    string   `json:"source"`
	Module    string   `json:"module"`
	UseRegex  *bool    `json:"useRegex"` // 是否使用正则匹配
	Limit     int      `json:"limit"`
	Offset    int      `json:"offset"`
}

func (h *LogHandler) GetLogs(c *gin.Context) {
	var req LogQueryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.LogResponse{
			Success: false,
			Error:   "请求参数解析失败: " + err.Error(),
		})
		return
	}

	// 支持单个文件ID或多个文件ID（用逗号分隔）
	queryFileID := req.FileID
	if req.FileIDs != "" {
		queryFileID = req.FileIDs
	}

	// 如果没有提供任何文件ID参数，则返回错误
	if queryFileID == "" {
		c.JSON(http.StatusBadRequest, model.LogResponse{
			Success: false,
			Error:   "文件ID不能为空",
		})
		return
	}

	// 构建过滤条件
	filter := h.buildFilterFromRequest(req)

	// 从数据库获取日志条目
	entries, err := h.storage.GetLogEntries(queryFileID, filter)
	if err != nil {
		c.JSON(http.StatusNotFound, model.LogResponse{
			Success: false,
			Error:   "获取日志失败: " + err.Error(),
		})
		return
	}

	// 获取统计信息
	stats, err := h.storage.GetLogStats(queryFileID, filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.LogResponse{
			Success: false,
			Error:   "获取统计信息失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, model.LogResponse{
		Success: true,
		Data:    entries,
		Stats:   stats,
	})
}

func (h *LogHandler) GetLogStats(c *gin.Context) {
	fileID := c.Query("file_id")
	if fileID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "文件ID不能为空",
		})
		return
	}

	// 构建过滤条件
	filter := h.buildFilter(c)

	// 从数据库获取统计信息
	stats, err := h.storage.GetLogStats(fileID, filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "获取统计信息失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    stats,
	})
}

func (h *LogHandler) GetLogLevels(c *gin.Context) {
	levels := make([]map[string]interface{}, 0, len(h.config.LogLevels))
	for level := range h.config.LogLevels {
		// 将日志级别转换为大写
		lev := strings.ToUpper(level)
		color := h.config.LogLevels[level]
		levels = append(levels, map[string]interface{}{
			"level": lev,
			"color": color,
		})
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    levels,
	})
}

func (h *LogHandler) buildFilter(c *gin.Context) model.LogFilter {
	filter := model.LogFilter{}

	// 解析日志级别
	if levels := c.Query("levels"); levels != "" {
		filter.Levels = strings.Split(levels, ",")
	}

	// 解析关键词
	if keywords := c.Query("keywords"); keywords != "" {
		filter.Keywords = strings.Split(keywords, ",")
	}

	// 解析时间范围
	if startTime := c.Query("start_time"); startTime != "" {
		if t, err := time.Parse("2006-01-02T15:04:05", startTime); err == nil {
			filter.StartTime = &t
		}
	}

	if endTime := c.Query("end_time"); endTime != "" {
		if t, err := time.Parse("2006-01-02T15:04:05", endTime); err == nil {
			filter.EndTime = &t
		}
	}

	// 解析来源
	if source := c.Query("source"); source != "" {
		filter.Source = source
	}

	// 解析模块
	if module := c.Query("module"); module != "" {
		filter.Module = module
	}

	// 解析分页参数
	if limitStr := c.Query("limit"); limitStr != "" {
		if limit, err := strconv.Atoi(limitStr); err == nil {
			filter.Limit = limit
		}
	}

	if offsetStr := c.Query("offset"); offsetStr != "" {
		if offset, err := strconv.Atoi(offsetStr); err == nil {
			filter.Offset = offset
		}
	}

	return filter
}

// buildFilterFromRequest 从JSON请求构建过滤条件
func (h *LogHandler) buildFilterFromRequest(req LogQueryRequest) model.LogFilter {
	filter := model.LogFilter{
		Levels:   req.Levels,
		Keywords: req.Keywords,
		Source:   req.Source,
		Module:   req.Module,
		UseRegex: false,
		Limit:    req.Limit,
		Offset:   req.Offset,
	}
	if req.UseRegex != nil {
		filter.UseRegex = *req.UseRegex
	}

	// 解析时间范围
	if req.StartTime != nil {
		if t, err := time.Parse("2006-01-02T15:04:05", *req.StartTime); err == nil {
			filter.StartTime = &t
		}
	}

	if req.EndTime != nil {
		if t, err := time.Parse("2006-01-02T15:04:05", *req.EndTime); err == nil {
			filter.EndTime = &t
		}
	}

	return filter
}

func (h *LogHandler) SearchLogs(c *gin.Context) {
	fileID := c.Query("file_id")
	query := c.Query("q")

	if fileID == "" || query == "" {
		c.JSON(http.StatusBadRequest, model.LogResponse{
			Success: false,
			Error:   "文件ID和搜索关键词不能为空",
		})
		return
	}

	// 从数据库搜索日志
	searchResults, err := h.storage.SearchLogs(fileID, query, 100)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.LogResponse{
			Success: false,
			Error:   "搜索失败: " + err.Error(),
		})
		return
	}

	// 获取统计信息
	stats := h.parser.GetLogStats(searchResults)

	c.JSON(http.StatusOK, model.LogResponse{
		Success: true,
		Data:    searchResults,
		Stats:   stats,
	})
}

func (h *LogHandler) GetModuleOptions(ctx *gin.Context) {
	fileID := ctx.Query("file_id")
	// 从数据库查询 （根据file_id）
	moduleOptions, err := h.storage.GetModuleOptions(fileID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, model.LogResponse{
			Success: false,
			Error:   "获取模块选项失败: " + err.Error(),
		})
		return
	}
	ctx.JSON(http.StatusOK, model.R{
		Success: true,
		Data:    moduleOptions,
	})
}
