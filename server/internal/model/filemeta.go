package model

import (
	"path/filepath"
	"regexp"
	"strings"
)

// FileFormatFromName returns lower-case extension including dot, e.g. ".log".
func FileFormatFromName(name string) string {
	ext := filepath.Ext(name)
	if ext == "" {
		return ""
	}
	return strings.ToLower(ext)
}

// OriginalBaseName returns the base file name without directory segments.
func OriginalBaseName(name string) string {
	name = strings.ReplaceAll(name, "\\", "/")
	return filepath.Base(name)
}

var storageTimestampSuffix = regexp.MustCompile(`_\d{8}_\d{6}$`)

// InferOriginalFromStorageName strips the upload timestamp suffix from stored names.
// e.g. app_20060102_150405.log -> app.log
func InferOriginalFromStorageName(storageName string) string {
	ext := filepath.Ext(storageName)
	base := strings.TrimSuffix(storageName, ext)
	if storageTimestampSuffix.MatchString(base) {
		return storageTimestampSuffix.ReplaceAllString(base, "") + ext
	}
	return storageName
}

// NormalizeArchiveEntryName returns the nested archive file name without dedup timestamp.
// e.g. cat01_20260527_124556.7z -> cat01.7z
func NormalizeArchiveEntryName(entryName string) string {
	base := OriginalBaseName(entryName)
	ext := filepath.Ext(base)
	stem := strings.TrimSuffix(base, ext)
	if storageTimestampSuffix.MatchString(stem) {
		stem = storageTimestampSuffix.ReplaceAllString(stem, "")
	}
	return stem + ext
}

// FlattenedLogOriginalName: 单日志压缩包 → 压缩包文件名 + 内部日志扩展名（不含包内日志主名）。
// cat01.7z + log20250809.log → cat01.7z.log
func FlattenedLogOriginalName(archiveEntryName, innerLogName string) string {
	archive := NormalizeArchiveEntryName(archiveEntryName)
	ext := strings.ToLower(filepath.Ext(innerLogName))
	lower := strings.ToLower(innerLogName)
	if ext != ".log" && strings.Contains(lower, ".log") {
		ext = ".log"
	}
	if ext == "" {
		ext = ".log"
	}
	return archive + ext
}
