package logname

import "testing"

func TestRotationIndex(t *testing.T) {
	cases := []struct {
		name string
		want int
	}{
		{"logcat.log", -1},
		{"logcat.log.001.7z", 1},
		{"logcat.log.007.7z", 7},
		{"logcat.log.002", 2},
	}
	for _, c := range cases {
		if got := RotationIndex(c.name); got != c.want {
			t.Errorf("RotationIndex(%q) = %d, want %d", c.name, got, c.want)
		}
	}
}

func TestSortStrings(t *testing.T) {
	names := []string{
		"logcat.log.007.7z",
		"logcat.log.001.7z",
		"logcat.log",
		"logcat.log.003.7z",
	}
	SortStrings(names)
	want := []string{"logcat.log", "logcat.log.001.7z", "logcat.log.003.7z", "logcat.log.007.7z"}
	for i := range want {
		if names[i] != want[i] {
			t.Fatalf("got %v, want %v", names, want)
		}
	}
}
