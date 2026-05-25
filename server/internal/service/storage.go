package service

import (
	"archive/zip"
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"log-tools/server/internal/config"
	"log-tools/server/internal/model"
	"log-tools/server/pkg/job"

	"github.com/google/uuid"
	"github.com/mholt/archiver/v4"
)

type StorageService struct {
	cfg    *config.Config
	db     *model.Database
	parser *Parser
	pool   *job.Pool
}

func NewStorageService(cfg *config.Config, db *model.Database, parser *Parser, pool *job.Pool) *StorageService {
	return &StorageService{cfg: cfg, db: db, parser: parser, pool: pool}
}

func (s *StorageService) ValidateFile(size int64, filename string) error {
	if size > s.cfg.Storage.MaxFileSize {
		return fmt.Errorf("file exceeds max size %d", s.cfg.Storage.MaxFileSize)
	}
	ext := strings.ToLower(filepath.Ext(filename))
	for _, allowed := range []string{".log", ".txt", ".zip", ".rar", ".7z", ".gz"} {
		if ext == allowed {
			return nil
		}
	}
	return fmt.Errorf("unsupported file type: %s", ext)
}

func (s *StorageService) SaveUpload(src io.Reader, filename string) (string, error) {
	if err := os.MkdirAll(s.cfg.Storage.UploadDir, 0o755); err != nil {
		return "", err
	}
	ts := time.Now().Format("20060102_150405")
	ext := filepath.Ext(filename)
	base := strings.TrimSuffix(filename, ext)
	dst := filepath.Join(s.cfg.Storage.UploadDir, fmt.Sprintf("%s_%s%s", base, ts, ext))
	out, err := os.Create(dst)
	if err != nil {
		return "", err
	}
	defer out.Close()
	if _, err := io.Copy(out, src); err != nil {
		return "", err
	}
	return dst, nil
}

func (s *StorageService) ExtractArchive(path string) ([]string, error) {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".zip":
		return s.extractZip(path)
	case ".rar", ".7z":
		return s.extractArchiver(path)
	default:
		return []string{path}, nil
	}
}

func (s *StorageService) extractZip(zipPath string) ([]string, error) {
	reader, err := zip.OpenReader(zipPath)
	if err != nil {
		return nil, err
	}
	defer reader.Close()
	dir := filepath.Join(s.cfg.Storage.UploadDir, "extracted_"+time.Now().Format("20060102_150405"))
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	var files []string
	for _, f := range reader.File {
		if f.FileInfo().IsDir() {
			continue
		}
		e := strings.ToLower(filepath.Ext(f.Name))
		if e != ".log" && e != ".txt" && e != ".gz" {
			continue
		}
		target := filepath.Join(dir, filepath.Base(f.Name))
		if err := extractZipEntry(f, target); err == nil {
			files = append(files, target)
		}
	}
	return files, nil
}

func extractZipEntry(f *zip.File, target string) error {
	rc, err := f.Open()
	if err != nil {
		return err
	}
	defer rc.Close()
	out, err := os.Create(target)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, rc)
	return err
}

func (s *StorageService) extractArchiver(path string) ([]string, error) {
	dir := filepath.Join(s.cfg.Storage.UploadDir, "extracted_"+time.Now().Format("20060102_150405"))
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	format, _, err := archiver.Identify(context.Background(), path, file)
	if err != nil {
		return nil, err
	}
	ex, ok := format.(archiver.Extractor)
	if !ok {
		return nil, fmt.Errorf("unsupported archive")
	}
	_, _ = file.Seek(0, 0)
	var files []string
	err = ex.Extract(context.Background(), file, func(ctx context.Context, fi archiver.FileInfo) error {
		if fi.IsDir() {
			return nil
		}
		e := strings.ToLower(filepath.Ext(fi.Name()))
		if e != ".log" && e != ".txt" && e != ".gz" {
			return nil
		}
		target := filepath.Join(dir, filepath.Base(fi.Name()))
		out, err := os.Create(target)
		if err != nil {
			return err
		}
		defer out.Close()
		r, err := fi.Open()
		if err != nil {
			return err
		}
		defer r.Close()
		_, err = io.Copy(out, r)
		if err == nil {
			files = append(files, target)
		}
		return err
	})
	return files, err
}

func (s *StorageService) ingestFile(ctx context.Context, deviceID, fileID, path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	fileSize := info.Size()
	name := filepath.Base(path)

	s.report(ctx, fileID, name, "parsing", "开始解析文件", 0, 0, 5)

	parsed, err := s.parser.ParseFile(deviceID, fileID, path, func(lines int) {
		progress := parsingProgress(lines, fileSize)
		msg := fmt.Sprintf("解析中，已读取 %d 行", lines)
		s.report(ctx, fileID, name, "parsing", msg, lines, 0, progress)
	})
	if err != nil {
		s.report(ctx, fileID, name, "failed", err.Error(), 0, 0, 0)
		return err
	}

	total := len(parsed.Entries)
	s.report(ctx, fileID, name, "inserting", fmt.Sprintf("解析完成，共 %d 行，开始入库", total), total, total, 40)

	if err := s.db.DeleteEntriesByFile(ctx, parsed.ID); err != nil {
		s.report(ctx, fileID, name, "failed", err.Error(), 0, 0, 0)
		return err
	}

	batch := s.cfg.Ingest.BatchSize
	if batch <= 0 {
		batch = 500
	}
	entries := parsed.Entries
	for i := 0; i < len(entries); i += batch {
		end := i + batch
		if end > len(entries) {
			end = len(entries)
		}
		if err := s.db.BatchInsertEntries(ctx, deviceID, parsed.ID, entries[i:end]); err != nil {
			s.report(ctx, fileID, name, "failed", err.Error(), end, total, 0)
			return err
		}
		progress := 40 + end*60/total
		if total == 0 {
			progress = 100
		}
		msg := fmt.Sprintf("入库中 %d / %d 行", end, total)
		s.report(ctx, fileID, name, "inserting", msg, end, total, progress)
	}

	s.report(ctx, fileID, name, "ready", fmt.Sprintf("完成，共 %d 行", total), total, total, 100)
	return nil
}

func (s *StorageService) report(ctx context.Context, fileID, name, status, msg string, parsedLines, total, progress int) {
	_ = s.db.UpdateFileProgress(ctx, fileID, status, msg, parsedLines, total, progress)
	log.Printf("[ingest] file=%s name=%s status=%s progress=%d%% %s", fileID, name, status, progress, msg)
}

func parsingProgress(lines int, fileSize int64) int {
	if fileSize <= 0 {
		if lines >= 100000 {
			return 35
		}
		return min(35, lines/3000)
	}
	estimated := int(fileSize / 80)
	if estimated < 1 {
		estimated = 1
	}
	p := lines * 35 / estimated
	if p > 35 {
		p = 35
	}
	if p < 5 && lines > 0 {
		p = 5
	}
	return p
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func (s *StorageService) StartIngest(deviceID, path string) (*model.LogFile, error) {
	ctx := context.Background()
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	fileID := uuid.NewString()
	f := &model.LogFile{
		ID:        fileID,
		DeviceID:  deviceID,
		Name:      filepath.Base(path),
		Size:      info.Size(),
		UploadAt:  time.Now(),
		Status:    "parsing",
		StatusMsg: "排队等待解析",
		Progress:  0,
	}
	if err := s.db.SaveLogFile(ctx, f); err != nil {
		return nil, err
	}
	log.Printf("[ingest] queued file=%s name=%s device=%s", fileID, f.Name, deviceID)
	s.pool.Submit(func(c context.Context) error {
		return s.ingestFile(c, deviceID, fileID, path)
	})
	return f, nil
}

func (s *StorageService) GetFiles(ctx context.Context, deviceID string) ([]model.LogFile, error) {
	return s.db.GetLogFiles(ctx, deviceID)
}

func (s *StorageService) DeleteFile(ctx context.Context, deviceID, fileID string) error {
	return s.db.DeleteLogFile(ctx, deviceID, fileID)
}

func (s *StorageService) GetEntries(ctx context.Context, filter model.LogFilter) ([]model.LogEntry, error) {
	return s.db.GetLogEntries(ctx, filter)
}

func (s *StorageService) GetContext(ctx context.Context, deviceID, fileID string, line, before, after int) ([]model.LogEntry, error) {
	return s.db.GetContextEntries(ctx, deviceID, fileID, line, before, after)
}

func (s *StorageService) GetFile(ctx context.Context, deviceID, fileID string) (*model.LogFile, error) {
	return s.db.GetLogFile(ctx, deviceID, fileID)
}
