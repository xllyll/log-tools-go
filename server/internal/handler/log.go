package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"log-tools/server/internal/model"
	"log-tools/server/internal/service"

	"github.com/gin-gonic/gin"
)

type LogHandler struct {
	storage *service.StorageService
}

func NewLogHandler(storage *service.StorageService) *LogHandler {
	return &LogHandler{storage: storage}
}

type logQueryReq struct {
	FileID        string          `json:"file_id"`
	FileIDs       []string        `json:"file_ids"`
	Keywords      []string        `json:"keywords"`
	SceneKeywords json.RawMessage `json:"scene_keywords"`
	UseRegex              bool            `json:"use_regex"`
	KeywordCaseSensitive  bool            `json:"keyword_case_sensitive"`
	Limit         int             `json:"limit"`
	Offset        int             `json:"offset"`
}

func parseSceneKeywordFilters(raw json.RawMessage) []model.SceneKeywordFilter {
	if len(raw) == 0 {
		return nil
	}
	var legacy []string
	if json.Unmarshal(raw, &legacy) == nil && len(legacy) > 0 {
		out := make([]model.SceneKeywordFilter, 0, len(legacy))
		for _, s := range legacy {
			if s != "" {
				sk := model.SceneKeywordFilter{Keyword: s, Mode: model.SceneKwModeWord, CaseSensitive: model.SceneKwCaseIgnore}
				model.NormalizeSceneKeywordFilter(&sk)
				out = append(out, sk)
			}
		}
		return out
	}
	var arr []json.RawMessage
	if json.Unmarshal(raw, &arr) != nil {
		return nil
	}
	out := make([]model.SceneKeywordFilter, 0, len(arr))
	for _, item := range arr {
		sk := model.ParseSceneKeywordFilterJSON(item)
		if sk.Keyword == "" {
			continue
		}
		out = append(out, sk)
	}
	return out
}

func (h *LogHandler) Query(c *gin.Context) {
	deviceID := GetDeviceID(c)
	var req logQueryReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.APIResponse{Success: false, Error: err.Error()})
		return
	}
	if req.FileID == "" && len(req.FileIDs) == 0 {
		c.JSON(http.StatusBadRequest, model.APIResponse{Success: false, Error: "file_id required"})
		return
	}
	if req.Limit == 0 {
		req.Limit = 2000
	}
	// Limit < 0：不分页，返回全部匹配（用于导出）
	filter := model.LogFilter{
		DeviceID:      deviceID,
		FileID:        req.FileID,
		FileIDs:       req.FileIDs,
		Keywords:      req.Keywords,
		SceneKeywords: parseSceneKeywordFilters(req.SceneKeywords),
		UseRegex:             req.UseRegex,
		KeywordCaseSensitive: req.KeywordCaseSensitive,
		Limit:         req.Limit,
		Offset:        req.Offset,
	}
	entries, err := h.storage.GetEntries(c.Request.Context(), filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.APIResponse{Success: false, Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, model.APIResponse{
		Success: true,
		Data: map[string]any{
			"entries": entries,
			"stats":   model.LogStats{TotalEntries: len(entries)},
		},
	})
}

func (h *LogHandler) Context(c *gin.Context) {
	deviceID := GetDeviceID(c)
	fileID := c.Query("file_id")
	lineStr := c.Query("line")
	if fileID == "" || lineStr == "" {
		c.JSON(http.StatusBadRequest, model.APIResponse{Success: false, Error: "file_id and line required"})
		return
	}
	line, _ := strconv.Atoi(lineStr)
	before, _ := strconv.Atoi(c.DefaultQuery("before", "10"))
	after, _ := strconv.Atoi(c.DefaultQuery("after", "10"))
	entries, err := h.storage.GetContext(c.Request.Context(), deviceID, fileID, line, before, after)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.APIResponse{Success: false, Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, model.APIResponse{Success: true, Data: entries})
}
