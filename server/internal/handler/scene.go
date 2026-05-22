package handler

import (
	"encoding/json"
	"net/http"

	"log-tools/server/internal/model"

	"github.com/gin-gonic/gin"
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
