package service

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

	"log-tools/server/internal/model"
	"log-tools/server/pkg/xencoding"
	"log-tools/server/pkg/xlog"
)

func lineMatchesFilter(content string, filter model.LogFilter) bool {
	for _, kw := range filter.Keywords {
		if kw == "" {
			continue
		}
		if filter.UseRegex {
			re, err := regexp.Compile(kw)
			if err != nil || !re.MatchString(content) {
				return false
			}
		} else if !strings.Contains(strings.ToLower(content), strings.ToLower(kw)) {
			return false
		}
	}
	if len(filter.SceneKeywords) > 0 {
		matched := false
		for _, kw := range filter.SceneKeywords {
			if kw == "" {
				continue
			}
			if strings.Contains(strings.ToLower(content), strings.ToLower(kw)) {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}
	return true
}

// QueryFileEntries reads the log file line by line and returns filtered entries.
func (p *Parser) QueryFileEntries(fileID, filePath string, filter model.LogFilter) ([]model.LogEntry, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	reader, err := openReader(file, filePath)
	if err != nil {
		return nil, err
	}

	limit := filter.Limit
	if limit <= 0 {
		limit = 2000
	}
	offset := filter.Offset
	if offset < 0 {
		offset = 0
	}

	scanner := bufio.NewScanner(reader)
	buf := make([]byte, 0, 1024*1024)
	scanner.Buffer(buf, 10*1024*1024)

	now := time.Now()
	lineNo := 0
	skipped := 0
	var entries []model.LogEntry

	for scanner.Scan() {
		lineNo++
		line := xencoding.DecodeLogLine(scanner.Bytes())
		if strings.TrimSpace(line) == "" {
			continue
		}
		if !lineMatchesFilter(line, filter) {
			continue
		}
		if skipped < offset {
			skipped++
			continue
		}
		entries = append(entries, model.LogEntry{
			ID:      fmt.Sprintf("%s_%d", fileID, lineNo),
			FileID:  fileID,
			LogTime: now,
			Content: line,
			Line:    lineNo,
			Level:   xlog.ParseLevel(line),
			Message: line,
		})
		if len(entries) >= limit {
			break
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return entries, nil
}

// FileContextEntries reads lines around the given physical line number from the file.
func (p *Parser) FileContextEntries(fileID, filePath string, line, before, after int) ([]model.LogEntry, error) {
	if before <= 0 {
		before = 10
	}
	if after <= 0 {
		after = 10
	}
	start := line - before
	if start < 1 {
		start = 1
	}
	end := line + after

	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	reader, err := openReader(file, filePath)
	if err != nil {
		return nil, err
	}

	scanner := bufio.NewScanner(reader)
	buf := make([]byte, 0, 1024*1024)
	scanner.Buffer(buf, 10*1024*1024)

	now := time.Now()
	lineNo := 0
	var entries []model.LogEntry

	for scanner.Scan() {
		lineNo++
		if lineNo > end {
			break
		}
		if lineNo < start {
			continue
		}
		content := xencoding.DecodeLogLine(scanner.Bytes())
		entries = append(entries, model.LogEntry{
			ID:      fmt.Sprintf("%s_%d", fileID, lineNo),
			FileID:  fileID,
			LogTime: now,
			Content: content,
			Line:    lineNo,
			Level:   xlog.ParseLevel(content),
			Message: content,
		})
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return entries, nil
}
