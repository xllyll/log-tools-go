package model

import (
	"database/sql"
	"fmt"
	"log-tools-go/internal/config"
	_ "modernc.org/sqlite"
	"time"
)

type Database struct {
	db *sql.DB
}

func NewDatabase(cfg *config.Config) (*Database, error) {
	dbPath := cfg.Storage.DatabasePath
	if dbPath == "" {
		dbPath = "./logs.db"
	}
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("打开数据库失败: %w", err)
	}
	// 测试连接
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("数据库连接失败: %w", err)
	}
	database := &Database{db: db}
	// 初始化表结构
	if err := database.initTables(); err != nil {
		return nil, fmt.Errorf("初始化数据库表失败: %w", err)
	}
	return database, nil
}

/**
 * 关闭数据库连接
 */
func (d *Database) Close() error {
	return d.db.Close()
}

func (d *Database) initTables() error {
	// 创建日志文件表
	createLogFilesTable := `
	CREATE TABLE IF NOT EXISTS log_files (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		size INTEGER NOT NULL,
		upload_at DATETIME NOT NULL,
		total_entries INTEGER DEFAULT 0,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);
	`

	// 创建日志条目表
	createLogEntriesTable := `
	CREATE TABLE IF NOT EXISTS log_entries (
		id TEXT PRIMARY KEY,
		file_id TEXT NOT NULL,
		log_time DATETIME NOT NULL,
		save_time DATETIME NOT NULL,
		level TEXT NOT NULL,
		module TEXT,
		process TEXT,
		thread TEXT,
		class TEXT,
		class_line TEXT,
		tag TEXT,
		message TEXT NOT NULL,
		content TEXT NOT NULL,
		source TEXT NOT NULL,
		line_number INTEGER NOT NULL,
		color TEXT NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (file_id) REFERENCES log_files(id) ON DELETE CASCADE
	);
	`

	// 创建索引
	createIndexes := `
	CREATE INDEX IF NOT EXISTS idx_log_entries_file_id ON log_entries(file_id);
	CREATE INDEX IF NOT EXISTS idx_log_entries_logtime ON log_entries(log_time);
	CREATE INDEX IF NOT EXISTS idx_log_entries_level ON log_entries(level);
	CREATE INDEX IF NOT EXISTS idx_log_entries_source ON log_entries(source);
	`

	// 执行创建表语句
	if _, err := d.db.Exec(createLogFilesTable); err != nil {
		return fmt.Errorf("创建log_files表失败: %w", err)
	}

	if _, err := d.db.Exec(createLogEntriesTable); err != nil {
		return fmt.Errorf("创建log_entries表失败: %w", err)
	}

	if _, err := d.db.Exec(createIndexes); err != nil {
		return fmt.Errorf("创建索引失败: %w", err)
	}

	return nil
}

// 保存日志文件信息
func (d *Database) SaveLogFile(logFile *LogFile) error {
	tx, err := d.db.Begin()
	if err != nil {
		return fmt.Errorf("开始事务失败: %w", err)
	}
	defer tx.Rollback()

	// 插入或更新日志文件信息
	stmt := `
	INSERT OR REPLACE INTO log_files (id, name, size, upload_at, total_entries)
	VALUES (?, ?, ?, ?, ?)`

	_, err = tx.Exec(stmt, logFile.ID, logFile.Name, logFile.Size, logFile.UploadAt, logFile.Total)
	if err != nil {
		return fmt.Errorf("保存日志文件信息失败: %w", err)
	}

	// 删除旧的日志条目
	_, err = tx.Exec("DELETE FROM log_entries WHERE file_id = ?", logFile.ID)
	if err != nil {
		return fmt.Errorf("删除旧日志条目失败: %w", err)
	}

	// 批量插入日志条目
	if len(logFile.Entries) > 0 {
		entryStmt := `
			INSERT INTO log_entries (id, file_id, log_time, save_time, module,level,process, thread,class,class_line,tag,message, content, source, line_number, color)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ? ,?)`

		for _, entry := range logFile.Entries {
			_, err = tx.Exec(entryStmt,
				entry.ID, logFile.ID, entry.LogTime, entry.SaveTime, entry.Module, entry.Level, entry.Process, entry.Thread, entry.Class, entry.ClassLine, entry.Tag,
				entry.Message, entry.Content, entry.Source, entry.Line, entry.Color)
			if err != nil {
				return fmt.Errorf("插入日志条目失败: %w", err)
			}
		}
	}

	return tx.Commit()
}

// 获取日志文件列表
func (d *Database) GetLogFiles() ([]LogFile, error) {
	rows, err := d.db.Query(`
		SELECT id, name, size, upload_at, total_entries
		FROM log_files
		ORDER BY upload_at DESC`)
	if err != nil {
		return nil, fmt.Errorf("查询日志文件失败: %w", err)
	}
	defer rows.Close()
	var files []LogFile
	for rows.Next() {
		var file LogFile
		err := rows.Scan(&file.ID, &file.Name, &file.Size, &file.UploadAt, &file.Total)
		if err != nil {
			return nil, fmt.Errorf("扫描日志文件数据失败: %w", err)
		}
		files = append(files, file)
	}
	return files, nil
}

// 获取日志条目
func (d *Database) GetLogEntries(fileID string, filter LogFilter) ([]LogEntry, error) {
	query := `
		SELECT id, log_time, save_time, module ,level, process, thread, class, class_line, tag,message, content, source, line_number, color
		FROM log_entries
		WHERE file_id = ?`
	args := []interface{}{fileID}

	// 添加过滤条件
	if len(filter.Levels) > 0 {
		placeholders := ""
		for i, _ := range filter.Levels {
			if i > 0 {
				placeholders += ","
			}
			placeholders += "?"
			args = append(args, filter.Levels[i])
		}
		query += " AND level IN (" + placeholders + ")"
	}

	if len(filter.Keywords) > 0 {
		for _, keyword := range filter.Keywords {
			query += " AND message LIKE ?"
			args = append(args, "%"+keyword+"%")
		}
	}

	if filter.StartTime != nil {
		query += " AND log_time >= ?"
		args = append(args, filter.StartTime)
	}

	if filter.EndTime != nil {
		query += " AND log_time <= ?"
		args = append(args, filter.EndTime)
	}

	if filter.Source != "" {
		query += " AND source LIKE ?"
		args = append(args, "%"+filter.Source+"%")
	}

	if filter.Module != "" {
		query += " AND module = ?"
		args = append(args, filter.Module)
	}

	// 添加排序和分页
	query += " ORDER BY line_number ASC"

	if filter.Limit > 0 {
		query += " LIMIT ?"
		args = append(args, filter.Limit)
	}

	if filter.Offset > 0 {
		query += " OFFSET ?"
		args = append(args, filter.Offset)
	}

	fmt.Printf("查询日志文件: %s, \n参数: %+v", query, args)
	rows, err := d.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("查询日志条目失败: %w", err)
	}
	defer rows.Close()

	var entries []LogEntry
	for rows.Next() {
		var entry LogEntry
		err := rows.Scan(
			&entry.ID,
			&entry.LogTime,
			&entry.SaveTime,
			&entry.Module,
			&entry.Level,
			&entry.Process,
			&entry.Thread,
			&entry.Class,
			&entry.ClassLine,
			&entry.Tag,
			&entry.Message,
			&entry.Content,
			&entry.Source,
			&entry.Line,
			&entry.Color,
		)
		if err != nil {
			return nil, fmt.Errorf("扫描日志条目数据失败: %w", err)
		}
		entries = append(entries, entry)
	}

	return entries, nil
}

// 获取日志统计信息
func (d *Database) GetLogStats(fileID string, filter LogFilter) (LogStats, error) {
	stats := LogStats{
		LevelCounts: make(map[string]int),
	}

	// 获取总条目数
	query := "SELECT COUNT(*) FROM log_entries WHERE file_id = ?"
	args := []interface{}{fileID}

	// 添加过滤条件
	if len(filter.Levels) > 0 {
		placeholders := ""
		for i, _ := range filter.Levels {
			if i > 0 {
				placeholders += ","
			}
			placeholders += "?"
			args = append(args, filter.Levels[i])
		}
		query += " AND level IN (" + placeholders + ")"
	}

	if len(filter.Keywords) > 0 {
		for _, keyword := range filter.Keywords {
			query += " AND message LIKE ?"
			args = append(args, "%"+keyword+"%")
		}
	}

	if filter.StartTime != nil {
		query += " AND log_time >= ?"
		args = append(args, filter.StartTime)
	}

	if filter.EndTime != nil {
		query += " AND log_time <= ?"
		args = append(args, filter.EndTime)
	}

	if filter.Source != "" {
		query += " AND source LIKE ?"
		args = append(args, "%"+filter.Source+"%")
	}

	if filter.Module != "" {
		query += " AND module = ?"
		args = append(args, filter.Module)
	}

	// 获取总条目数
	err := d.db.QueryRow(query, args...).Scan(&stats.TotalEntries)
	if err != nil {
		return stats, fmt.Errorf("获取总条目数失败: %w", err)
	}

	// 获取时间范围 - 修复时间类型转换
	timeQuery := "SELECT MIN(log_time), MAX(log_time) FROM log_entries WHERE file_id = ?"
	var startTimeStr, endTimeStr string
	err = d.db.QueryRow(timeQuery, fileID).Scan(&startTimeStr, &endTimeStr)
	if err != nil && err != sql.ErrNoRows {
		return stats, fmt.Errorf("获取时间范围失败: %w", err)
	}

	// 解析时间字符串
	if startTimeStr != "" {
		if t, err := time.Parse("2006-01-02 15:04:05", startTimeStr); err == nil {
			stats.TimeRange.Start = t
		}
	}
	if endTimeStr != "" {
		if t, err := time.Parse("2006-01-02 15:04:05", endTimeStr); err == nil {
			stats.TimeRange.End = t
		}
	}

	// 获取各级别统计
	levelQuery := `
		SELECT level, COUNT(*) 
		FROM log_entries 
		WHERE file_id = ?
		GROUP BY level`

	levelRows, err := d.db.Query(levelQuery, fileID)
	if err != nil {
		return stats, fmt.Errorf("获取级别统计失败: %w", err)
	}
	defer levelRows.Close()

	for levelRows.Next() {
		var level string
		var count int
		err := levelRows.Scan(&level, &count)
		if err != nil {
			return stats, fmt.Errorf("扫描级别统计失败: %w", err)
		}
		stats.LevelCounts[level] = count
	}

	return stats, nil
}

// 删除日志文件
func (d *Database) DeleteLogFile(fileID string) error {
	_, err := d.db.Exec("DELETE FROM log_files WHERE id = ?", fileID)
	if err != nil {
		return fmt.Errorf("删除日志文件失败: %w", err)
	}
	_, err = d.db.Exec("DELETE FROM log_entries WHERE file_id = ?", fileID)
	if err != nil {
		return fmt.Errorf("删除日志条目失败: %w", err)
	}
	return nil
}

// 搜索日志
func (d *Database) SearchLogs(fileID string, query string, limit int) ([]LogEntry, error) {
	rows, err := d.db.Query(`
		SELECT id, log_time,save_time, level, message, source, line_number, color
		FROM log_entries
		WHERE file_id = ? AND message LIKE ?
		ORDER BY log_time DESC
		LIMIT ?`, fileID, "%"+query+"%", limit)
	if err != nil {
		return nil, fmt.Errorf("搜索日志失败: %w", err)
	}
	defer rows.Close()

	var entries []LogEntry
	for rows.Next() {
		var entry LogEntry
		err := rows.Scan(&entry.ID, &entry.LogTime, &entry.SaveTime, &entry.Level,
			&entry.Message, &entry.Source, &entry.Line, &entry.Color)
		if err != nil {
			return nil, fmt.Errorf("扫描搜索结果失败: %w", err)
		}
		entries = append(entries, entry)
	}

	return entries, nil
}
