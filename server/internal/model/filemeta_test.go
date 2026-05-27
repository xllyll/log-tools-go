package model

import "testing"

func TestFlattenedLogOriginalName(t *testing.T) {
	if got := FlattenedLogOriginalName("cat01.7z", "log20250809.log"); got != "cat01.7z.log" {
		t.Fatalf("got %q", got)
	}
	if got := FlattenedLogOriginalName("cat01_20260527_124556.7z", "log20250809.log"); got != "cat01.7z.log" {
		t.Fatalf("timestamp strip got %q", got)
	}
	if got := FlattenedLogOriginalName("cat01.7z", "logcat.log_1775463023.txt"); got != "cat01.7z.log" {
		t.Fatalf("got %q", got)
	}
}
