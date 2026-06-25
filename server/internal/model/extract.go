package model

// ExtractedFile is one log file produced from an upload or archive extraction.
type ExtractedFile struct {
	DiskPath        string
	OriginalName    string
	FileFormat      string
	ArchiveDirParts []string
}

// ArchiveExtractResult holds extracted logs and all folder chains to register in DB.
type ArchiveExtractResult struct {
	Files        []ExtractedFile
	FolderChains [][]string
	ExtractRoot  string // 解压目录；非压缩包直传时为空
}
