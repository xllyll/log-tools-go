package service

import (
	"bufio"
	"compress/gzip"
	"crypto/md5"
	"fmt"
	"io"
	"log-tools-go/internal/config"
	"log-tools-go/internal/model"
	"log-tools-go/pkg/xmatch"
	"log-tools-go/pkg/xtime"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type LogParser struct {
	config *config.Config
	rule   *config.LogParseRule
}

func NewLogParserWithRule(cfg *config.Config, rule *config.LogParseRule) *LogParser {
	return &LogParser{config: cfg, rule: rule}
}

func (p *LogParser) ParseLogFile(filePath string) (*model.LogFile, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("打开文件失败: %w", err)
	}
	defer file.Close()

	var reader io.Reader = file

	// 检查是否为gzip压缩文件
	if strings.HasSuffix(strings.ToLower(filePath), ".gz") {
		gzReader, err := gzip.NewReader(file)
		if err != nil {
			return nil, fmt.Errorf("解压gzip文件失败: %w", err)
		}
		defer gzReader.Close()
		reader = gzReader
	}

	// 生成文件ID
	fileID := p.generateFileID(filePath)

	// 获取文件信息
	fileInfo, err := file.Stat()
	if err != nil {
		return nil, fmt.Errorf("获取文件信息失败: %w", err)
	}

	logFile := &model.LogFile{
		ID:       fileID,
		Name:     filepath.Base(filePath),
		Size:     fileInfo.Size(),
		UploadAt: time.Now(),
		Entries:  []model.LogEntry{},
	}

	scanner := bufio.NewScanner(reader)
	lineNumber := 0

	for scanner.Scan() {
		lineNumber++
		line := strings.TrimSpace(scanner.Text())

		if line == "" {
			continue
		}

		entry := p.parseLogLine(line, lineNumber, filePath)
		if entry != nil {
			logFile.Entries = append(logFile.Entries, *entry)
		}
	}

	logFile.Total = len(logFile.Entries)

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("读取文件失败: %w", err)
	}

	return logFile, nil
}

func (p *LogParser) parseLogLine(line string, lineNumber int, source string) *model.LogEntry {
	rule := p.rule
	timestampStr := xmatch.Match(rule.Timestamp, line)
	timestamp, err := xtime.ParseTime(timestampStr, rule.TimestampFormat)
	if err != nil {
		timestamp = time.Now()
	}
	level := xmatch.Match(rule.Level, line)
	message := xmatch.Match(rule.Message, line)
	thread := xmatch.Match(rule.Thread, line)
	class := xmatch.Match(rule.Class, line)
	classLine := xmatch.Match(rule.ClassLine, line)
	fmt.Printf("匹配结果: %s, %s, %s\n", timestampStr, level, message)
	return &model.LogEntry{
		ID:        fmt.Sprintf("%s_%d", p.generateFileID(source), lineNumber),
		LogTime:   timestamp,
		SaveTime:  time.Now(),
		Level:     level,
		Thread:    &thread,
		Class:     &class,
		ClassLine: &classLine,
		Message:   message,
		Content:   line,
		Source:    source,
		Line:      lineNumber,
		Color:     "#6c757d",
	}
}

func (p *LogParser) generateFileID(filePath string) string {
	hash := md5.Sum([]byte(filePath + time.Now().String()))
	return fmt.Sprintf("%x", hash)[:8]
}

func (p *LogParser) FilterLogs(entries []model.LogEntry, filter model.LogFilter) []model.LogEntry {
	var filtered []model.LogEntry

	for _, entry := range entries {
		if p.matchesFilter(entry, filter) {
			filtered = append(filtered, entry)
		}
	}

	return filtered
}

func (p *LogParser) matchesFilter(entry model.LogEntry, filter model.LogFilter) bool {
	// 检查日志级别
	if len(filter.Levels) > 0 {
		levelMatch := false
		for _, level := range filter.Levels {
			if strings.EqualFold(entry.Level, level) {
				levelMatch = true
				break
			}
		}
		if !levelMatch {
			return false
		}
	}

	// 检查关键词
	if len(filter.Keywords) > 0 {
		keywordMatch := false
		for _, keyword := range filter.Keywords {
			if strings.Contains(strings.ToLower(entry.Message), strings.ToLower(keyword)) {
				keywordMatch = true
				break
			}
		}
		if !keywordMatch {
			return false
		}
	}

	// 检查时间范围
	if filter.StartTime != nil && entry.LogTime.Before(*filter.StartTime) {
		return false
	}
	if filter.EndTime != nil && entry.LogTime.After(*filter.EndTime) {
		return false
	}

	// 检查来源
	if filter.Source != "" && !strings.Contains(entry.Source, filter.Source) {
		return false
	}

	return true
}

func (p *LogParser) GetLogStats(entries []model.LogEntry) model.LogStats {
	stats := model.LogStats{
		LevelCounts: make(map[string]int),
	}

	if len(entries) == 0 {
		return stats
	}

	stats.TotalEntries = len(entries)
	stats.TimeRange.Start = entries[0].LogTime
	stats.TimeRange.End = entries[0].LogTime

	for _, entry := range entries {
		stats.LevelCounts[entry.Level]++

		if entry.LogTime.Before(stats.TimeRange.Start) {
			stats.TimeRange.Start = entry.LogTime
		}
		if entry.LogTime.After(stats.TimeRange.End) {
			stats.TimeRange.End = entry.LogTime
		}
	}

	return stats
}
