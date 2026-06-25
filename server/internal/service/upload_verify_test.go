package service

import (
	"strings"
	"testing"
)

func TestVerifyDownloadSize(t *testing.T) {
	if err := VerifyDownloadSize(100, 100, "a.zip"); err != nil {
		t.Fatalf("expected ok: %v", err)
	}
	if err := VerifyDownloadSize(0, 100, "a.zip"); err == nil || !strings.Contains(err.Error(), "为空") {
		t.Fatalf("expected empty error, got %v", err)
	}
	if err := VerifyDownloadSize(50, 100, "a.zip"); err == nil || !strings.Contains(err.Error(), "不完整") {
		t.Fatalf("expected incomplete error, got %v", err)
	}
	if err := VerifyDownloadSize(100, 0, "a.zip"); err != nil {
		t.Fatalf("unknown size should pass: %v", err)
	}
}

func TestFindUniqueFileByBaseName(t *testing.T) {
	root := t.TempDir()
	if _, err := findUniqueFileByBaseName(root, "missing.log"); err == nil {
		t.Fatal("expected missing error")
	}
}
