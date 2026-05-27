package model

import "strings"

// Log file extensions ingested and displayed by the system.
var LogExtensions = []string{".log", ".txt", ".json"}

func IsLogExtension(ext string) bool {
	ext = strings.ToLower(ext)
	for _, e := range LogExtensions {
		if ext == e {
			return true
		}
	}
	return false
}

func IsLogFileName(name string) bool {
	return IsLogExtension(FileFormatFromName(name))
}
