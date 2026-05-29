package service

import (
	"os"
	"path/filepath"
	"strings"
)

func isExtractRootDir(name string) bool {
	return strings.HasPrefix(name, "extracted_")
}

// findExtractRoot 从文件路径向上查找 extracted_* 解压根目录
func findExtractRoot(absPath string) string {
	dir, err := filepath.Abs(filepath.Dir(absPath))
	if err != nil {
		dir = filepath.Dir(absPath)
	}
	for {
		if isExtractRootDir(filepath.Base(dir)) {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return ""
}

func absUploadDir(uploadDir string) string {
	if uploadDir == "" {
		return ""
	}
	abs, err := filepath.Abs(uploadDir)
	if err != nil {
		return filepath.Clean(uploadDir)
	}
	return abs
}

// pruneEmptyParents 自底向上删除空目录，直到 stopDir（含）或无法再删
func pruneEmptyParents(fromDir, stopDir string) {
	stopDir = absUploadDir(stopDir)
	dir, err := filepath.Abs(fromDir)
	if err != nil {
		dir = filepath.Clean(fromDir)
	}
	for {
		if stopDir != "" {
			rel, err := filepath.Rel(stopDir, dir)
			if err != nil || rel == "." {
				break
			}
		}
		entries, err := os.ReadDir(dir)
		if err != nil || len(entries) > 0 {
			break
		}
		if err := os.Remove(dir); err != nil {
			break
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
}

func removePhysicalFilePath(path, uploadDir string) {
	if path == "" {
		return
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		abs = path
	}
	if err := os.Remove(abs); err != nil && !os.IsNotExist(err) {
		return
	}
	pruneEmptyParents(filepath.Dir(abs), uploadDir)
	if root := findExtractRoot(abs); root != "" {
		if emptyDir(root) {
			_ = os.RemoveAll(root)
		}
	}
}

func emptyDir(dir string) bool {
	entries, err := os.ReadDir(dir)
	return err == nil && len(entries) == 0
}

// removePhysicalFolderDirs 删除与虚拟文件夹对应的磁盘目录（解压目录下的同名路径）
func removePhysicalFolderDirs(uploadDir string, folderSegs []string, sourcePaths []string) {
	if len(folderSegs) == 0 || len(sourcePaths) == 0 {
		return
	}
	rel := filepath.FromSlash(strings.Join(folderSegs, "/"))
	roots := make(map[string]struct{})
	for _, p := range sourcePaths {
		if root := findExtractRoot(p); root != "" {
			roots[root] = struct{}{}
		}
	}
	uploadAbs := absUploadDir(uploadDir)
	for root := range roots {
		target, err := filepath.Abs(filepath.Join(root, rel))
		if err != nil {
			target = filepath.Join(root, rel)
		}
		if st, err := os.Stat(target); err == nil && st.IsDir() {
			_ = os.RemoveAll(target)
			pruneEmptyParents(filepath.Dir(target), uploadAbs)
			if emptyDir(root) {
				_ = os.RemoveAll(root)
			}
			continue
		}
		// 兼容：文件夹名与路径不完全一致时，若该层下已无文件则尝试清理
		pruneEmptyParents(target, uploadAbs)
	}
}
