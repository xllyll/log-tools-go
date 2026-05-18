package main

import (
	"database/sql"
	"fmt"
	"log"
	_ "modernc.org/sqlite"
)

// 为现有数据库添加优化索引
func main() {
	dbPath := "./logs_v1.db"

	fmt.Println("正在打开数据库:", dbPath)
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		log.Fatalf("打开数据库失败: %v", err)
	}
	defer db.Close()

	// 测试连接
	if err := db.Ping(); err != nil {
		log.Fatalf("数据库连接失败: %v", err)
	}

	fmt.Println("数据库连接成功")

	// 创建优化索引
	indexes := []string{
		"CREATE INDEX IF NOT EXISTS idx_log_entries_module ON log_entries(module)",
		"CREATE INDEX IF NOT EXISTS idx_log_entries_file_module_time ON log_entries(file_id, module, log_time)",
		"CREATE INDEX IF NOT EXISTS idx_log_entries_file_time ON log_entries(file_id, log_time)",
		"CREATE INDEX IF NOT EXISTS idx_log_entries_file_level ON log_entries(file_id, level)",
	}

	for i, indexSQL := range indexes {
		fmt.Printf("正在创建索引 %d/4...\n", i+1)
		if _, err := db.Exec(indexSQL); err != nil {
			log.Printf("创建索引失败: %v\nSQL: %s", err, indexSQL)
		} else {
			fmt.Printf("✓ 索引 %d 创建成功\n", i+1)
		}
	}

	fmt.Println("\n索引创建完成！")
	fmt.Println("建议执行以下命令优化数据库性能:")
	fmt.Println("  VACUUM;  -- 重建数据库文件，优化存储")
	fmt.Println("  ANALYZE; -- 更新统计信息，优化查询计划")

	// 可选：执行VACUUM和ANALYZE
	fmt.Println("\n是否立即执行数据库优化？(y/n)")
	var response string
	fmt.Scanln(&response)

	if response == "y" || response == "Y" {
		fmt.Println("正在执行 VACUUM...")
		if _, err := db.Exec("VACUUM"); err != nil {
			log.Printf("VACUUM 失败: %v", err)
		} else {
			fmt.Println("✓ VACUUM 完成")
		}

		fmt.Println("正在执行 ANALYZE...")
		if _, err := db.Exec("ANALYZE"); err != nil {
			log.Printf("ANALYZE 失败: %v", err)
		} else {
			fmt.Println("✓ ANALYZE 完成")
		}

		fmt.Println("\n数据库优化完成！重启应用即可享受性能提升。")
	}
}
