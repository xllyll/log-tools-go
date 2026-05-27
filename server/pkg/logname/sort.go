package logname

import (
	"path/filepath"
	"strconv"
	"strings"
)

// RotationIndex parses log rotation suffix: no numeric suffix = newest (index -1).
// e.g. logcat.log -> -1; logcat.log.001.7z -> 1; logcat.log.007 -> 7.
func RotationIndex(name string) int {
	base := filepath.Base(name)
	lower := strings.ToLower(base)
	for _, ext := range []string{".7z", ".zip", ".rar", ".gz"} {
		if strings.HasSuffix(lower, ext) {
			base = base[:len(base)-len(ext)]
			lower = strings.ToLower(base)
			break
		}
	}
	dot := strings.LastIndex(base, ".")
	if dot < 0 {
		return -1
	}
	numPart := base[dot+1:]
	if numPart == "" {
		return -1
	}
	for _, c := range numPart {
		if c < '0' || c > '9' {
			return -1
		}
	}
	n, err := strconv.Atoi(numPart)
	if err != nil {
		return -1
	}
	return n
}

// Less sorts newest first: no suffix, then ascending rotation number.
func Less(a, b string) bool {
	ia, ib := RotationIndex(a), RotationIndex(b)
	if ia != ib {
		return ia < ib
	}
	return strings.ToLower(a) < strings.ToLower(b)
}

// SortStrings sorts filenames in place (newest → oldest).
func SortStrings(names []string) {
	if len(names) < 2 {
		return
	}
	// simple sort to avoid importing sort package... actually use sort
	sortSlice(names)
}

func sortSlice(names []string) {
	for i := 0; i < len(names); i++ {
		for j := i + 1; j < len(names); j++ {
			if !Less(names[i], names[j]) {
				names[i], names[j] = names[j], names[i]
			}
		}
	}
}

// PathEntry is a file path with its display/original name for sorting.
type PathEntry struct {
	Path string
	Name string
}

// SortPathEntries sorts by Name using rotation order.
func SortPathEntries(entries []PathEntry) {
	for i := 0; i < len(entries); i++ {
		for j := i + 1; j < len(entries); j++ {
			if !Less(entries[i].Name, entries[j].Name) {
				entries[i], entries[j] = entries[j], entries[i]
			}
		}
	}
}
