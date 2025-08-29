package router

import (
	"fmt"
	"log"
	"log-tools-go/internal/config"
	"log-tools-go/internal/handler"
	"log-tools-go/internal/model"
	"log-tools-go/internal/service"
	"log-tools-go/pkg/xip"
	"os/exec"
	"runtime"

	"github.com/gin-gonic/gin"
)

func InitRouter(r *gin.Engine, cfg *config.Config) {
	// 初始化数据库
	fmt.Println("正在初始化数据库...")
	database, err := model.NewDatabase(cfg)
	if err != nil {
		log.Fatalf("初始化数据库失败: %v", err)
	}

	fmt.Println("数据库初始化完成")
	// 创建服务实例
	fmt.Println("正在初始化服务...")
	parser := service.NewLogParserWithRule(cfg, nil)
	storage := service.NewStorageService(cfg, parser, database)
	projectService := service.NewProjectService(cfg, parser, database)
	fmt.Println("服务初始化完成")

	// 创建处理器
	fmt.Println("正在创建HTTP处理器...")
	uploadHandler := handler.NewUploadHandler(cfg, storage, parser)
	logHandler := handler.NewLogHandler(cfg, storage, parser)
	aiHandler := handler.NewAiHandler(cfg, storage, parser)
	projectHandler := handler.NewProjectHandler(cfg, storage, parser, projectService)
	fmt.Println("HTTP处理器创建完成")

	// 静态文件服务
	r.Static("/static", "./web/static")
	r.StaticFile("/", "./web/templates/index.html")

	// API路由
	api := r.Group("/api")
	{
		// 项目相关
		api.GET("/projects", projectHandler.GetProjects)

		// 文件上传相关
		api.POST("/upload", uploadHandler.UploadFile)
		api.GET("/files", uploadHandler.GetUploadedFiles)
		api.DELETE("/files/:id", uploadHandler.DeleteFile)

		// 日志相关
		api.GET("/logs", logHandler.GetLogs)
		api.GET("/logs/stats", logHandler.GetLogStats)
		api.GET("/logs/levels", logHandler.GetLogLevels)
		api.GET("/logs/search", logHandler.SearchLogs)

		// Ai 日志分析
		api.POST("/logs/analysis", aiHandler.AnalysisLog)
		api.POST("/logs/analysis/stream", aiHandler.AnalysisLogStream)
		api.POST("/logs/rule/generate", aiHandler.GenerateLogRule)
	}
}

// 打开浏览器
func OpenBrowser(url string) {
	var err error

	switch runtime.GOOS {
	case "linux":
		err = exec.Command("xdg-open", url).Start()
	case "windows":
		err = exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	case "darwin": // macOS
		err = exec.Command("open", url).Start()
	default:
		log.Printf("不支持的操作系统: %s", runtime.GOOS)
		return
	}

	if err != nil {
		log.Printf("无法打开浏览器: %v", err)
	}
}

// 获取本机IP
func GetLocalIP() string {
	return xip.GetLocalIP()
}
