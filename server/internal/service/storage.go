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
	allowed := append([]string{".zip", ".rar", ".7z"}, model.LogExtensions...)
	for _, a := range allowed {
		if ext == a {
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

// UploadFileMeta describes a registered upload beyond the on-disk storage name.
type UploadFileMeta struct {
	Path           string
	OriginalName   string
	FileFormat     string
	ParentID string
}

func archiveDirParts(archiveRel string) []string {
	rel := filepath.ToSlash(archiveRel)
	dir := filepath.Dir(rel)
	if dir == "." || dir == "" {
		return nil
	}
	return splitPathSegments(dir)
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

func (s *StorageService) ingestFile(ctx context.Context, deviceID, fileID, path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	fileSize := info.Size()
	name := filepath.Base(path)
	estimatedTotal := estimatedLineCount(fileSize)

	if err := s.db.DeleteEntriesByFile(ctx, fileID); err != nil {
		s.report(ctx, fileID, name, "failed", err.Error(), 0, 0, 0)
		return err
	}

	s.report(ctx, fileID, name, "parsing", "开始解析入库", 0, estimatedTotal, 5)

	batch := s.cfg.Ingest.BatchSize
	if batch <= 0 {
		batch = 500
	}

	var lastProgress int
	stats, err := s.parser.IngestStream(deviceID, fileID, path, batch, IngestStreamHooks{
		OnBatch: func(entries []model.LogEntry, st IngestStreamStats) error {
			if err := s.db.BatchInsertEntries(ctx, deviceID, fileID, entries); err != nil {
				return err
			}
			progress := ingestProgress(st.LinesRead, fileSize)
			if progress-lastProgress >= 5 {
				msg := fmt.Sprintf("入库中 %d 行", st.EntryCount)
				s.report(ctx, fileID, name, "inserting", msg, st.EntryCount, estimatedTotal, progress)
				lastProgress = progress
			}
			return nil
		},
		OnProgress: func(lines int) {
			progress := ingestProgress(lines, fileSize)
			if progress-lastProgress >= 5 {
				msg := fmt.Sprintf("解析入库中，已处理 %d 行", lines)
				s.report(ctx, fileID, name, "inserting", msg, 0, estimatedTotal, progress)
				lastProgress = progress
			}
		},
	})
	if err != nil {
		s.report(ctx, fileID, name, "failed", err.Error(), 0, 0, 0)
		return err
	}

	total := stats.EntryCount
	s.report(ctx, fileID, name, "ready", fmt.Sprintf("完成，共 %d 行", total), total, total, 100)
	return nil
}

func (s *StorageService) report(ctx context.Context, fileID, name, status, msg string, parsedLines, total, progress int) {
	_ = s.db.UpdateFileProgress(ctx, fileID, status, msg, parsedLines, total, progress)
	log.Printf("[ingest] file=%s name=%s status=%s progress=%d%% %s", fileID, name, status, progress, msg)
}

func estimatedLineCount(fileSize int64) int {
	n := int(fileSize / 80)
	if n < 1 {
		return 1
	}
	return n
}

func ingestProgress(linesRead int, fileSize int64) int {
	estimated := estimatedLineCount(fileSize)
	p := linesRead * 95 / estimated
	if p > 95 {
		p = 95
	}
	if p < 5 && linesRead > 0 {
		p = 5
	}
	return p
}

// RegisterUpload records an uploaded file without ingesting into the database.
func (s *StorageService) RegisterUpload(deviceID string, meta UploadFileMeta) (*model.LogFile, error) {
	ctx := context.Background()
	info, err := os.Stat(meta.Path)
	if err != nil {
		return nil, err
	}
	if meta.OriginalName == "" {
		meta.OriginalName = model.OriginalBaseName(meta.Path)
	}
	if meta.FileFormat == "" {
		meta.FileFormat = model.FileFormatFromName(meta.OriginalName)
	}
	fileID := uuid.NewString()
	f := &model.LogFile{
		ID:             fileID,
		DeviceID:       deviceID,
		Name:           filepath.Base(meta.Path),
		OriginalName:   meta.OriginalName,
		FileFormat:     meta.FileFormat,
		EntryType:      model.EntryTypeFile,
		ParentID:       meta.ParentID,
		Size:           info.Size(),
		UploadAt:       time.Now(),
		Status:         "uploaded",
		StatusMsg:      "已上传，可预览；点击入库写入数据库",
		Progress:       0,
		SourcePath:     meta.Path,
	}
	if err := s.db.SaveLogFile(ctx, f); err != nil {
		return nil, err
	}
	log.Printf("[upload] registered file=%s storage=%s original=%s format=%s folder=%s device=%s",
		fileID, f.Name, f.OriginalName, f.FileFormat, f.ParentID, deviceID)
	return f, nil
}

func (s *StorageService) usesDatabase(f *model.LogFile) bool {
	return f.Status == "ready"
}

// BeginIngest starts background ingest for an uploaded or failed file.
func (s *StorageService) BeginIngest(ctx context.Context, deviceID, fileID string) error {
	f, err := s.db.GetLogFile(ctx, deviceID, fileID)
	if err != nil {
		return fmt.Errorf("file not found: %w", err)
	}
	switch f.Status {
	case "parsing", "inserting":
		return fmt.Errorf("文件正在入库中")
	case "ready":
		return fmt.Errorf("文件已入库")
	case "uploaded", "failed":
	default:
		return fmt.Errorf("当前状态无法入库: %s", f.Status)
	}
	path, err := s.resolveSourcePath(f)
	if err != nil {
		return err
	}
	if f.Status == "failed" {
		if err := s.db.DeleteEntriesByFile(ctx, fileID); err != nil {
			return err
		}
	}
	_ = s.db.UpdateFileProgress(ctx, fileID, "parsing", "排队等待入库", 0, 0, 0)
	log.Printf("[ingest] queued file=%s name=%s device=%s", fileID, f.Name, deviceID)
	s.pool.Submit(func(c context.Context) error {
		return s.ingestFile(c, deviceID, fileID, path)
	})
	return nil
}

func (s *StorageService) EnsureFolders(ctx context.Context, deviceID string, dirParts []string) (string, error) {
	return s.db.EnsureFolderChain(ctx, deviceID, normalizeFolderChain(dirParts))
}

func (s *StorageService) EnsureFolderChains(ctx context.Context, deviceID string, chains [][]string) error {
	for _, chain := range chains {
		if _, err := s.EnsureFolders(ctx, deviceID, chain); err != nil {
			return err
		}
	}
	return nil
}

func (s *StorageService) GetFiles(ctx context.Context, deviceID string) ([]model.LogFile, error) {
	return s.db.GetLogFiles(ctx, deviceID)
}

func (s *StorageService) ListFiles(ctx context.Context, deviceID string) (*model.FileListData, error) {
	return s.db.ListDeviceFiles(ctx, deviceID)
}

func (s *StorageService) DeleteFile(ctx context.Context, deviceID, fileID string) error {
	item, err := s.db.GetLogItem(ctx, deviceID, fileID)
	if err != nil {
		return err
	}
	if err := s.db.DeleteLogFile(ctx, deviceID, fileID); err != nil {
		return err
	}
	if item.IsFile() {
		s.removePhysicalSources(item)
	}
	log.Printf("[delete] %s=%s name=%s device=%s", item.EntryType, fileID, item.Name, deviceID)
	return nil
}

func (s *StorageService) removePhysicalSources(f *model.LogFile) {
	seen := make(map[string]struct{})
	add := func(p string) {
		if p == "" {
			return
		}
		abs, err := filepath.Abs(p)
		if err != nil {
			abs = p
		}
		if _, ok := seen[abs]; ok {
			return
		}
		seen[abs] = struct{}{}
		removePhysicalFile(abs)
	}
	add(f.SourcePath)
	if p, err := s.findPathByName(f.Name); err == nil {
		add(p)
	}
}

func (s *StorageService) GetEntries(ctx context.Context, filter model.LogFilter) ([]model.LogEntry, error) {
	if filter.FileID == "" && len(filter.FileIDs) == 0 {
		return nil, fmt.Errorf("file_id required")
	}
	if len(filter.FileIDs) > 0 {
		var all []model.LogEntry
		for _, id := range filter.FileIDs {
			f, err := s.db.GetLogFile(ctx, filter.DeviceID, id)
			if err != nil {
				return nil, err
			}
			sub := filter
			sub.FileID = id
			sub.FileIDs = nil
			chunk, err := s.getEntriesForFile(ctx, f, sub)
			if err != nil {
				return nil, err
			}
			all = append(all, chunk...)
		}
		return all, nil
	}
	f, err := s.db.GetLogFile(ctx, filter.DeviceID, filter.FileID)
	if err != nil {
		return nil, err
	}
	return s.getEntriesForFile(ctx, f, filter)
}

func (s *StorageService) getEntriesForFile(ctx context.Context, f *model.LogFile, filter model.LogFilter) ([]model.LogEntry, error) {
	if s.usesDatabase(f) {
		return s.db.GetLogEntries(ctx, filter)
	}
	path, err := s.resolveSourcePath(f)
	if err != nil {
		return nil, err
	}
	return s.parser.QueryFileEntries(f.ID, path, filter)
}

func (s *StorageService) GetContext(ctx context.Context, deviceID, fileID string, line, before, after int) ([]model.LogEntry, error) {
	f, err := s.db.GetLogFile(ctx, deviceID, fileID)
	if err != nil {
		return nil, err
	}
	if s.usesDatabase(f) {
		return s.db.GetContextEntries(ctx, deviceID, fileID, line, before, after)
	}
	path, err := s.resolveSourcePath(f)
	if err != nil {
		return nil, err
	}
	return s.parser.FileContextEntries(fileID, path, line, before, after)
}

func (s *StorageService) GetFile(ctx context.Context, deviceID, fileID string) (*model.LogFile, error) {
	return s.db.GetLogFile(ctx, deviceID, fileID)
}

func (s *StorageService) RetryIngest(ctx context.Context, deviceID, fileID string) error {
	return s.BeginIngest(ctx, deviceID, fileID)
}

func (s *StorageService) resolveSourcePath(f *model.LogFile) (string, error) {
	if f.SourcePath != "" {
		if _, err := os.Stat(f.SourcePath); err == nil {
			return f.SourcePath, nil
		}
	}
	return s.findPathByName(f.Name)
}

func (s *StorageService) findPathByName(name string) (string, error) {
	if name == "" {
		return "", fmt.Errorf("empty file name")
	}
	var matches []string
	err := filepath.Walk(s.cfg.Storage.UploadDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info == nil || info.IsDir() {
			return nil
		}
		if filepath.Base(path) == name {
			matches = append(matches, path)
		}
		return nil
	})
	_ = err
	if len(matches) == 0 {
		return "", fmt.Errorf("源文件不存在，请重新上传: %s", name)
	}
	best := matches[0]
	var bestMod time.Time
	for _, p := range matches {
		if st, err := os.Stat(p); err == nil && st.ModTime().After(bestMod) {
			bestMod = st.ModTime()
			best = p
		}
	}
	return best, nil
}
