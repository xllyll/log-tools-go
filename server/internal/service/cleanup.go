package service

import (
	"context"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// RunRetentionCleanup 启动后台任务：定期清理超过 retention_days 的上传日志（库记录 + 磁盘文件）。
// retention_days 为 0 时不启动。
func (s *StorageService) RunRetentionCleanup(ctx context.Context) {
	days := s.cfg.Storage.RetentionDays
	if days <= 0 {
		log.Printf("[cleanup] retention disabled (retention_days=0)")
		return
	}
	interval := time.Duration(s.cfg.Storage.CleanupIntervalHours) * time.Hour
	if interval <= 0 {
		interval = 24 * time.Hour
	}

	run := func() {
		n, err := s.purgeExpired(ctx)
		if err != nil {
			log.Printf("[cleanup] purge failed: %v", err)
			return
		}
		if n > 0 {
			log.Printf("[cleanup] removed %d expired log file(s) (older than %d days)", n, days)
		}
	}

	run()
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			run()
		}
	}
}

func (s *StorageService) purgeExpired(ctx context.Context) (int, error) {
	days := s.cfg.Storage.RetentionDays
	if days <= 0 {
		return 0, nil
	}
	cutoff := time.Now().AddDate(0, 0, -days)

	files, err := s.db.ListLogFilesBefore(ctx, cutoff)
	if err != nil {
		return 0, err
	}

	removed := 0
	seenPaths := make(map[string]struct{})
	for _, f := range files {
		if f.SourcePath != "" {
			seenPaths[f.SourcePath] = struct{}{}
			removePhysicalFile(f.SourcePath)
		}
		if err := s.db.DeleteLogFileByID(ctx, f.ID); err != nil {
			log.Printf("[cleanup] delete file id=%s: %v", f.ID, err)
			continue
		}
		removed++
	}

	s.purgeStaleUploadDir(cutoff, seenPaths)
	return removed, nil
}

func removePhysicalFile(path string) {
	if path == "" {
		return
	}
	if err := os.Remove(path); err == nil {
		return
	}
	if strings.Contains(path, "extracted_") {
		dir := filepath.Dir(path)
		if strings.Contains(filepath.Base(dir), "extracted_") {
			_ = os.RemoveAll(dir)
		}
	}
}

func (s *StorageService) purgeStaleUploadDir(cutoff time.Time, activePaths map[string]struct{}) {
	dir := s.cfg.Storage.UploadDir
	if dir == "" {
		return
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	for _, ent := range entries {
		full := filepath.Join(dir, ent.Name())
		if _, ok := activePaths[full]; ok {
			continue
		}
		info, err := ent.Info()
		if err != nil || !info.ModTime().Before(cutoff) {
			continue
		}
		if ent.IsDir() && strings.HasPrefix(ent.Name(), "extracted_") {
			_ = os.RemoveAll(full)
			continue
		}
		if !ent.IsDir() {
			_ = os.Remove(full)
		}
	}
}
