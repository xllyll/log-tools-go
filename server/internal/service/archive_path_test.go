package service

import "testing"

func TestSplitPathSegments(t *testing.T) {
	got := splitPathSegments(`OS_xxx\os\kernel`)
	if len(got) != 3 || got[0] != "OS_xxx" || got[2] != "kernel" {
		t.Fatalf("got %v", got)
	}
}

func TestNormalizeFolderChainSplitsBackslash(t *testing.T) {
	got := normalizeFolderChain([]string{`OS_xxx\aee`})
	if len(got) != 2 || got[0] != "OS_xxx" || got[1] != "aee" {
		t.Fatalf("got %v", got)
	}
}

func TestCollectFolderChainsPrefixes(t *testing.T) {
	chains := collectFolderChains(nil, "root.zip", true, [][]string{
		{"OS_xxx", "os"},
		{"OS_xxx", "aee"},
	})
	keys := map[string]bool{}
	for _, c := range chains {
		keys[stringsJoin(c)] = true
	}
	for _, want := range []string{
		"root.zip",
		"root.zip/OS_xxx",
		"root.zip/OS_xxx/os",
		"root.zip/OS_xxx/aee",
	} {
		if !keys[want] {
			t.Fatalf("missing chain %q in %v", want, keys)
		}
	}
}

func stringsJoin(parts []string) string {
	s := ""
	for i, p := range parts {
		if i > 0 {
			s += "/"
		}
		s += p
	}
	return s
}
