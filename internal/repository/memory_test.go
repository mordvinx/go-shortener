package repository

import (
	"errors"
	"fmt"
	"sync"
	"testing"
)

func TestMemoryDuplicateID(t *testing.T) {
	store := NewMemory()
	const original = "https://example.com/original"
	if err := store.Save("id", original); err != nil {
		t.Fatal(err)
	}
	for _, url := range []string{original, "https://example.com/replacement"} {
		if err := store.Save("id", url); !errors.Is(err, ErrAlreadyExists) {
			t.Errorf("Save returned %v, want ErrAlreadyExists", err)
		}
	}
	if got, err := store.Get("id"); err != nil || got != original {
		t.Errorf("stored URL = (%q, %v), want %q", got, err, original)
	}
}

func TestMemoryConcurrentDuplicateID(t *testing.T) {
	store := NewMemory()
	const count = 20
	winners := make(chan string, count)
	var workers sync.WaitGroup
	for i := 0; i < count; i++ {
		workers.Add(1)
		go func(i int) {
			defer workers.Done()
			url := fmt.Sprintf("https://example.com/%d", i)
			err := store.Save("same-id", url)
			if err == nil {
				winners <- url
			} else if !errors.Is(err, ErrAlreadyExists) {
				t.Errorf("unexpected Save error: %v", err)
			}
		}(i)
	}
	workers.Wait()
	close(winners)
	if len(winners) != 1 {
		t.Fatalf("successful saves = %d, want 1", len(winners))
	}
	winner := <-winners
	if got, err := store.Get("same-id"); err != nil || got != winner {
		t.Errorf("stored URL = (%q, %v), want winner %q", got, err, winner)
	}
}

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
