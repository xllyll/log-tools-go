package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"log-tools/server/internal/model"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
)

type SceneHandlerDB struct {
	db *model.Database
}

func NewSceneHandlerDB(db *model.Database) *SceneHandlerDB {
	return &SceneHandlerDB{db: db}
}

func (h *SceneHandlerDB) Save(c *gin.Context) {
	deviceID := GetDeviceID(c)
	var req struct {
		Name   string            `json:"name"`
		Config model.SceneConfig `json:"config"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.APIResponse{Success: false, Error: err.Error()})
		return
	}
	if req.Name == "" {
		req.Name = "default"
	}
	raw, err := json.Marshal(req.Config)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.APIResponse{Success: false, Error: err.Error()})
		return
	}
	if err := h.db.SaveSceneConfig(c.Request.Context(), deviceID, req.Name, raw); err != nil {
		c.JSON(http.StatusInternalServerError, model.APIResponse{Success: false, Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, model.APIResponse{Success: true, Message: "saved"})
}

func (h *SceneHandlerDB) List(c *gin.Context) {
	deviceID := GetDeviceID(c)
	list, err := h.db.ListSceneConfigs(c.Request.Context(), deviceID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.APIResponse{Success: false, Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, model.APIResponse{Success: true, Data: list})
}

// SaveShared stores scene config globally (not per device).
func (h *SceneHandlerDB) SaveShared(c *gin.Context) {
	var req struct {
		Config model.SceneConfig `json:"config"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.APIResponse{Success: false, Error: err.Error()})
		return
	}
	raw, err := json.Marshal(req.Config)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.APIResponse{Success: false, Error: err.Error()})
		return
	}
	if err := h.db.SaveSharedSceneConfig(c.Request.Context(), raw); err != nil {
		c.JSON(http.StatusInternalServerError, model.APIResponse{Success: false, Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, model.APIResponse{Success: true, Message: "shared scene config saved"})
}

// GetShared returns the global scene config for all users.
func (h *SceneHandlerDB) GetShared(c *gin.Context) {
	raw, updated, err := h.db.GetSharedSceneConfig(c.Request.Context())
	if errors.Is(err, pgx.ErrNoRows) {
		c.JSON(http.StatusNotFound, model.APIResponse{Success: false, Error: "服务器暂无共享场景配置，请先上传"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.APIResponse{Success: false, Error: err.Error()})
		return
	}
	cfg, err := model.ParseSceneConfigJSON(raw)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.APIResponse{Success: false, Error: "invalid shared config"})
		return
	}
	c.JSON(http.StatusOK, model.APIResponse{
		Success: true,
		Data: map[string]any{
			"config":     cfg,
			"updated_at": updated,
		},
	})
}
