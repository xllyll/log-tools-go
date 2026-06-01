package multivolume

import "testing"

func TestParseFilenamePartRar(t *testing.T) {
	key, ext, n, isPart := ParseFilename("OS_20260527_165714.part01.rar")
	if key != "OS_20260527_165714" || ext != "rar" || n != 1 || !isPart {
		t.Fatalf("part01: key=%q ext=%q n=%d isPart=%v", key, ext, n, isPart)
	}
	_, _, n2, _ := ParseFilename("OS_20260527_165714.part02.rar")
	if n2 != 2 {
		t.Fatalf("part02 num=%d", n2)
	}
}

func TestGroupFilenamesMultiVolume(t *testing.T) {
	names := []string{
		"OS_20260527_165714.part02.rar",
		"logcat.log",
		"OS_20260527_165714.part01.rar",
	}
	groups, standalone := GroupFilenames(names)
	if len(standalone) != 1 || standalone[0] != "logcat.log" {
		t.Fatalf("standalone=%v", standalone)
	}
	if len(groups) != 1 {
		t.Fatalf("groups=%d", len(groups))
	}
	if !groups[0].IsMultiVolume() || groups[0].DisplayName() != "OS_20260527_165714.rar" {
		t.Fatalf("group=%+v", groups[0])
	}
	if groups[0].Parts[0].Filename != "OS_20260527_165714.part01.rar" {
		t.Fatalf("sort order: %+v", groups[0].Parts)
	}
}

func TestGroupFilenamesSingleRar(t *testing.T) {
	groups, standalone := GroupFilenames([]string{"only.rar"})
	if len(groups) != 0 || len(standalone) != 1 {
		t.Fatalf("groups=%v standalone=%v", groups, standalone)
	}
}
