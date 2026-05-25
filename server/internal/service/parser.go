package service

import (
	"bufio"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"log-tools/server/internal/model"
	"log-tools/server/pkg/xencoding"
	"log-tools/server/pkg/xlog"

	"github.com/google/uuid"
)

type Parser struct{}

func NewParser() *Parser {
	return &Parser{}
}

type ParseProgress func(lines int)

func (p *Parser) ParseFile(deviceID, fileID, filePath string, onProgress ParseProgress) (*model.LogFile, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	reader, err := openReader(file, filePath)
	if err != nil {
		return nil, err
	}

	info, err := file.Stat()
	if err != nil {
		return nil, err
	}

	if fileID == "" {
		fileID = uuid.NewString()
	}
	logFile := &model.LogFile{
		ID:       fileID,
		DeviceID: deviceID,
		Name:     filepath.Base(filePath),
		Size:     info.Size(),
		UploadAt: time.Now(),
		Status:   "parsing",
	}

	scanner := bufio.NewScanner(reader)
	buf := make([]byte, 0, 1024*1024)
	scanner.Buffer(buf, 10*1024*1024)

	lineNo := 0
	now := time.Now()
	for scanner.Scan() {
		lineNo++
		line := xencoding.DecodeLogLine(scanner.Bytes())
		if strings.TrimSpace(line) == "" {
			continue
		}
		logFile.Entries = append(logFile.Entries, model.LogEntry{
			ID:      fmt.Sprintf("%s_%d", fileID, lineNo),
			LogTime: now,
			Content: line,
			Line:    lineNo,
			Level:   xlog.ParseLevel(line),
			Message: line,
		})
		if onProgress != nil && lineNo%5000 == 0 {
			onProgress(lineNo)
		}
	}
	if onProgress != nil && lineNo > 0 {
		onProgress(lineNo)
	}
	logFile.Total = len(logFile.Entries)
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return logFile, nil
}

func openReader(file *os.File, path string) (io.Reader, error) {
	if strings.HasSuffix(strings.ToLower(path), ".gz") {
		if _, err := file.Seek(0, 0); err != nil {
			return nil, err
		}
		return gzip.NewReader(file)
	}
	if _, err := file.Seek(0, 0); err != nil {
		return nil, err
	}
	return file, nil
}
