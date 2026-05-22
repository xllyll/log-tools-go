package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"log-tools/server/internal/config"
	"log-tools/server/internal/model"
	"log-tools/server/internal/router"

	"github.com/gin-gonic/gin"
)

func main() {
	cfgPath := os.Getenv("CONFIG_PATH")
	if cfgPath == "" {
		cfgPath = filepath.Join("config", "config.yaml")
	}
	if err := config.Load(cfgPath); err != nil {
		log.Fatalf("load config: %v", err)
	}
	cfg := config.Get()

	db, err := model.NewDatabase(cfg)
	if err != nil {
		log.Fatalf("database: %v", err)
	}
	defer db.Close()

	gin.SetMode(gin.ReleaseMode)
	r := router.Setup(cfg, db)
	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	log.Printf("log-tools server listening on %s", addr)
	if err := r.Run(addr); err != nil {
		log.Fatalf("server: %v", err)
	}
}
