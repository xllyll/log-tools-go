package service

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRemovePhysicalFolderDirs(t *testing.T) {
	root := t.TempDir()
	extractRoot := filepath.Join(root, "extracted_20260101_120000")
	pkgDir := filepath.Join(extractRoot, "archive.zip", "pkg")
	if err := os.MkdirAll(pkgDir, 0o755); err != nil {
		t.Fatal(err)
	}
	logPath := filepath.Join(pkgDir, "a.log")
	if err := os.WriteFile(logPath, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	removePhysicalFolderDirs(root, []string{"archive.zip", "pkg"}, []string{logPath})
	if _, err := os.Stat(pkgDir); !os.IsNotExist(err) {
		t.Fatalf("pkg dir should be removed: %v", err)
	}
	if _, err := os.Stat(logPath); !os.IsNotExist(err) {
		t.Fatal("log file should be removed")
	}
}

func TestFindExtractRoot(t *testing.T) {
	root := t.TempDir()
	extractRoot := filepath.Join(root, "extracted_20260101_120000")
	p := filepath.Join(extractRoot, "a.zip", "b.log")
	got := findExtractRoot(p)
	if got != extractRoot {
		t.Fatalf("got %q want %q", got, extractRoot)
	}
}

func TestPruneEmptyParents(t *testing.T) {
	root := t.TempDir()
	leaf := filepath.Join(root, "a", "b", "c")
	if err := os.MkdirAll(leaf, 0o755); err != nil {
		t.Fatal(err)
	}
	pruneEmptyParents(leaf, root)
	if _, err := os.Stat(filepath.Join(root, "a")); !os.IsNotExist(err) {
		t.Fatal("empty chain should be pruned")
	}
}
