package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

const DeviceIDHeader = "X-Device-ID"

func RequireDeviceID() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.GetHeader(DeviceIDHeader)
		if id == "" {
			id = c.Query("device_id")
		}
		if id == "" {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
				"success": false,
				"error":   "missing device id (header X-Device-ID)",
			})
			return
		}
		c.Set("device_id", id)
		c.Next()
	}
}

func GetDeviceID(c *gin.Context) string {
	return c.GetString("device_id")
}
