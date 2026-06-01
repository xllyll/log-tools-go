package service

import (
	"archive/zip"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"log-tools/server/internal/model"
	"log-tools/server/internal/pkg/multivolume"
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
	result, err := s.processArchiveAtPath(path, nil, extractRoot, containerName)
	if err != nil {
		return nil, err
	}
	removeUploadedArchive(path)
	return result, nil
}

// removeUploadedArchive 解压完成后删除 uploads 下的原始压缩包，避免重复占盘
func removeUploadedArchive(path string) {
	if path == "" {
		return
	}
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		log.Printf("[extract] remove uploaded archive %s: %v", path, err)
		return
	}
	log.Printf("[extract] removed uploaded archive %s", filepath.Base(path))
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
	archiveNames := make([]string, 0)
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
		if isArchiveExt(ext) {
			archiveNames = append(archiveNames, base)
		}
	}
	bindContainer := archiveBindContainer(itemCount, hasSubpath)

	volumeGroups, _ := multivolume.GroupFilenames(archiveNames)
	volumePartNames := make(map[string]struct{})
	for _, g := range volumeGroups {
		for _, p := range g.Parts {
			volumePartNames[p.Filename] = struct{}{}
		}
	}

	var out []model.ExtractedFile
	var nestedChains [][]string

	for _, g := range volumeGroups {
		if !g.IsMultiVolume() {
			return nil, multivolume.VolumeIncompleteError(g)
		}
		relDir := zipMemberRelDir(reader.File, g.Parts[0].Filename)
		parentForNested := folderChainForNestedArchive(parentFolders, containerName, bindContainer, relDir)
		chunk, err := s.extractZipNestedVolumeGroup(reader, g, parentForNested, extractRoot)
		if err != nil {
			log.Printf("[extract] zip nested volume %s in %s: %v", g.DisplayName(), filepath.Base(zipPath), err)
			return nil, err
		}
		out = append(out, chunk.Files...)
		nestedChains = mergeFolderChains(nestedChains, chunk.FolderChains)
	}

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
				log.Printf("[extract] zip log %s: %v", base, err)
				continue
			}
			out = append(out, model.ExtractedFile{
				DiskPath:        target,
				OriginalName:    base,
				FileFormat:      ext,
				ArchiveDirParts: normalizeFolderChain(folderBinding),
			})
		case isArchiveExt(ext):
			if _, inVolume := volumePartNames[base]; inVolume {
				continue
			}
			parentForNested := folderChainForNestedArchive(parentFolders, containerName, bindContainer, relDir)
			chunk, err := s.handleNestedZipMember(f, base, parentForNested, extractRoot)
			if err != nil {
				log.Printf("[extract] zip nested %s: %v", base, err)
				continue
			}
			out = append(out, chunk.Files...)
			nestedChains = mergeFolderChains(nestedChains, chunk.FolderChains)
		}
	}
	if len(out) == 0 {
		log.Printf("[extract] zip %s: no log files extracted (entries: %d)", filepath.Base(zipPath), len(reader.File))
	}
	folderChains := mergeFolderChains(
		collectFolderChains(parentFolders, containerName, bindContainer, pathPartsList),
		nestedChains,
	)
	return &model.ArchiveExtractResult{Files: out, FolderChains: folderChains}, nil
}

func zipMemberRelDir(files []*zip.File, baseName string) []string {
	for _, f := range files {
		if f.FileInfo().IsDir() || isArchiveNoisePath(f.Name) {
			continue
		}
		if filepath.Base(filepath.ToSlash(f.Name)) == baseName {
			return archiveDirParts(filepath.ToSlash(f.Name))
		}
	}
	return nil
}

func findZipMemberByBaseName(files []*zip.File, baseName string) *zip.File {
	for _, f := range files {
		if f.FileInfo().IsDir() || isArchiveNoisePath(f.Name) {
			continue
		}
		if filepath.Base(filepath.ToSlash(f.Name)) == baseName {
			return f
		}
	}
	return nil
}

func (s *StorageService) extractZipNestedVolumeGroup(
	reader *zip.ReadCloser,
	g multivolume.Group,
	parentFolders []string,
	extractRoot string,
) (*model.ArchiveExtractResult, error) {
	volDir, err := os.MkdirTemp(extractRoot, "nested-vol-*")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(volDir)

	for _, p := range g.Parts {
		zf := findZipMemberByBaseName(reader.File, p.Filename)
		if zf == nil {
			return nil, fmt.Errorf("zip 内缺少分卷 %s", p.Filename)
		}
		dest := filepath.Join(volDir, filepath.Base(p.Filename))
		if err := extractZipEntry(zf, dest); err != nil {
			return nil, err
		}
	}
	first := filepath.Join(volDir, multivolume.FirstPartFilename(g))
	return s.expandArchiveFromDisk(first, g.DisplayName(), parentFolders, extractRoot, "")
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
