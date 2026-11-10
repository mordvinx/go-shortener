package repository

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestFileRestoreAndDuplicate(t *testing.T) {
	path := filepath.Join(t.TempDir(), "links.json")
	store, err := NewFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Save("first", "https://example.com/first"); err != nil {
		t.Fatal(err)
	}
	if err := store.Save("second", "https://example.com/second"); err != nil {
		t.Fatal(err)
	}
	if err := store.Save("first", "https://wrong.example"); !errors.Is(err, ErrAlreadyExists) {
		t.Fatalf("duplicate error=%v", err)
	}
	restored, err := NewFile(path)
	if err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{"first", "second"} {
		if got, err := restored.Get(id); err != nil || got != "https://example.com/"+id {
			t.Errorf("Get(%q)=(%q,%v)", id, got, err)
		}
	}
	if restored.NextCounter() != 2 {
		t.Errorf("next counter=%d", restored.NextCounter())
	}
	if _, err := restored.Get("unknown"); err == nil {
		t.Error("missing ID should return error")
	}
}

func TestFileRestoreMaximumSequence(t *testing.T) {
	path := filepath.Join(t.TempDir(), "links.json")
	data := `[{"uuid":"12","short_url":"a","original_url":"https://example.com/a"},{"uuid":"3","short_url":"b","original_url":"https://example.com/b"}]`
	if err := os.WriteFile(path, []byte(data), 0600); err != nil {
		t.Fatal(err)
	}
	store, err := NewFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if store.NextCounter() != 12 {
		t.Fatalf("counter=%d, want 12", store.NextCounter())
	}
	if err := store.Save("c", "https://example.com/c"); err != nil {
		t.Fatal(err)
	}
	restored, err := NewFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if restored.NextCounter() != 13 {
		t.Errorf("counter=%d, want 13", restored.NextCounter())
	}
}

func TestFileRejectsInvalidData(t *testing.T) {
	for _, data := range []string{`{`, `{}`, `[{"uuid":"bad","short_url":"a","original_url":"x"}]`} {
		path := filepath.Join(t.TempDir(), "links.json")
		if err := os.WriteFile(path, []byte(data), 0600); err != nil {
			t.Fatal(err)
		}
		if _, err := NewFile(path); err == nil {
			t.Errorf("accepted invalid file %s", data)
		}
		got, err := os.ReadFile(path)
		if err != nil || string(got) != data {
			t.Errorf("load changed invalid file: %q, %v", got, err)
		}
	}
}

func TestFileWriteError(t *testing.T) {
	path := filepath.Join(t.TempDir(), "links.json")
	store, err := NewFile(path)
	if err != nil {
		t.Fatal(err)
	}
	// A directory cannot be overwritten with JSON.
	if err := os.Mkdir(path, 0700); err != nil {
		t.Fatal(err)
	}
	if err := store.Save("new", "https://example.com/new"); err == nil {
		t.Fatal("expected write error")
	}
	if _, err := store.Get("new"); err == nil {
		t.Error("failed save became visible")
	}
	if store.NextCounter() != 0 {
		t.Error("failed save advanced counter")
	}
}

func TestFileReadError(t *testing.T) {
	if _, err := NewFile(t.TempDir()); err == nil {
		t.Error("expected error when reading a directory as a file")
	}
}
