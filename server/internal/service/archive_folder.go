package service

import (
	"path/filepath"
	"strings"

	"log-tools/server/internal/model"
)

// archiveContentItemCount: 压缩包内待处理条目（日志文件 + 嵌套压缩包）。
func archiveContentItemCount(logCount, archiveCount int) int {
	return logCount + archiveCount
}

// needsArchiveFolder: 包内多于 1 个条目（含嵌套压缩包）时需要为本包建文件夹层。
func needsArchiveFolder(logCount, archiveCount int) bool {
	return archiveContentItemCount(logCount, archiveCount) > 1
}

// archiveBindContainer: 顶层压缩包是否需以容器名建文件夹（多文件或存在子目录路径）。
func archiveBindContainer(itemCount int, hasSubpath bool) bool {
	return itemCount > 1 || hasSubpath
}

func normalizeFolderPart(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return name
	}
	ext := strings.ToLower(filepath.Ext(name))
	if isArchiveExt(ext) {
		return model.NormalizeArchiveEntryName(name)
	}
	return name
}

// splitPathSegments 将路径拆成各级目录名（支持 / 与 \\）。
func splitPathSegments(path string) []string {
	path = strings.TrimSpace(filepath.ToSlash(path))
	if path == "" || path == "." {
		return nil
	}
	var out []string
	for _, seg := range strings.Split(path, "/") {
		seg = strings.TrimSpace(seg)
		if seg != "" && seg != "." {
			out = append(out, seg)
		}
	}
	return out
}

// normalizeFolderChain 统一文件夹链各段名称（压缩包名去时间戳等），并展开含反斜杠的段。
func normalizeFolderChain(parts []string) []string {
	if len(parts) == 0 {
		return nil
	}
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		for _, seg := range splitPathSegments(p) {
			seg = normalizeFolderPart(seg)
			if seg != "" && seg != "." {
				out = append(out, seg)
			}
		}
	}
	return out
}

// stripSingleArchiveRoot 若包内条目均在同一顶层目录下，去掉该目录，避免「容器名.rar / 包内同名目录」重复一层。
func stripSingleArchiveRoot(pathPartsList [][]string) ([][]string, string) {
	if len(pathPartsList) == 0 {
		return pathPartsList, ""
	}
	var root string
	for _, parts := range pathPartsList {
		p := normalizeFolderChain(parts)
		if len(p) == 0 {
			return pathPartsList, ""
		}
		if root == "" {
			root = p[0]
		} else if p[0] != root {
			return pathPartsList, ""
		}
	}
	hasNested := false
	for _, parts := range pathPartsList {
		if len(normalizeFolderChain(parts)) > 1 {
			hasNested = true
			break
		}
	}
	if !hasNested {
		return pathPartsList, ""
	}
	out := make([][]string, 0, len(pathPartsList))
	for _, parts := range pathPartsList {
		p := normalizeFolderChain(parts)
		if len(p) > 1 {
			out = append(out, p[1:])
		} else {
			out = append(out, nil)
		}
	}
	return out, root
}

func stripRootFromRelDir(relDir []string, root string) []string {
	relDir = normalizeFolderChain(relDir)
	if root == "" || len(relDir) == 0 {
		return relDir
	}
	if relDir[0] == root {
		return relDir[1:]
	}
	return relDir
}

// collectFolderChains 从压缩包内各路径收集需入库的文件夹链（含全部中间层级）。
func collectFolderChains(parentFolders []string, containerName string, bindContainer bool, pathPartsList [][]string) [][]string {
	pathPartsList, _ = stripSingleArchiveRoot(pathPartsList)
	seen := make(map[string]struct{})
	var chains [][]string

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
		chains = append(chains, chain)
	}

	if bindContainer {
		add(folderChainForLogInArchive(parentFolders, containerName, "", true, nil))
	}

	for _, parts := range pathPartsList {
		parts = normalizeFolderChain(parts)
		for i := 1; i <= len(parts); i++ {
			add(folderChainForLogInArchive(parentFolders, containerName, "", bindContainer, parts[:i]))
		}
	}
	return chains
}

func containerLabel(parentFolders []string, containerName, archiveName string) string {
	if containerName != "" && len(parentFolders) == 0 {
		return normalizeFolderPart(containerName)
	}
	if archiveName != "" && len(parentFolders) == 0 {
		return normalizeFolderPart(archiveName)
	}
	return ""
}

// folderChainForLogInArchive 计算日志文件在库中的文件夹路径。
func folderChainForLogInArchive(parentFolders []string, containerName, archiveName string, bindContainer bool, relDirParts []string) []string {
	parentFolders = normalizeFolderChain(parentFolders)
	relDirParts = normalizeFolderChain(relDirParts)

	if !bindContainer {
		if len(parentFolders) == 0 && len(relDirParts) == 0 {
			return nil
		}
		chain := append([]string{}, parentFolders...)
		if label := containerLabel(parentFolders, containerName, archiveName); label != "" {
			chain = append(chain, label)
		}
		chain = append(chain, relDirParts...)
		return chain
	}

	chain := append([]string{}, parentFolders...)
	if label := containerLabel(parentFolders, containerName, archiveName); label != "" {
		chain = append(chain, label)
	}
	chain = append(chain, relDirParts...)
	return chain
}

// folderChainForNestedArchive 嵌套压缩包展开时的父文件夹链。
func folderChainForNestedArchive(parentFolders []string, containerName string, bindContainer bool, relDirParts []string) []string {
	parentFolders = normalizeFolderChain(parentFolders)
	relDirParts = normalizeFolderChain(relDirParts)

	if !bindContainer {
		chain := append([]string{}, parentFolders...)
		if len(relDirParts) > 0 {
			if label := containerLabel(parentFolders, containerName, ""); label != "" {
				chain = append(chain, label)
			}
			chain = append(chain, relDirParts...)
		}
		return chain
	}

	chain := append([]string{}, parentFolders...)
	if label := containerLabel(parentFolders, containerName, ""); label != "" {
		chain = append(chain, label)
	}
	chain = append(chain, relDirParts...)
	return chain
}

// inheritedFolderChain 单日志压缩包展平时继承的外层文件夹（不为本包再建文件夹）。
func inheritedFolderChain(parentFolders []string) []string {
	return normalizeFolderChain(parentFolders)
}

// folderChainInsideArchive 包内多文件时，为本压缩包再增加一层文件夹名。
func folderChainInsideArchive(parentFolders []string, archiveName string, logCount, archiveCount int) []string {
	parentFolders = normalizeFolderChain(parentFolders)
	if !needsArchiveFolder(logCount, archiveCount) {
		return parentFolders
	}
	archiveName = normalizeFolderPart(archiveName)
	if len(parentFolders) == 0 {
		return []string{archiveName}
	}
	return append(append([]string{}, parentFolders...), archiveName)
}
