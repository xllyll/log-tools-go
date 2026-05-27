package service

import (
	"os"
	"path/filepath"
	"strings"
)

// isArchiveNoisePath skips macOS / hidden metadata inside archives.
func isArchiveNoisePath(path string) bool {
	rel := strings.ToLower(filepath.ToSlash(path))
	if strings.Contains(rel, "__macosx/") {
		return true
	}
	base := filepath.Base(path)
	if strings.HasPrefix(base, "._") {
		return true
	}
	if strings.EqualFold(base, ".ds_store") || strings.EqualFold(base, "thumbs.db") {
		return true
	}
	return false
}

func scanExtractedTreeFiltered(root string) (logPaths, archivePaths []string, err error) {
	err = filepath.WalkDir(root, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			return nil
		}
		if isArchiveNoisePath(path) {
			return nil
		}
		ext := strings.ToLower(filepath.Ext(d.Name()))
		switch {
		case isLogExt(ext):
			logPaths = append(logPaths, path)
		case isArchiveExt(ext):
			archivePaths = append(archivePaths, path)
		}
		return nil
	})
	return logPaths, archivePaths, err
}
