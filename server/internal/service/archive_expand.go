package service

import (
	"archive/zip"
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"

	"log-tools/server/internal/model"

	"github.com/mholt/archiver/v4"
)

func (s *StorageService) expandArchiveFromDisk(archivePath, archiveName string, parentFolders []string, extractRoot string, containerName string) (*model.ArchiveExtractResult, error) {
	archiveName = normalizeFolderPart(archiveName)
	parentFolders = normalizeFolderChain(parentFolders)
	tmpDir, err := os.MkdirTemp(extractRoot, "expand-*")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(tmpDir)

	if err := extractArchiveToDir(archivePath, tmpDir); err != nil {
		return nil, err
	}

	logPaths, archivePaths, err := scanExtractedTreeFiltered(tmpDir)
	if err != nil {
		return nil, err
	}

	logN, arcN := len(logPaths), len(archivePaths)

	walkParts := collectWalkPathParts(tmpDir)
	hasSubpath := false
	for _, p := range walkParts {
		if len(p) > 0 {
			hasSubpath = true
			break
		}
	}
	bindInner := archiveBindContainer(logN+arcN, hasSubpath)
	folderChains := collectFolderChains(parentFolders, containerName, bindInner, walkParts)

	if arcN == 0 && logN == 1 {
		rel := toSlashRel(tmpDir, logPaths[0])
		relDir := archiveDirParts(rel)
		if len(relDir) == 0 {
			files, err := s.buildFlattenedLogEntry(archiveName, logPaths[0], extractRoot, parentFolders)
			if err != nil {
				return nil, err
			}
			return &model.ArchiveExtractResult{Files: files, FolderChains: folderChains}, nil
		}
		chain := folderChainForLogInArchive(parentFolders, containerName, archiveName, true, relDir)
		files, err := s.buildExtractedLogEntry(logPaths[0], extractRoot, filepath.Base(logPaths[0]), chain)
		if err != nil {
			return nil, err
		}
		return &model.ArchiveExtractResult{Files: files, FolderChains: mergeFolderChains(folderChains, collectFolderChains(parentFolders, containerName, true, [][]string{relDir}))}, nil
	}

	targetFolders := folderChainInsideArchive(parentFolders, archiveName, logN, arcN)
	if len(targetFolders) == 0 && needsArchiveFolder(logN, arcN) && containerName != "" {
		targetFolders = []string{normalizeFolderPart(containerName)}
	}

	var out []model.ExtractedFile
	for _, logPath := range logPaths {
		rel := toSlashRel(tmpDir, logPath)
		inner := append(append([]string{}, targetFolders...), archiveDirParts(rel)...)
		base := filepath.Base(logPath)
		target, err := diskTarget(extractRoot, inner, base)
		if err != nil {
			continue
		}
		if err := copyFile(logPath, target); err != nil {
			continue
		}
		out = append(out, model.ExtractedFile{
			DiskPath:        target,
			OriginalName:    base,
			FileFormat:      strings.ToLower(filepath.Ext(base)),
			ArchiveDirParts: normalizeFolderChain(inner),
		})
	}
	for _, arcPath := range archivePaths {
		base := filepath.Base(arcPath)
		rel := toSlashRel(tmpDir, arcPath)
		parent := append(append([]string{}, targetFolders...), archiveDirParts(rel)...)
		chunk, err := s.expandArchiveFromDisk(arcPath, base, parent, extractRoot, "")
		if err != nil {
			continue
		}
		out = append(out, chunk.Files...)
		folderChains = mergeFolderChains(folderChains, chunk.FolderChains)
	}
	folderChains = mergeFolderChains(folderChains, collectFolderChains(parentFolders, containerName, bindInner, walkParts))
	return &model.ArchiveExtractResult{Files: out, FolderChains: folderChains}, nil
}

// buildFlattenedLogEntry: 包内仅 1 个日志 → 展平命名；仅继承外层多文件压缩包的文件夹链，不为本包建文件夹。
func (s *StorageService) buildFlattenedLogEntry(archiveName, logPath string, extractRoot string, parentFolders []string) ([]model.ExtractedFile, error) {
	innerLog := filepath.Base(logPath)
	origName := model.FlattenedLogOriginalName(archiveName, innerLog)
	format := logFormatForFlatten(innerLog, origName)
	folderBinding := inheritedFolderChain(parentFolders)
	target, err := diskTarget(extractRoot, folderBinding, sanitizeDiskName(origName))
	if err != nil {
		return nil, err
	}
	if err := copyFile(logPath, target); err != nil {
		return nil, err
	}
	log.Printf("[archive] flatten %s + %s => %s folders=%v", archiveName, innerLog, origName, folderBinding)
	return []model.ExtractedFile{{
		DiskPath:        target,
		OriginalName:    origName,
		FileFormat:      format,
		ArchiveDirParts: normalizeFolderChain(folderBinding),
	}}, nil
}

func (s *StorageService) buildExtractedLogEntry(logPath, extractRoot, originalName string, folderParts []string) ([]model.ExtractedFile, error) {
	folderParts = normalizeFolderChain(folderParts)
	target, err := diskTarget(extractRoot, folderParts, originalName)
	if err != nil {
		return nil, err
	}
	if err := copyFile(logPath, target); err != nil {
		return nil, err
	}
	ext := strings.ToLower(filepath.Ext(originalName))
	return []model.ExtractedFile{{
		DiskPath:        target,
		OriginalName:    originalName,
		FileFormat:      ext,
		ArchiveDirParts: folderParts,
	}}, nil
}

func logFormatForFlatten(innerLogName, flattenedName string) string {
	if ext := model.FileFormatFromName(innerLogName); ext != "" {
		return ext
	}
	if ext := model.FileFormatFromName(flattenedName); ext != "" {
		return ext
	}
	return ".log"
}

func toSlashRel(root, path string) string {
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return filepath.Base(path)
	}
	return filepath.ToSlash(rel)
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	return err
}

func extractArchiveToDir(archivePath, destDir string) error {
	ext := strings.ToLower(filepath.Ext(archivePath))
	if ext == ".zip" {
		return unzipToDir(archivePath, destDir)
	}
	return extractArchiverToDir(archivePath, destDir)
}

func unzipToDir(zipPath, destDir string) error {
	reader, err := zip.OpenReader(zipPath)
	if err != nil {
		return err
	}
	defer reader.Close()
	for _, f := range reader.File {
		if f.FileInfo().IsDir() {
			continue
		}
		if isArchiveNoisePath(f.Name) {
			continue
		}
		target := filepath.Join(destDir, filepath.FromSlash(filepath.ToSlash(f.Name)))
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		if err := extractZipEntry(f, target); err != nil {
			return err
		}
	}
	return nil
}

func extractArchiverToDir(archivePath, destDir string) error {
	file, err := os.Open(archivePath)
	if err != nil {
		return err
	}
	defer file.Close()
	format, _, err := archiver.Identify(context.Background(), archivePath, file)
	if err != nil {
		return err
	}
	ex, ok := format.(archiver.Extractor)
	if !ok {
		return fmt.Errorf("unsupported archive format: %s", filepath.Ext(archivePath))
	}
	_, _ = file.Seek(0, 0)
	return ex.Extract(context.Background(), file, func(_ context.Context, fi archiver.FileInfo) error {
		if fi.IsDir() {
			return nil
		}
		if isArchiveNoisePath(fi.Name()) {
			return nil
		}
		rel := filepath.ToSlash(fi.Name())
		target := filepath.Join(destDir, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		out, err := os.Create(target)
		if err != nil {
			return err
		}
		rc, err := fi.Open()
		if err != nil {
			out.Close()
			return err
		}
		_, copyErr := io.Copy(out, rc)
		rc.Close()
		out.Close()
		return copyErr
	})
}
