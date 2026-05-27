package model

import "testing"

func TestFlattenedLogOriginalName(t *testing.T) {
	if got := FlattenedLogOriginalName("cat01.7z", "log20250809.log"); got != "cat01.7z.log" {
		t.Fatalf("got %q", got)
	}
	if got := FlattenedLogOriginalName("cat01_20260527_124556.7z", "log20250809.log"); got != "cat01.7z.log" {
		t.Fatalf("timestamp strip got %q", got)
	}
	if got := FlattenedLogOriginalName("cat01.7z", "logcat.log_1775463023.txt"); got != "cat01.7z.txt" {
		t.Fatalf("inner txt got %q", got)
	}
	if got := FlattenedLogOriginalName("logcat.log.003.zip", "logcat.log"); got != "logcat.log.003.log" {
		t.Fatalf("log archive + inner log got %q", got)
	}
	if got := FlattenedLogOriginalName("logcat.log.003.zip", "notes.txt"); got != "logcat.log.003.txt" {
		t.Fatalf("log archive + inner txt got %q", got)
	}
	if got := FlattenedLogOriginalName("logcat.log.zip", "logcat.log"); got != "logcat.log" {
		t.Fatalf("stem already .log got %q", got)
	}
	if got := FlattenedLogOriginalName("report.txt.002.zip", "data.txt"); got != "report.txt.002.txt" {
		t.Fatalf("txt rotation got %q", got)
	}
}

func TestArchiveStemLooksLikeLogName(t *testing.T) {
	if !archiveStemLooksLikeLogName("logcat.log.003") {
		t.Fatal("rotation log stem")
	}
	if !archiveStemLooksLikeLogName("logcat.log") {
		t.Fatal("plain log stem")
	}
	if archiveStemLooksLikeLogName("cat01") {
		t.Fatal("generic archive stem should not match")
	}
}
