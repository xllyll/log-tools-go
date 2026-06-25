package service

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func sanitizeDeviceID(deviceID string) string {
	s := sanitizeDiskName(strings.TrimSpace(deviceID))
	if s == "" {
		return "_unknown"
	}
	return s
}

// DeviceUploadDir returns (and creates) the on-disk directory for one device under upload_dir.
func (s *StorageService) DeviceUploadDir(deviceID string) (string, error) {
	dir := filepath.Join(s.cfg.Storage.UploadDir, sanitizeDeviceID(deviceID))
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	return dir, nil
}

func findFilesByBaseName(root, name string) ([]string, error) {
	if name == "" {
		return nil, fmt.Errorf("empty file name")
	}
	var matches []string
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil || info == nil || info.IsDir() {
			return nil
		}
		if filepath.Base(path) == name {
			matches = append(matches, path)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return matches, nil
}

// findUniqueFileByBaseName returns the only file with the given base name under root.
// Ambiguous or missing matches return an error to avoid reading the wrong content.
func findUniqueFileByBaseName(root, name string) (string, error) {
	matches, err := findFilesByBaseName(root, name)
	if err != nil {
		return "", err
	}
	switch len(matches) {
	case 0:
		return "", fmt.Errorf("源文件不存在，请重新上传: %s", name)
	case 1:
		abs, err := absFilePath(matches[0])
		if err != nil {
			return matches[0], nil
		}
		return abs, nil
	default:
		return "", fmt.Errorf("存在多个同名文件 %s，请删除后重新导入", name)
	}
}
