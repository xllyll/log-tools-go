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

var archiveExtensions = map[string]bool{".zip": true, ".rar": true, ".7z": true}

var rotationNumberSuffix = regexp.MustCompile(`^\d+$`)

func isArchiveExtension(ext string) bool {
	return archiveExtensions[strings.ToLower(ext)]
}

// flattenSuffixFromInner 使用包内日志的真实扩展名（.log / .txt / .json），无扩展名时默认 .log。
func flattenSuffixFromInner(innerLogName string) string {
	ext := strings.ToLower(filepath.Ext(innerLogName))
	if IsLogExtension(ext) {
		return ext
	}
	if ext == "" {
		return ".log"
	}
	return ext
}

// archiveStemLooksLikeLogName 判断去掉压缩包后缀后的名称是否已是日志类文件名（含轮转序号）。
// 例如 logcat.log.003、logcat.log、report.txt.002
func archiveStemLooksLikeLogName(stem string) bool {
	lower := strings.ToLower(stem)
	if IsLogExtension(filepath.Ext(lower)) {
		return true
	}
	lastDot := strings.LastIndex(lower, ".")
	if lastDot < 0 {
		return false
	}
	if !rotationNumberSuffix.MatchString(lower[lastDot+1:]) {
		return false
	}
	prefix := lower[:lastDot]
	for _, logExt := range LogExtensions {
		if strings.HasSuffix(prefix, logExt) {
			return true
		}
	}
	return false
}

// FlattenedLogOriginalName: 单日志压缩包展平后的展示名（不含包内日志主名）。
// cat01.7z + log20250809.log → cat01.7z.log
// logcat.log.003.zip + logcat.log → logcat.log.003.log
// logcat.log.003.zip + notes.txt → logcat.log.003.txt
func FlattenedLogOriginalName(archiveEntryName, innerLogName string) string {
	archive := NormalizeArchiveEntryName(archiveEntryName)
	suffix := flattenSuffixFromInner(innerLogName)
	arcExt := strings.ToLower(filepath.Ext(archive))
	if isArchiveExtension(arcExt) {
		stem := strings.TrimSuffix(archive, filepath.Ext(archive))
		if archiveStemLooksLikeLogName(stem) {
			if strings.HasSuffix(strings.ToLower(stem), suffix) {
				return stem
			}
			return stem + suffix
		}
	}
	return archive + suffix
}
