package service

import "testing"

func TestFolderChainForNestedArchive(t *testing.T) {
	got := folderChainForNestedArchive(nil, "root.zip", true, nil)
	if len(got) != 1 || got[0] != "root.zip" {
		t.Fatalf("got %v", got)
	}
	got = folderChainForNestedArchive([]string{"root.zip"}, "", true, []string{"pkg"})
	if len(got) != 2 || got[0] != "root.zip" || got[1] != "pkg" {
		t.Fatalf("got %v", got)
	}
}
