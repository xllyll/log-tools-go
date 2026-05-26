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

type IngestStreamStats struct {
	LinesRead  int
	EntryCount int
	FileSize   int64
}

type IngestStreamHooks struct {
	OnBatch    func(entries []model.LogEntry, stats IngestStreamStats) error
	OnProgress func(linesRead int)
}

// IngestStream reads the file line-by-line, batches non-empty entries, and flushes via OnBatch.
func (p *Parser) IngestStream(deviceID, fileID, filePath string, batchSize int, hooks IngestStreamHooks) (IngestStreamStats, error) {
	if batchSize <= 0 {
		batchSize = 500
	}

	file, err := os.Open(filePath)
	if err != nil {
		return IngestStreamStats{}, err
	}
	defer file.Close()

	reader, err := openReader(file, filePath)
	if err != nil {
		return IngestStreamStats{}, err
	}

	info, err := file.Stat()
	if err != nil {
		return IngestStreamStats{}, err
	}

	stats := IngestStreamStats{FileSize: info.Size()}
	scanner := bufio.NewScanner(reader)
	buf := make([]byte, 0, 1024*1024)
	scanner.Buffer(buf, 10*1024*1024)

	now := time.Now()
	lineNo := 0
	batch := make([]model.LogEntry, 0, batchSize)

	flush := func() error {
		if len(batch) == 0 {
			return nil
		}
		if hooks.OnBatch != nil {
			if err := hooks.OnBatch(batch, stats); err != nil {
				return err
			}
		}
		batch = batch[:0]
		return nil
	}

	reportProgress := func() {
		if hooks.OnProgress != nil && lineNo > 0 && lineNo%10000 == 0 {
			stats.LinesRead = lineNo
			hooks.OnProgress(lineNo)
		}
	}

	for scanner.Scan() {
		lineNo++
		line := xencoding.DecodeLogLine(scanner.Bytes())
		if strings.TrimSpace(line) == "" {
			reportProgress()
			continue
		}
		stats.EntryCount++
		batch = append(batch, model.LogEntry{
			ID:      fmt.Sprintf("%s_%d", fileID, lineNo),
			LogTime: now,
			Content: line,
			Line:    lineNo,
			Level:   xlog.ParseLevel(line),
			Message: line,
		})
		stats.LinesRead = lineNo
		if len(batch) >= batchSize {
			if err := flush(); err != nil {
				return stats, err
			}
		}
		reportProgress()
	}
	if err := flush(); err != nil {
		return stats, err
	}

	stats.LinesRead = lineNo
	if hooks.OnProgress != nil && lineNo > 0 {
		hooks.OnProgress(lineNo)
	}
	if err := scanner.Err(); err != nil {
		return stats, err
	}
	return stats, nil
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

// ParseFile loads all entries into memory; prefer IngestStream for ingest.
func (p *Parser) ParseFile(deviceID, fileID, filePath string, onProgress func(lines int)) (*model.LogFile, error) {
	if fileID == "" {
		fileID = uuid.NewString()
	}
	var entries []model.LogEntry
	stats, err := p.IngestStream(deviceID, fileID, filePath, 10000, IngestStreamHooks{
		OnBatch: func(batch []model.LogEntry, _ IngestStreamStats) error {
			entries = append(entries, batch...)
			return nil
		},
		OnProgress: onProgress,
	})
	if err != nil {
		return nil, err
	}
	return &model.LogFile{
		ID:       fileID,
		DeviceID: deviceID,
		Name:     filepath.Base(filePath),
		Size:     stats.FileSize,
		UploadAt: time.Now(),
		Status:   "parsing",
		Total:    stats.EntryCount,
		Entries:  entries,
	}, nil
}
