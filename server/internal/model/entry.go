package model

const (
	EntryTypeFile   = "file"
	EntryTypeFolder = "folder"
)

func (f *LogFile) IsFolder() bool {
	return f.EntryType == EntryTypeFolder
}

func (f *LogFile) IsFile() bool {
	return f.EntryType == "" || f.EntryType == EntryTypeFile
}
