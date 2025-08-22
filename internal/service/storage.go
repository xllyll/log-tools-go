package service

import (
	"archive/zip"
	"fmt"
	"io"
	"log-tools-go/internal/config"
	"log-tools-go/internal/model"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type StorageService struct {
	config   *config.Config
	parser   *LogParser
	database *model.Database
}

func NewStorageService(cfg *config.Config, parser *LogParser, database *model.Database) *StorageService {
	return &StorageService{
		config:   cfg,
		parser:   parser,
		database: database,
	}
}

func (s *StorageService) SaveUploadedFile(file *os.File, filename string) (string, error) {
	// 确保上传目录存在
	if err := os.MkdirAll(s.config.Storage.UploadDir, 0755); err != nil {
		return "", fmt.Errorf("创建上传目录失败: %w", err)
	}

	// 生成唯一文件名
	timestamp := time.Now().Format("20060102_150405")
	ext := filepath.Ext(filename)
	baseName := strings.TrimSuffix(filename, ext)
	newFilename := fmt.Sprintf("%s_%s%s", baseName, timestamp, ext)
	filePath := filepath.Join(s.config.Storage.UploadDir, newFilename)

	// 创建目标文件
	dst, err := os.Create(filePath)
	if err != nil {
		return "", fmt.Errorf("创建目标文件失败: %w", err)
	}
	defer dst.Close()

	// 重置文件指针到开始位置
	if _, err := file.Seek(0, 0); err != nil {
		return "", fmt.Errorf("重置文件指针失败: %w", err)
	}

	// 复制文件内容
	if _, err := io.Copy(dst, file); err != nil {
		return "", fmt.Errorf("复制文件失败: %w", err)
	}

	return filePath, nil
}

func (s *StorageService) ExtractZipFile(zipPath string) ([]string, error) {
	var extractedFiles []string

	reader, err := zip.OpenReader(zipPath)
	if err != nil {
		return nil, fmt.Errorf("打开zip文件失败: %w", err)
	}
	defer reader.Close()

	extractDir := filepath.Join(s.config.Storage.UploadDir, "extracted_"+time.Now().Format("20060102_150405"))
	if err := os.MkdirAll(extractDir, 0755); err != nil {
		return nil, fmt.Errorf("创建解压目录失败: %w", err)
	}

	for _, file := range reader.File {
		if file.FileInfo().IsDir() {
			continue
		}

		// 检查文件扩展名
		ext := strings.ToLower(filepath.Ext(file.Name))
		if ext != ".txt" && ext != ".log" && ext != ".gz" {
			continue
		}

		filePath := filepath.Join(extractDir, file.Name)

		// 创建目录结构
		if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
			continue
		}

		// 创建文件
		dst, err := os.Create(filePath)
		if err != nil {
			continue
		}

		// 打开zip中的文件
		src, err := file.Open()
		if err != nil {
			dst.Close()
			continue
		}

		// 复制内容
		_, err = io.Copy(dst, src)
		dst.Close()
		src.Close()

		if err == nil {
			extractedFiles = append(extractedFiles, filePath)
		}
	}

	return extractedFiles, nil
}

func (s *StorageService) SaveParsedLogs(logFile *model.LogFile) error {
	// 保存到SQLite数据库
	return s.database.SaveLogFile(logFile)
}

func (s *StorageService) LoadParsedLogs(fileID string) (*model.LogFile, error) {
	// 从数据库获取日志文件信息
	files, err := s.database.GetLogFiles()
	if err != nil {
		return nil, err
	}

	// 查找指定的文件
	for _, file := range files {
		if file.ID == fileID {
			// 获取日志条目
			entries, err := s.database.GetLogEntries(fileID, model.LogFilter{})
			if err != nil {
				return nil, err
			}
			logFile := &model.LogFile{
				ID:       file.ID,
				Name:     file.Name,
				Size:     file.Size,
				UploadAt: file.UploadAt,
				Entries:  entries,
				Total:    len(entries),
			}

			return logFile, nil
		}
	}

	return nil, fmt.Errorf("日志文件不存在: %s", fileID)
}

func (s *StorageService) GetUploadedFiles() ([]model.LogFile, error) {
	// 从数据库获取所有日志文件
	return s.database.GetLogFiles()
}

func (s *StorageService) DeleteFile(fileID string) error {
	// 从数据库删除日志文件
	return s.database.DeleteLogFile(fileID)
}

func (s *StorageService) ValidateFile(file *os.File, filename string) error {
	// 检查文件大小
	fileInfo, err := file.Stat()
	if err != nil {
		return fmt.Errorf("获取文件信息失败: %w", err)
	}

	if fileInfo.Size() > s.config.Storage.MaxFileSize {
		return fmt.Errorf("文件大小超过限制: %d bytes", s.config.Storage.MaxFileSize)
	}

	// 检查文件扩展名
	ext := strings.ToLower(filepath.Ext(filename))
	allowedExts := []string{".txt", ".log", ".gz", ".zip"}

	allowed := false
	for _, allowedExt := range allowedExts {
		if ext == allowedExt {
			allowed = true
			break
		}
	}

	if !allowed {
		return fmt.Errorf("不支持的文件类型: %s", ext)
	}

	return nil
}

// 新增：从数据库获取日志条目（带过滤）
func (s *StorageService) GetLogEntries(fileID string, filter model.LogFilter) ([]model.LogEntry, error) {
	return s.database.GetLogEntries(fileID, filter)
}

// 新增：从数据库获取统计信息
func (s *StorageService) GetLogStats(fileID string, filter model.LogFilter) (model.LogStats, error) {
	return s.database.GetLogStats(fileID, filter)
}

// 新增：从数据库搜索日志
func (s *StorageService) SearchLogs(fileID string, query string, limit int) ([]model.LogEntry, error) {
	return s.database.SearchLogs(fileID, query, limit)
}
