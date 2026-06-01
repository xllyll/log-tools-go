package model

import "testing"

func TestIsLogRotatedName(t *testing.T) {
	cases := []struct {
		name string
		want bool
	}{
		{"logcat.log.01", true},
		{"logcat.log.24", true},
		{"logcat.log.001", true},
		{"report.TXT.02", true},
		{"data.json.10", true},
		{"logcat.log", false},
		{"logcat.log.7z", false},
		{"archive.log.01.7z", false},
	}
	for _, tc := range cases {
		if got := IsLogRotatedName(tc.name); got != tc.want {
			t.Fatalf("%q: got %v want %v", tc.name, got, tc.want)
		}
	}
}

func TestLogFormatFromNameRotated(t *testing.T) {
	if got := LogFormatFromName("logcat.log.01"); got != ".log" {
		t.Fatalf("got %q", got)
	}
	if got := LogFormatFromName("report.txt.24"); got != ".txt" {
		t.Fatalf("got %q", got)
	}
}
