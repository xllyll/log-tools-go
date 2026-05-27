package service

import (
	"archive/zip"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"log-tools/server/internal/model"
)

var archiveExtensions = map[string]bool{".zip": true, ".rar": true, ".7z": true}

func isLogExt(ext string) bool {
	return model.IsLogExtension(ext)
}

func isArchiveExt(ext string) bool {
	return archiveExtensions[strings.ToLower(ext)]
}

func flattenArchiveLogName(archiveEntryName, innerLogName string) string {
	return model.FlattenedLogOriginalName(archiveEntryName, innerLogName)
}

func diskTarget(extractRoot string, folderParts []string, fileName string) (string, error) {
	dir := extractRoot
	if len(folderParts) > 0 {
		dir = filepath.Join(extractRoot, filepath.FromSlash(strings.Join(folderParts, "/")))
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	return filepath.Join(dir, fileName), nil
}

func (s *StorageService) ExtractArchive(path, uploadOriginalName string) (*model.ArchiveExtractResult, error) {
	ext := strings.ToLower(filepath.Ext(path))
	if !isArchiveExt(ext) {
		orig := model.OriginalBaseName(uploadOriginalName)
		if orig == "" {
			orig = model.OriginalBaseName(path)
		}
		if !model.IsLogFileName(orig) {
			return nil, fmt.Errorf("unsupported file type: %s", ext)
		}
		return &model.ArchiveExtractResult{Files: []model.ExtractedFile{{
			DiskPath:     path,
			OriginalName: orig,
			FileFormat:   model.FileFormatFromName(orig),
		}}}, nil
	}

	extractRoot := filepath.Join(s.cfg.Storage.UploadDir, "extracted_"+time.Now().Format("20060102_150405"))
	if err := os.MkdirAll(extractRoot, 0o755); err != nil {
		return nil, err
	}
	containerName := model.OriginalBaseName(uploadOriginalName)
	return s.processArchiveAtPath(path, nil, extractRoot, containerName)
}

func (s *StorageService) processArchiveAtPath(archivePath string, parentFolders []string, extractRoot string, containerName string) (*model.ArchiveExtractResult, error) {
	ext := strings.ToLower(filepath.Ext(archivePath))
	switch ext {
	case ".zip":
		return s.processZipArchive(archivePath, parentFolders, extractRoot, containerName)
	case ".rar", ".7z":
		return s.expandArchiveFromDisk(archivePath, filepath.Base(archivePath), parentFolders, extractRoot, containerName)
	default:
		return nil, fmt.Errorf("unsupported archive: %s", ext)
	}
}

func (s *StorageService) processZipArchive(zipPath string, parentFolders []string, extractRoot string, containerName string) (*model.ArchiveExtractResult, error) {
	reader, err := zip.OpenReader(zipPath)
	if err != nil {
		return nil, err
	}
	defer reader.Close()

	pathPartsList := collectZipPathParts(&reader.Reader)

	itemCount := 0
	hasSubpath := false
	for _, f := range reader.File {
		if f.FileInfo().IsDir() || isArchiveNoisePath(f.Name) {
			continue
		}
		rel := filepath.ToSlash(f.Name)
		if parts := archiveDirParts(rel); len(parts) > 0 {
			hasSubpath = true
		}
		base := filepath.Base(rel)
		ext := strings.ToLower(filepath.Ext(base))
		if isLogExt(ext) || isArchiveExt(ext) {
			itemCount++
		}
	}
	bindContainer := archiveBindContainer(itemCount, hasSubpath)

	var out []model.ExtractedFile
	var nestedChains [][]string
	for _, f := range reader.File {
		if f.FileInfo().IsDir() {
			continue
		}
		rel := filepath.ToSlash(f.Name)
		if isArchiveNoisePath(rel) {
			continue
		}
		base := filepath.Base(rel)
		ext := strings.ToLower(filepath.Ext(base))
		relDir := archiveDirParts(rel)

		switch {
		case isLogExt(ext):
			folderBinding := folderChainForLogInArchive(parentFolders, containerName, "", bindContainer, relDir)
			target, err := diskTarget(extractRoot, folderBinding, base)
			if err != nil {
				continue
			}
			if err := extractZipEntry(f, target); err != nil {
				continue
			}
			out = append(out, model.ExtractedFile{
				DiskPath:        target,
				OriginalName:    base,
				FileFormat:      ext,
				ArchiveDirParts: normalizeFolderChain(folderBinding),
			})
		case isArchiveExt(ext):
			parentForNested := folderChainForNestedArchive(parentFolders, containerName, bindContainer, relDir)
			chunk, err := s.handleNestedZipMember(f, base, parentForNested, extractRoot)
			if err != nil {
				continue
			}
			out = append(out, chunk.Files...)
			nestedChains = mergeFolderChains(nestedChains, chunk.FolderChains)
		}
	}
	folderChains := mergeFolderChains(
		collectFolderChains(parentFolders, containerName, bindContainer, pathPartsList),
		nestedChains,
	)
	return &model.ArchiveExtractResult{Files: out, FolderChains: folderChains}, nil
}

func (s *StorageService) handleNestedZipMember(f *zip.File, archiveName string, parentFolders []string, extractRoot string) (*model.ArchiveExtractResult, error) {
	tmp, err := os.CreateTemp(extractRoot, "nested-*"+filepath.Ext(archiveName))
	if err != nil {
		return nil, err
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)
	if err := extractZipEntry(f, tmpPath); err != nil {
		tmp.Close()
		return nil, err
	}
	tmp.Close()

	return s.handleNestedArchiveFile(tmpPath, archiveName, parentFolders, extractRoot)
}

func (s *StorageService) handleNestedArchiveFile(archivePath, archiveName string, parentFolders []string, extractRoot string) (*model.ArchiveExtractResult, error) {
	return s.expandArchiveFromDisk(archivePath, archiveName, parentFolders, extractRoot, "")
}

func sanitizeDiskName(name string) string {
	return strings.Map(func(r rune) rune {
		if r == '/' || r == '\\' || r == ':' || r == '*' || r == '?' || r == '"' || r == '<' || r == '>' || r == '|' {
			return '_'
		}
		return r
	}, name)
}
