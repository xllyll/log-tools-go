package service

import "testing"

func TestNeedsArchiveFolder(t *testing.T) {
	if needsArchiveFolder(1, 0) {
		t.Fatal("single log should not need folder")
	}
	if !needsArchiveFolder(2, 0) {
		t.Fatal("two logs need folder")
	}
	if !needsArchiveFolder(1, 1) {
		t.Fatal("log + nested archive need folder")
	}
}

func TestArchiveBindContainer(t *testing.T) {
	if !archiveBindContainer(1, true) {
		t.Fatal("subpath alone should bind container")
	}
	if archiveBindContainer(1, false) {
		t.Fatal("single root entry should not bind")
	}
}

func TestFolderChainForLogInArchive(t *testing.T) {
	got := folderChainForLogInArchive(nil, "root.zip", "", true, nil)
	if len(got) != 1 || got[0] != "root.zip" {
		t.Fatalf("got %v", got)
	}
	got = folderChainForLogInArchive(nil, "root.zip", "", true, []string{"pkg"})
	if len(got) != 2 || got[0] != "root.zip" || got[1] != "pkg" {
		t.Fatalf("got %v", got)
	}
	if folderChainForLogInArchive(nil, "root.zip", "", false, nil) != nil {
		t.Fatal("single-item container should not bind folder")
	}
	got = inheritedFolderChain([]string{"root.zip"})
	if len(got) != 1 || got[0] != "root.zip" {
		t.Fatalf("flattened nested should inherit parent, got %v", got)
	}
}

func TestFolderChainNestedInSubpath(t *testing.T) {
	got := folderChainForNestedArchive(nil, "root.zip", false, []string{"pkg"})
	if len(got) != 2 || got[0] != "root.zip" || got[1] != "pkg" {
		t.Fatalf("single nested archive in subdir, got %v", got)
	}
}

func TestFolderChainInsideArchive(t *testing.T) {
	got := folderChainInsideArchive([]string{"root.zip"}, "oslog.zip", 2, 0)
	if len(got) != 2 || got[0] != "root.zip" || got[1] != "oslog.zip" {
		t.Fatalf("got %v", got)
	}
	got = folderChainInsideArchive([]string{"root.zip"}, "cat01.7z", 1, 0)
	if len(got) != 1 || got[0] != "root.zip" {
		t.Fatalf("single log archive should not add self folder, got %v", got)
	}
}

func TestFlattenArchiveLogName(t *testing.T) {
	if got := flattenArchiveLogName("cat01_20260527_124556.7z", "log20250809.log"); got != "cat01.7z.log" {
		t.Fatalf("got %q", got)
	}
}
