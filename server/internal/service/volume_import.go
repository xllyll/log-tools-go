package service

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"log-tools/server/internal/pkg/multivolume"
)

// ImportMultiVolumeArchive imports a multi-part archive when all parts are already on disk in the same directory.
func (s *StorageService) ImportMultiVolumeArchive(ctx context.Context, deviceID, displayName string, partPaths []string) ([]string, error) {
	if len(partPaths) == 0 {
		return nil, fmt.Errorf("no volume parts")
	}
	dir := filepath.Dir(partPaths[0])
	names := make([]string, len(partPaths))
	for i, p := range partPaths {
		names[i] = filepath.Base(p)
		if filepath.Dir(p) != dir {
			return nil, fmt.Errorf("分卷文件必须在同一目录")
		}
	}
	groups, _ := multivolume.GroupFilenames(names)
	if len(groups) != 1 {
		return nil, fmt.Errorf("分卷压缩包识别失败")
	}
	g := groups[0]
	if !g.IsMultiVolume() {
		return nil, multivolume.VolumeIncompleteError(g)
	}
	label := displayName
	if label == "" {
		label = g.DisplayName()
	}
	first := multivolume.FindFirstVolumePath(dir, partPaths[0])
	return s.ImportSavedFile(ctx, deviceID, first, label)
}

// SaveVolumePart writes one volume part under volDir using its original base name.
func SaveVolumePart(volDir string, filename string, src io.Reader) (string, error) {
	partPath := filepath.Join(volDir, filepath.Base(filename))
	out, err := os.Create(partPath)
	if err != nil {
		return "", err
	}
	defer out.Close()
	if _, err := io.Copy(out, src); err != nil {
		return "", err
	}
	return partPath, nil
}

// PrepareVolumeDir creates a temp directory for volume parts.
func (s *StorageService) PrepareVolumeDir() (string, error) {
	if err := os.MkdirAll(s.cfg.Storage.UploadDir, 0o755); err != nil {
		return "", err
	}
	return os.MkdirTemp(s.cfg.Storage.UploadDir, "upload-vol-*")
}
