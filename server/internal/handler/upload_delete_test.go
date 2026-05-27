package handler

import (
	"testing"

	"log-tools/server/internal/model"
)

func TestFilterBatchDeleteRoots(t *testing.T) {
	items := []model.LogFile{
		{ID: "root", EntryType: model.EntryTypeFolder},
		{ID: "sub", EntryType: model.EntryTypeFolder, ParentID: "root"},
		{ID: "f1", EntryType: model.EntryTypeFile, ParentID: "sub"},
	}
	ids := []string{"root", "sub", "f1"}
	got := filterBatchDeleteRoots(ids, items)
	if len(got) != 1 || got[0] != "root" {
		t.Fatalf("want [root], got %v", got)
	}
}

func TestIsDeleteIgnorable(t *testing.T) {
	if isDeleteIgnorable(nil) {
		t.Fatal("nil should not be ignorable")
	}
	if !isDeleteIgnorable(errNotFound("file not found")) {
		t.Fatal("file not found should be ignorable")
	}
}

type errNotFound string

func (e errNotFound) Error() string { return string(e) }
