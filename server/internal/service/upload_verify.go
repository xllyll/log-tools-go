package service

import (
	"fmt"
	"os"
	"path/filepath"
)

func absFilePath(path string) (string, error) {
	if path == "" {
		return "", fmt.Errorf("empty path")
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return filepath.Clean(path), nil
	}
	return abs, nil
}

// verifyReadableFile ensures the path exists, is a non-empty regular file.
func verifyReadableFile(path string) (int64, error) {
	abs, err := absFilePath(path)
	if err != nil {
		return 0, err
	}
	info, err := os.Stat(abs)
	if err != nil {
		return 0, fmt.Errorf("文件不存在: %s", abs)
	}
	if info.IsDir() {
		return 0, fmt.Errorf("路径是目录而非文件: %s", abs)
	}
	if info.Size() == 0 {
		return 0, fmt.Errorf("文件为空: %s", abs)
	}
	return info.Size(), nil
}

// VerifyDownloadSize checks Jira/API download completeness when expected size is known.
func VerifyDownloadSize(got int64, expected int64, label string) error {
	if got == 0 {
		return fmt.Errorf("附件 %s 下载为空", label)
	}
	if expected > 0 && got != expected {
		return fmt.Errorf("附件 %s 下载不完整（期望 %d 字节，实际 %d 字节）", label, expected, got)
	}
	return nil
}
