package multivolume

import (
	"fmt"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

var (
	rePartExt   = regexp.MustCompile(`(?i)^(.+)\.part(\d+)\.(rar|7z|zip)$`)
	reSplitExt  = regexp.MustCompile(`(?i)^(.+)\.(7z|zip)\.(\d+)$`)
	reRarContin = regexp.MustCompile(`(?i)^(.+)\.r(\d{2})$`)
)

// PartInfo describes one file in a multi-volume archive set.
type PartInfo struct {
	Filename string
	PartNum  int
}

// Group is a single archive or a multi-volume set sharing the same base name.
type Group struct {
	Key        string
	ArchiveExt string
	Parts      []PartInfo
}

// IsMultiVolume is true when more than one part file belongs to the same archive.
func (g Group) IsMultiVolume() bool {
	return len(g.Parts) > 1
}

// DisplayName is the user-facing archive name (e.g. OS_xxx.rar).
func (g Group) DisplayName() string {
	if len(g.Parts) == 0 {
		return ""
	}
	if g.Key != "" && g.ArchiveExt != "" {
		return g.Key + "." + g.ArchiveExt
	}
	return g.Parts[0].Filename
}

// ParseFilename returns volume group key, archive extension, 1-based part index, and whether
// the name looks like a multi-volume part (not a standalone archive).
func ParseFilename(filename string) (key, archiveExt string, partNum int, isVolumePart bool) {
	base := filepath.Base(filename)

	if m := rePartExt.FindStringSubmatch(base); m != nil {
		n, _ := strconv.Atoi(m[2])
		if n < 1 {
			n = 1
		}
		return m[1], strings.ToLower(m[3]), n, true
	}
	if m := reSplitExt.FindStringSubmatch(base); m != nil {
		n, _ := strconv.Atoi(m[3])
		if n < 1 {
			n = 1
		}
		return m[1], strings.ToLower(m[2]), n, true
	}
	if m := reRarContin.FindStringSubmatch(base); m != nil {
		n, _ := strconv.Atoi(m[2])
		return m[1], "rar", n + 2, true
	}
	ext := strings.ToLower(filepath.Ext(base))
	if ext == ".rar" || ext == ".7z" || ext == ".zip" {
		stem := strings.TrimSuffix(base, ext)
		stem = strings.TrimSuffix(stem, filepath.Ext(stem))
		return stem, strings.TrimPrefix(ext, "."), 1, false
	}
	return "", "", 0, false
}

// GroupFilenames clusters file names into volume groups and standalone files.
func GroupFilenames(names []string) (groups []Group, standalone []string) {
	type bucket struct {
		key string
		ext string
		m   map[int]string
	}
	buckets := make(map[string]*bucket)
	var others []string

	for _, name := range names {
		key, ext, partNum, isPart := ParseFilename(name)
		if !isPart && ext != "" {
			others = append(others, name)
			continue
		}
		if isPart {
			bk := key + "\x00" + ext
			if buckets[bk] == nil {
				buckets[bk] = &bucket{key: key, ext: ext, m: make(map[int]string)}
			}
			buckets[bk].m[partNum] = name
			continue
		}
		others = append(others, name)
	}

	for _, b := range buckets {
		g := Group{Key: b.key, ArchiveExt: b.ext}
		for n, fn := range b.m {
			g.Parts = append(g.Parts, PartInfo{Filename: fn, PartNum: n})
		}
		sort.Slice(g.Parts, func(i, j int) bool { return g.Parts[i].PartNum < g.Parts[j].PartNum })
		if g.IsMultiVolume() {
			groups = append(groups, g)
		} else if len(g.Parts) == 1 {
			_, _, _, isPart := ParseFilename(g.Parts[0].Filename)
			if isPart {
				groups = append(groups, g)
			} else {
				others = append(others, g.Parts[0].Filename)
			}
		}
	}
	sort.Slice(groups, func(i, j int) bool { return groups[i].DisplayName() < groups[j].DisplayName() })
	sort.Strings(others)
	return groups, others
}

// FirstPartFilename returns the first volume file name in a group.
func FirstPartFilename(g Group) string {
	if len(g.Parts) == 0 {
		return ""
	}
	return g.Parts[0].Filename
}

// HasRarVolumeSiblings reports whether other RAR volume parts exist beside archivePath in the same directory.
func HasRarVolumeSiblings(archivePath string) bool {
	dir := filepath.Dir(archivePath)
	base := filepath.Base(archivePath)
	key, ext, _, isPart := ParseFilename(base)
	if ext != "rar" {
		return false
	}
	if !isPart {
		return false
	}
	entries, err := filepath.Glob(filepath.Join(dir, "*"))
	if err != nil {
		return false
	}
	count := 0
	for _, p := range entries {
		k, e, _, part := ParseFilename(filepath.Base(p))
		if part && e == "rar" && k == key {
			count++
		}
	}
	return count > 1
}

// FindFirstVolumePath picks the lowest-numbered RAR part in dir for the same volume set as anyPartPath.
func FindFirstVolumePath(dir, anyPartPath string) string {
	key, ext, _, isPart := ParseFilename(filepath.Base(anyPartPath))
	if ext != "rar" || !isPart {
		return anyPartPath
	}
	entries, err := filepath.Glob(filepath.Join(dir, "*"))
	if err != nil {
		return anyPartPath
	}
	bestNum := -1
	bestPath := anyPartPath
	for _, p := range entries {
		if filepath.Dir(p) != dir {
			continue
		}
		k, e, n, part := ParseFilename(filepath.Base(p))
		if !part || e != "rar" || k != key {
			continue
		}
		if bestNum < 0 || n < bestNum {
			bestNum = n
			bestPath = p
		}
	}
	return bestPath
}

// VolumeIncompleteError is returned when not all parts of a set were selected for import.
func VolumeIncompleteError(g Group) error {
	return fmt.Errorf("分卷压缩包 %s 需同时选择全部分卷后再导入", g.DisplayName())
}
