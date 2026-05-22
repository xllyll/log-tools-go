package handler

import (
	"net/http"
	"os"
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

	paths, err := h.storage.ExtractArchive(saved)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.UploadResponse{Success: false, Error: err.Error()})
		return
	}
	if len(paths) == 0 {
		paths = []string{saved}
	}

	var fileIDs []string
	for _, p := range paths {
		ext := strings.ToLower(filepath.Ext(p))
		if ext != ".log" && ext != ".txt" && ext != ".gz" {
			continue
		}
		lf, err := h.storage.StartIngest(deviceID, p)
		if err != nil {
			continue
		}
		fileIDs = append(fileIDs, lf.ID)
	}

	if len(fileIDs) == 0 {
		c.JSON(http.StatusBadRequest, model.UploadResponse{Success: false, Error: "no valid log files found"})
		return
	}

	c.JSON(http.StatusOK, model.UploadResponse{
		Success: true,
		Message: "upload accepted, parsing in background",
		FileIDs: fileIDs,
	})
}

func (h *UploadHandler) ListFiles(c *gin.Context) {
	deviceID := GetDeviceID(c)
	files, err := h.storage.GetFiles(c.Request.Context(), deviceID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.APIResponse{Success: false, Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, model.APIResponse{Success: true, Data: files})
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
	for _, id := range req.IDs {
		_ = h.storage.DeleteFile(c.Request.Context(), deviceID, id)
	}
	c.JSON(http.StatusOK, model.APIResponse{Success: true, Message: "batch deleted"})
}

// DeletePhysical removes uploaded raw file from disk (optional cleanup)
func DeletePhysical(path string) {
	_ = os.Remove(path)
}
