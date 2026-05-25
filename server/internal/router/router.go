package router

import (
	"context"

	"log-tools/server/internal/config"
	"log-tools/server/internal/handler"
	"log-tools/server/internal/model"
	"log-tools/server/internal/service"
	"log-tools/server/pkg/job"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func Setup(cfg *config.Config, db *model.Database) *gin.Engine {
	pool := job.NewPool(cfg.Ingest.WorkerCount)
	parser := service.NewParser()
	storage := service.NewStorageService(cfg, db, parser, pool)
	go storage.RunRetentionCleanup(context.Background())

	uploadH := handler.NewUploadHandler(storage)
	logH := handler.NewLogHandler(storage)
	sceneH := handler.NewSceneHandlerDB(db)
	sceneLibH := handler.NewSceneLibraryHandler(db)
	jiraH := handler.NewJiraHandler(cfg, storage)

	r := gin.Default()
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "X-Device-ID"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
	}))

	api := r.Group("/api")
	api.Use(handler.RequireDeviceID())
	{
		api.POST("/upload", uploadH.Upload)
		api.GET("/files", uploadH.ListFiles)
		api.GET("/files/:id", uploadH.GetFileStatus)
		api.DELETE("/files/:id", uploadH.DeleteFile)
		api.POST("/files/:id/retry", uploadH.RetryIngest)
		api.POST("/files/batch-delete", uploadH.BatchDelete)

		api.POST("/logs/query", logH.Query)
		api.GET("/logs/context", logH.Context)

		api.GET("/scenes", sceneH.List)
		api.POST("/scenes", sceneH.Save)

		api.GET("/scene-library", sceneLibH.List)
		api.POST("/scene-library", sceneLibH.Publish)
		api.GET("/scene-library/:id", sceneLibH.Get)
		api.DELETE("/scene-library/:id", sceneLibH.Delete)

		api.POST("/jira/import", jiraH.Import)
		api.GET("/jira/issues/:key/attachments", jiraH.ListAttachments)
	}

	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"ok": true})
	})
	return r
}
