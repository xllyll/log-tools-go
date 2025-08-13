package main

import (
	"fmt"
	"log"
	"log-tools-go/internal/config"
	"log-tools-go/internal/model"
	"time"
)

func main() {
	fmt.Println("=== 数据库功能测试 ===")

	// 加载配置
	fmt.Println("1. 加载配置...")
	if err := config.LoadConfig(); err != nil {
		log.Fatalf("配置加载失败: %v", err)
	}
	fmt.Println("✓ 配置加载成功")

	cfg := config.GetConfig()

	// 初始化数据库
	fmt.Println("2. 初始化数据库...")
	database, err := model.NewDatabase(cfg)
	if err != nil {
		log.Fatalf("数据库初始化失败: %v", err)
	}
	defer database.Close()
	fmt.Println("✓ 数据库初始化成功")

	// 创建测试日志文件
	fmt.Println("3. 创建测试数据...")
	testLogFile := &model.LogFile{
		ID:       "test_001",
		Name:     "test.log",
		Size:     1024,
		UploadAt: time.Now(),
		Entries: []model.LogEntry{
			{
				ID:        "entry_001",
				Timestamp: time.Now(),
				Level:     "INFO",
				Message:   "测试日志消息1",
				Source:    "test.log",
				Line:      1,
				Color:     "#17a2b8",
			},
			{
				ID:        "entry_002",
				Timestamp: time.Now().Add(time.Minute),
				Level:     "ERROR",
				Message:   "测试错误消息",
				Source:    "test.log",
				Line:      2,
				Color:     "#dc3545",
			},
		},
		Total: 2,
	}

	// 保存到数据库
	err = database.SaveLogFile(testLogFile)
	if err != nil {
		log.Fatalf("保存日志文件失败: %v", err)
	}
	fmt.Println("✓ 测试数据保存成功")

	// 查询日志文件列表
	fmt.Println("4. 查询日志文件列表...")
	files, err := database.GetLogFiles()
	if err != nil {
		log.Fatalf("查询日志文件失败: %v", err)
	}
	fmt.Printf("✓ 找到 %d 个日志文件\n", len(files))
	for _, file := range files {
		fmt.Printf("  - %s (%d 条日志)\n", file.Name, file.Total)
	}

	// 查询日志条目
	fmt.Println("5. 查询日志条目...")
	entries, err := database.GetLogEntries("test_001", model.LogFilter{})
	if err != nil {
		log.Fatalf("查询日志条目失败: %v", err)
	}
	fmt.Printf("✓ 找到 %d 条日志条目\n", len(entries))
	for _, entry := range entries {
		fmt.Printf("  - [%s] %s\n", entry.Level, entry.Message)
	}

	// 获取统计信息
	fmt.Println("6. 获取统计信息...")
	stats, err := database.GetLogStats("test_001", model.LogFilter{})
	if err != nil {
		log.Fatalf("获取统计信息失败: %v", err)
	}
	fmt.Printf("✓ 总条目数: %d\n", stats.TotalEntries)
	for level, count := range stats.LevelCounts {
		fmt.Printf("  - %s: %d\n", level, count)
	}

	fmt.Println("=== 数据库测试完成 ===")
}
