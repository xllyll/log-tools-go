package handler

import (
	"net/http"

	"log-tools/server/internal/model"

	"github.com/gin-gonic/gin"
)

type SceneLibraryHandler struct {
	db *model.Database
}

func NewSceneLibraryHandler(db *model.Database) *SceneLibraryHandler {
	return &SceneLibraryHandler{db: db}
}

func (h *SceneLibraryHandler) Publish(c *gin.Context) {
	deviceID := GetDeviceID(c)
	var req struct {
		Title       string            `json:"title"`
		Description string            `json:"description"`
		Config      model.SceneConfig `json:"config"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.APIResponse{Success: false, Error: err.Error()})
		return
	}
	if req.Title == "" {
		c.JSON(http.StatusBadRequest, model.APIResponse{Success: false, Error: "title required"})
		return
	}
	if len(req.Config.Modules) == 0 {
		c.JSON(http.StatusBadRequest, model.APIResponse{Success: false, Error: "config modules empty"})
		return
	}
	id, err := h.db.PublishSceneLibrary(c.Request.Context(), deviceID, req.Title, req.Description, req.Config)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.APIResponse{Success: false, Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, model.APIResponse{Success: true, Message: "published", Data: gin.H{"id": id}})
}

func (h *SceneLibraryHandler) List(c *gin.Context) {
	deviceID := GetDeviceID(c)
	list, err := h.db.ListSceneLibrary(c.Request.Context(), deviceID, 100)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.APIResponse{Success: false, Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, model.APIResponse{Success: true, Data: list})
}

func (h *SceneLibraryHandler) Get(c *gin.Context) {
	id := c.Param("id")
	item, err := h.db.GetSceneLibrary(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, model.APIResponse{Success: false, Error: "scene pack not found"})
		return
	}
	item.IsMine = item.DeviceID == GetDeviceID(c)
	c.JSON(http.StatusOK, model.APIResponse{Success: true, Data: item})
}

func (h *SceneLibraryHandler) Delete(c *gin.Context) {
	deviceID := GetDeviceID(c)
	id := c.Param("id")
	if err := h.db.DeleteSceneLibrary(c.Request.Context(), deviceID, id); err != nil {
		c.JSON(http.StatusForbidden, model.APIResponse{Success: false, Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, model.APIResponse{Success: true, Message: "deleted"})
}
