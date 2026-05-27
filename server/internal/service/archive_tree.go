package service

import (
	"archive/zip"
	"os"
	"path/filepath"
	"strings"
)

func zipEntryPathParts(name string, isDir bool) []string {
	rel := filepath.ToSlash(name)
	if isArchiveNoisePath(rel) {
		return nil
	}
	if isDir {
		rel = strings.TrimSuffix(rel, "/")
		return splitPathSegments(rel)
	}
	return archiveDirParts(rel)
}

func collectZipPathParts(reader *zip.Reader) [][]string {
	var list [][]string
	for _, f := range reader.File {
		parts := zipEntryPathParts(f.Name, f.FileInfo().IsDir())
		if len(parts) > 0 {
			list = append(list, parts)
		}
	}
	return list
}

func collectWalkPathParts(root string) [][]string {
	var list [][]string
	_ = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel := toSlashRel(root, path)
		if isArchiveNoisePath(rel) {
			return nil
		}
		var parts []string
		if d.IsDir() {
			if rel == "." {
				return nil
			}
			parts = splitPathSegments(rel)
		} else {
			parts = archiveDirParts(rel)
		}
		if len(parts) > 0 {
			list = append(list, parts)
		}
		return nil
	})
	return list
}

func mergeFolderChains(a, b [][]string) [][]string {
	seen := make(map[string]struct{})
	var out [][]string
	add := func(chain []string) {
		chain = normalizeFolderChain(chain)
		if len(chain) == 0 {
			return
		}
		key := strings.Join(chain, "\x00")
		if _, ok := seen[key]; ok {
			return
		}
		seen[key] = struct{}{}
		out = append(out, chain)
	}
	for _, c := range a {
		add(c)
	}
	for _, c := range b {
		add(c)
	}
	return out
}
