package repository

import "testing"

func TestMemory(t *testing.T) {
	store := NewMemory()
	if _, err := store.Get("missing"); err == nil {
		t.Error("expected an error for unknown ID")
	}
	for id, original := range map[string]string{
		"first":  "https://example.com/first",
		"second": "https://example.com/second",
	} {
		if err := store.Save(id, original); err != nil {
			t.Fatal(err)
		}
		got, err := store.Get(id)
		if err != nil || got != original {
			t.Errorf("Get(%q) = (%q, %v), want %q", id, got, err, original)
		}
	}
	other := NewMemory()
	if _, err := other.Get("first"); err == nil {
		t.Error("independent stores must not share links")
	}
}
