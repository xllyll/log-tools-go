package handler

import (
	"fmt"
	"log"
	"net/http"
	"path/filepath"
	"strings"

	"log-tools/server/internal/model"
	"log-tools/server/internal/service"

	"github.com/gin-gonic/gin"
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

	result, err := h.storage.ExtractArchive(saved, header.Filename)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.UploadResponse{Success: false, Error: err.Error()})
		return
	}

	if err := h.storage.EnsureFolderChains(c.Request.Context(), deviceID, result.FolderChains); err != nil {
		c.JSON(http.StatusInternalServerError, model.UploadResponse{Success: false, Error: err.Error()})
		return
	}

	var fileIDs []string
	var lastErr string
	for _, ent := range result.Files {
		ext := ent.FileFormat
		if ext == "" {
			ext = strings.ToLower(filepath.Ext(ent.OriginalName))
		}
		if !model.IsLogExtension(ext) {
			lastErr = "unsupported extension: " + ent.OriginalName
			continue
		}
		parentID, err := h.storage.EnsureFolders(c.Request.Context(), deviceID, ent.ArchiveDirParts)
		if err != nil {
			lastErr = err.Error()
			log.Printf("[upload] ensure folders %v: %v", ent.ArchiveDirParts, err)
			continue
		}
		lf, err := h.storage.RegisterUpload(deviceID, service.UploadFileMeta{
			Path:         ent.DiskPath,
			OriginalName: ent.OriginalName,
			FileFormat:   ext,
			ParentID:     parentID,
		})
		if err != nil {
			lastErr = err.Error()
			log.Printf("[upload] register %s: %v", ent.OriginalName, err)
			continue
		}
		fileIDs = append(fileIDs, lf.ID)
	}

	if len(fileIDs) == 0 {
		msg := "no valid log files found"
		if len(result.Files) == 0 {
			msg = "压缩包内未解析到日志文件，请确认包含 .log/.txt/.json 或有效的嵌套压缩包"
		} else if lastErr != "" {
			msg = "未能登记任何日志文件: " + lastErr
		}
		log.Printf("[upload] failed device=%s file=%s entries=%d last_err=%s", deviceID, header.Filename, len(result.Files), lastErr)
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
	var failed int
	var lastErr string
	for _, id := range ids {
		if err := h.storage.DeleteFile(c.Request.Context(), deviceID, id); err != nil {
			if strings.Contains(err.Error(), "not found") {
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
