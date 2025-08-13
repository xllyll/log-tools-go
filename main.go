package main

import (
	"fmt"
	"log"
	"log-tools-go/internal/config"
	"log-tools-go/router"

	"github.com/gin-gonic/gin"
)

func main() {
	fmt.Println("正在启动日志分析工具...")

	// 加载配置
	fmt.Println("正在加载配置文件...")
	if err := config.LoadConfig(); err != nil {
		log.Fatalf("加载配置失败: %v", err)
	}
	fmt.Println("配置文件加载成功")

	cfg := config.GetConfig()
	fmt.Printf("服务器配置: %s:%d\n", cfg.Server.Host, cfg.Server.Port)

	// 加载项目规则
	fmt.Println("正在加载项目规则...")
	if err := config.LoadProjectRules("config/config.json"); err != nil {
		log.Fatalf("加载项目规则失败: %v", err)
	}
	fmt.Println("项目规则加载成功")

	// 设置Gin模式
	gin.SetMode(gin.ReleaseMode)

	// 创建路由
	fmt.Println("正在设置路由...")
	r := gin.Default()
	router.InitRouter(r, cfg)

	// 启动服务器
	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	fmt.Printf("正在启动服务器: http://%s\n", addr)
	fmt.Println("日志分析工具已就绪，请访问 http://" + addr)
	fmt.Println("按 Ctrl+C 停止服务器")

	if err := r.Run(addr); err != nil {
		log.Fatalf("启动服务器失败: %v", err)
	}
}
