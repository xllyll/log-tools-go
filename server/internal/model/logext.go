package model

import (
	"regexp"
	"strings"
)

// LogExtensions 为基础日志后缀。
// 轮转日志如 logcat.log.01、report.txt.24 由 IsLogFileName / LogFormatFromName 识别。
var LogExtensions = []string{".log", ".txt", ".json"}

// 匹配 name.log.01、name.txt.24 等（数字序号任意位数）。
var logRotatedNamePattern = regexp.MustCompile(`(?i)\.(log|txt|json)\.(\d+)$`)

func IsLogExtension(ext string) bool {
	ext = strings.ToLower(ext)
	for _, e := range LogExtensions {
		if ext == e {
			return true
		}
	}
	return false
}

// IsLogRotatedName reports logcat.log.01、report.txt.24 等轮转日志文件名。
func IsLogRotatedName(name string) bool {
	base := strings.ToLower(OriginalBaseName(name))
	return logRotatedNamePattern.MatchString(base)
}

func IsLogFileName(name string) bool {
	if IsLogRotatedName(name) {
		return true
	}
	return IsLogExtension(FileFormatFromName(name))
}

// LogFormatFromName returns the logical log extension (.log / .txt / .json), including rotated names.
func LogFormatFromName(name string) string {
	base := strings.ToLower(OriginalBaseName(name))
	if m := logRotatedNamePattern.FindStringSubmatch(base); m != nil {
		return "." + strings.ToLower(m[1])
	}
	return FileFormatFromName(name)
}
