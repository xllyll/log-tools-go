package handler

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"

	"log-tools/server/internal/model"
	"log-tools/server/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
)

type UploadHandler struct {
	storage *service.StorageService
}

func NewUploadHandler(storage *service.StorageService) *UploadHandler {
	return &UploadHandler{storage: storage}
}

func dedupeDeleteRoots(ids []string) []string {
	seen := make(map[string]struct{}, len(ids))
	out := make([]string, 0, len(ids))
	for _, id := range ids {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out
}

// filterBatchDeleteRoots keeps only top-level ids when both a folder and its descendants are selected.
func filterBatchDeleteRoots(ids []string, items []model.LogFile) []string {
	if len(ids) <= 1 || len(items) == 0 {
		return ids
	}
	parentOf := make(map[string]string, len(items))
	inBatch := make(map[string]struct{}, len(ids))
	for _, id := range ids {
		inBatch[id] = struct{}{}
	}
	for _, it := range items {
		if it.ParentID != "" {
			parentOf[it.ID] = it.ParentID
		}
	}
	out := make([]string, 0, len(ids))
	for _, id := range ids {
		if hasSelectedAncestor(id, parentOf, inBatch) {
			continue
		}
		out = append(out, id)
	}
	return out
}

func hasSelectedAncestor(id string, parentOf map[string]string, inBatch map[string]struct{}) bool {
	for p := parentOf[id]; p != ""; p = parentOf[p] {
		if _, ok := inBatch[p]; ok {
			return true
		}
	}
	return false
}

func isDeleteIgnorable(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return true
	}
	s := strings.ToLower(err.Error())
	return strings.Contains(s, "not found") || strings.Contains(s, "no rows")
}

func (h *UploadHandler) Upload(c *gin.Context) {
	deviceID := GetDeviceID(c)
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, model.UploadResponse{Success: false, Error: err.Error()})
		return
	}
	defer file.Close()

	if err := h.storage.ValidateFile(header.Size, header.Filename); err != nil {
		c.JSON(http.StatusBadRequest, model.UploadResponse{Success: false, Error: err.Error()})
		return
	}

	saved, err := h.storage.SaveUpload(file, header.Filename)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.UploadResponse{Success: false, Error: err.Error()})
		return
	}

	fileIDs, err := h.storage.ImportSavedFile(c.Request.Context(), deviceID, saved, header.Filename)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.UploadResponse{Success: false, Error: err.Error()})
		return
	}

	if len(fileIDs) == 0 {
		msg := "压缩包内未解析到日志文件，请确认包含 .log/.txt/.json 或有效的嵌套压缩包"
		log.Printf("[upload] failed device=%s file=%s", deviceID, header.Filename)
		c.JSON(http.StatusBadRequest, model.UploadResponse{Success: false, Error: msg})
		return
	}

	c.JSON(http.StatusOK, model.UploadResponse{
		Success: true,
		Message: "upload accepted",
		FileIDs: fileIDs,
	})
}

func (h *UploadHandler) ListFiles(c *gin.Context) {
	deviceID := GetDeviceID(c)
	data, err := h.storage.ListFiles(c.Request.Context(), deviceID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.APIResponse{Success: false, Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, model.APIResponse{Success: true, Data: data})
}

func (h *UploadHandler) GetFileStatus(c *gin.Context) {
	deviceID := GetDeviceID(c)
	fileID := c.Param("id")
	f, err := h.storage.GetFile(c.Request.Context(), deviceID, fileID)
	if err != nil {
		c.JSON(http.StatusNotFound, model.APIResponse{Success: false, Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, model.APIResponse{Success: true, Data: f})
}

func (h *UploadHandler) DeleteFile(c *gin.Context) {
	deviceID := GetDeviceID(c)
	fileID := c.Param("id")
	if err := h.storage.DeleteFile(c.Request.Context(), deviceID, fileID); err != nil {
		c.JSON(http.StatusInternalServerError, model.APIResponse{Success: false, Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, model.APIResponse{Success: true, Message: "deleted"})
}

func (h *UploadHandler) BatchDelete(c *gin.Context) {
	deviceID := GetDeviceID(c)
	var req struct {
		IDs []string `json:"ids"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.APIResponse{Success: false, Error: err.Error()})
		return
	}
	ids := dedupeDeleteRoots(req.IDs)
	if list, err := h.storage.ListFiles(c.Request.Context(), deviceID); err == nil && list != nil {
		ids = filterBatchDeleteRoots(ids, list.Items)
	}
	var failed int
	var lastErr string
	for _, id := range ids {
		if err := h.storage.DeleteFile(c.Request.Context(), deviceID, id); err != nil {
			if isDeleteIgnorable(err) {
				continue
			}
			failed++
			lastErr = err.Error()
			log.Printf("[delete] batch item %s: %v", id, err)
		}
	}
	if failed > 0 && failed == len(ids) {
		c.JSON(http.StatusInternalServerError, model.APIResponse{Success: false, Error: lastErr})
		return
	}
	msg := "batch deleted"
	if failed > 0 {
		msg = fmt.Sprintf("partial delete: %d failed", failed)
	}
	c.JSON(http.StatusOK, model.APIResponse{Success: true, Message: msg})
}

func (h *UploadHandler) IngestFile(c *gin.Context) {
	deviceID := GetDeviceID(c)
	fileID := c.Param("id")
	if err := h.storage.BeginIngest(c.Request.Context(), deviceID, fileID); err != nil {
		c.JSON(http.StatusBadRequest, model.APIResponse{Success: false, Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, model.APIResponse{Success: true, Message: "ingest started"})
}

func (h *UploadHandler) RetryIngest(c *gin.Context) {
	h.IngestFile(c)
}
