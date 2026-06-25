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
	"sync"
	"time"

	"log-tools/server/internal/config"
	"log-tools/server/internal/model"
	"log-tools/server/pkg/job"

	"github.com/google/uuid"
)

type StorageService struct {
	cfg       *config.Config
	db        *model.Database
	parser    *Parser
	pool      *job.Pool
	extractMu *sync.Map
}

func NewStorageService(cfg *config.Config, db *model.Database, parser *Parser, pool *job.Pool) *StorageService {
	s := &StorageService{cfg: cfg, db: db, parser: parser, pool: pool, extractMu: &sync.Map{}}
	return s
}

func (s *StorageService) ValidateFile(size int64, filename string) error {
	if size > s.cfg.Storage.MaxFileSize {
		return fmt.Errorf("file exceeds max size %d", s.cfg.Storage.MaxFileSize)
	}
	if model.IsLogFileName(filename) {
		return nil
	}
	ext := strings.ToLower(filepath.Ext(filename))
	for _, a := range []string{".zip", ".rar", ".7z"} {
		if ext == a {
			return nil
		}
	}
	return fmt.Errorf("unsupported file type: %s", ext)
}

func (s *StorageService) SaveUpload(src io.Reader, deviceID, filename string) (string, error) {
	deviceDir, err := s.DeviceUploadDir(deviceID)
	if err != nil {
		return "", err
	}
	ts := time.Now().Format("20060102_150405")
	ext := filepath.Ext(filename)
	base := strings.TrimSuffix(filename, ext)
	dst := filepath.Join(deviceDir, fmt.Sprintf("%s_%s%s", base, ts, ext))
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
	Path         string
	OriginalName string
	FileFormat   string
	ParentID     string
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
// 同目录下 original_name 相同则覆盖已有记录（含磁盘文件与已入库日志行）。
func (s *StorageService) RegisterUpload(ctx context.Context, deviceID string, meta UploadFileMeta) (*model.LogFile, error) {
	absPath, err := absFilePath(meta.Path)
	if err != nil {
		return nil, err
	}
	meta.Path = absPath
	size, err := verifyReadableFile(meta.Path)
	if err != nil {
		return nil, err
	}
	if meta.OriginalName == "" {
		meta.OriginalName = model.OriginalBaseName(meta.Path)
	}
	if meta.FileFormat == "" {
		meta.FileFormat = model.FileFormatFromName(meta.OriginalName)
	}

	existing, err := s.db.FindFileByOriginal(ctx, deviceID, meta.OriginalName, meta.ParentID)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		s.removePhysicalSources(existing)
		if err := s.db.DeleteEntriesByFile(ctx, existing.ID); err != nil {
			return nil, err
		}
		existing.Name = filepath.Base(meta.Path)
		existing.FileFormat = meta.FileFormat
		existing.Size = size
		existing.UploadAt = time.Now()
		existing.Status = "uploaded"
		existing.StatusMsg = "已上传，可预览；点击入库写入数据库"
		existing.Progress = 0
		existing.ParsedLines = 0
		existing.Total = 0
		existing.SourcePath = meta.Path
		if err := s.db.SaveLogFile(ctx, existing); err != nil {
			return nil, err
		}
		log.Printf("[upload] overwrite file=%s storage=%s original=%s folder=%s device=%s",
			existing.ID, existing.Name, existing.OriginalName, existing.ParentID, deviceID)
		return existing, nil
	}

	fileID := uuid.NewString()
	f := &model.LogFile{
		ID:           fileID,
		DeviceID:     deviceID,
		Name:         filepath.Base(meta.Path),
		OriginalName: meta.OriginalName,
		FileFormat:   meta.FileFormat,
		EntryType:    model.EntryTypeFile,
		ParentID:     meta.ParentID,
		Size:         size,
		UploadAt:     time.Now(),
		Status:       "uploaded",
		StatusMsg:    "已上传，可预览；点击入库写入数据库",
		Progress:     0,
		SourcePath:   meta.Path,
	}
	if err := s.db.SaveLogFile(ctx, f); err != nil {
		return nil, err
	}
	log.Printf("[upload] registered file=%s storage=%s original=%s format=%s folder=%s device=%s",
		fileID, f.Name, f.OriginalName, f.FileFormat, f.ParentID, deviceID)
	return f, nil
}

// ImportSavedFile 解析已落盘文件（压缩包会解压）并登记日志；同 device 并发时会排队等待。
func (s *StorageService) ImportSavedFile(ctx context.Context, deviceID, diskPath, originalName string) ([]string, error) {
	var fileIDs []string
	err := s.RunDeviceExtractExclusive(deviceID, func() error {
		var innerErr error
		fileIDs, innerErr = s.importSavedFile(ctx, deviceID, diskPath, originalName)
		return innerErr
	})
	return fileIDs, err
}

// ImportSavedFileUnlocked 在已持有 device 解压锁时调用（见 RunDeviceExtractExclusive）。
func (s *StorageService) ImportSavedFileUnlocked(ctx context.Context, deviceID, diskPath, originalName string) ([]string, error) {
	return s.importSavedFile(ctx, deviceID, diskPath, originalName)
}

func (s *StorageService) importSavedFile(ctx context.Context, deviceID, diskPath, originalName string) ([]string, error) {
	log.Printf("[import] extract start device=%s file=%s", deviceID, originalName)
	result, err := s.ExtractArchive(deviceID, diskPath, originalName)
	if err != nil {
		return nil, err
	}
	extractRoot := result.ExtractRoot

	if err := s.EnsureFolderChains(ctx, deviceID, result.FolderChains); err != nil {
		s.cleanupExtractRoot(extractRoot)
		return nil, err
	}

	var fileIDs []string
	for _, ent := range result.Files {
		if !model.IsLogFileName(ent.OriginalName) {
			continue
		}
		if _, err := verifyReadableFile(ent.DiskPath); err != nil {
			s.rollbackImportedFiles(ctx, deviceID, fileIDs)
			s.cleanupExtractRoot(extractRoot)
			return nil, fmt.Errorf("解压文件 %s 无效: %w", ent.OriginalName, err)
		}
		ext := ent.FileFormat
		if ext == "" || !model.IsLogExtension(ext) {
			ext = model.LogFormatFromName(ent.OriginalName)
		}
		parentID, err := s.EnsureFolders(ctx, deviceID, ent.ArchiveDirParts)
		if err != nil {
			s.rollbackImportedFiles(ctx, deviceID, fileIDs)
			s.cleanupExtractRoot(extractRoot)
			return nil, fmt.Errorf("创建文件夹 %v: %w", ent.ArchiveDirParts, err)
		}
		lf, err := s.RegisterUpload(ctx, deviceID, UploadFileMeta{
			Path:         ent.DiskPath,
			OriginalName: ent.OriginalName,
			FileFormat:   ext,
			ParentID:     parentID,
		})
		if err != nil {
			s.rollbackImportedFiles(ctx, deviceID, fileIDs)
			s.cleanupExtractRoot(extractRoot)
			return nil, fmt.Errorf("登记 %s: %w", ent.OriginalName, err)
		}
		fileIDs = append(fileIDs, lf.ID)
	}
	if len(fileIDs) == 0 {
		s.cleanupExtractRoot(extractRoot)
		return nil, fmt.Errorf("压缩包内未解析到可导入的日志文件")
	}
	log.Printf("[import] extract done device=%s file=%s ids=%d", deviceID, originalName, len(fileIDs))
	return fileIDs, nil
}

func (s *StorageService) cleanupExtractRoot(extractRoot string) {
	if extractRoot == "" {
		return
	}
	if err := os.RemoveAll(extractRoot); err != nil && !os.IsNotExist(err) {
		log.Printf("[import] cleanup extract root %s: %v", extractRoot, err)
	}
}

func (s *StorageService) rollbackImportedFiles(ctx context.Context, deviceID string, fileIDs []string) {
	for _, id := range fileIDs {
		if err := s.DeleteFile(ctx, deviceID, id); err != nil {
			log.Printf("[import] rollback delete file=%s: %v", id, err)
		}
	}
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

func (s *StorageService) ListFolders(ctx context.Context, deviceID string) (*model.FileListData, error) {
	items, err := s.db.ListDeviceFolders(ctx, deviceID)
	if err != nil {
		return nil, err
	}
	return &model.FileListData{Items: items}, nil
}

func (s *StorageService) ListFilesByParent(ctx context.Context, deviceID, parentID string) (*model.FileListData, error) {
	items, err := s.db.ListDeviceFilesByParent(ctx, deviceID, parentID)
	if err != nil {
		return nil, err
	}
	return &model.FileListData{Items: items}, nil
}

func (s *StorageService) ListProcessingFiles(ctx context.Context, deviceID string) (*model.FileListData, error) {
	items, err := s.db.ListDeviceProcessingFiles(ctx, deviceID)
	if err != nil {
		return nil, err
	}
	if items == nil {
		items = []model.LogFile{}
	}
	return &model.FileListData{Items: items}, nil
}

func (s *StorageService) DeleteFile(ctx context.Context, deviceID, fileID string) error {
	item, err := s.db.GetLogItem(ctx, deviceID, fileID)
	if err != nil {
		return err
	}
	files, err := s.db.ListSubtreeFiles(ctx, deviceID, fileID)
	if err != nil {
		return err
	}
	deviceDir, err := s.DeviceUploadDir(deviceID)
	if err != nil {
		return err
	}
	pathSeen := make(map[string]struct{})
	var sourcePaths []string
	for i := range files {
		for _, p := range s.collectPhysicalPaths(&files[i]) {
			if _, ok := pathSeen[p]; ok {
				continue
			}
			pathSeen[p] = struct{}{}
			sourcePaths = append(sourcePaths, p)
		}
	}
	for _, p := range sourcePaths {
		removePhysicalFilePath(p, deviceDir)
	}
	if item.EntryType == model.EntryTypeFolder {
		folderSegs, _ := s.db.ResolveFolderPath(ctx, fileID)
		removePhysicalFolderDirs(deviceDir, folderSegs, sourcePaths)
	}
	if err := s.db.DeleteLogItemSubtree(ctx, deviceID, fileID); err != nil {
		return err
	}
	log.Printf("[delete] %s=%s name=%s device=%s descendants_files=%d",
		item.EntryType, fileID, item.Name, deviceID, len(files))
	return nil
}

func (s *StorageService) collectPhysicalPaths(f *model.LogFile) []string {
	seen := make(map[string]struct{})
	var out []string
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
		out = append(out, abs)
	}
	add(f.SourcePath)
	return out
}

func (s *StorageService) removePhysicalSources(f *model.LogFile) {
	deviceDir, err := s.DeviceUploadDir(f.DeviceID)
	if err != nil {
		deviceDir = s.cfg.Storage.UploadDir
	}
	for _, p := range s.collectPhysicalPaths(f) {
		removePhysicalFilePath(p, deviceDir)
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

// SourceFileForDownload resolves the on-disk source path and suggested download name.
func (s *StorageService) SourceFileForDownload(ctx context.Context, deviceID, fileID string) (path, downloadName string, err error) {
	f, err := s.db.GetLogFile(ctx, deviceID, fileID)
	if err != nil {
		return "", "", err
	}
	if f.EntryType != "file" {
		return "", "", fmt.Errorf("not a file")
	}
	path, err = s.resolveSourcePath(f)
	if err != nil {
		return "", "", err
	}
	downloadName = f.OriginalName
	if downloadName == "" {
		downloadName = f.Name
	}
	return path, downloadName, nil
}

func (s *StorageService) RetryIngest(ctx context.Context, deviceID, fileID string) error {
	return s.BeginIngest(ctx, deviceID, fileID)
}

func (s *StorageService) resolveSourcePath(f *model.LogFile) (string, error) {
	if f.SourcePath == "" {
		label := f.OriginalName
		if label == "" {
			label = f.Name
		}
		return "", fmt.Errorf("源文件路径缺失，请重新导入: %s", label)
	}
	abs, err := absFilePath(f.SourcePath)
	if err != nil {
		return "", err
	}
	info, err := os.Stat(abs)
	if err != nil {
		label := f.OriginalName
		if label == "" {
			label = f.Name
		}
		return "", fmt.Errorf("源文件不存在，请重新导入: %s", label)
	}
	if info.IsDir() {
		return "", fmt.Errorf("源文件路径无效（是目录）: %s", abs)
	}
	if f.Size > 0 && info.Size() != f.Size {
		label := f.OriginalName
		if label == "" {
			label = f.Name
		}
		return "", fmt.Errorf("源文件大小与记录不一致（磁盘 %d 字节，记录 %d 字节），请重新导入: %s",
			info.Size(), f.Size, label)
	}
	return abs, nil
}

func (s *StorageService) findPathByName(deviceID, name string) (string, error) {
	deviceDir, err := s.DeviceUploadDir(deviceID)
	if err != nil {
		return "", err
	}
	path, err := findUniqueFileByBaseName(deviceDir, name)
	if err == nil {
		return path, nil
	}
	if strings.Contains(err.Error(), "存在多个同名") {
		return "", err
	}
	return findUniqueFileByBaseName(s.cfg.Storage.UploadDir, name)
}
