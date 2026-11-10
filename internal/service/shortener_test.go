package service

import (
	"errors"
	"fmt"
	"sync"
	"testing"

	"github.com/mordvinx/go-shortener/internal/repository"
)

type failingRepository struct{ err error }

func (r failingRepository) Save(string, string) error  { return r.err }
func (r failingRepository) Get(string) (string, error) { return "", r.err }

func TestShortenerRepositoryErrors(t *testing.T) {
	want := errors.New("storage unavailable")
	s := NewShortener(failingRepository{err: want}, 0)
	if id, err := s.Shorten("https://example.com/"); id != "" || !errors.Is(err, want) {
		t.Errorf("Shorten returned (%q, %v), want empty ID and storage error", id, err)
	}
	if url, err := s.Resolve("unknown"); url != "" || !errors.Is(err, want) {
		t.Errorf("Resolve returned (%q, %v), want empty URL and storage error", url, err)
	}
}

func TestShortenerConcurrentRoundTrip(t *testing.T) {
	s := NewShortener(repository.NewMemory(), 0)
	const count = 50
	ids := make(chan string, count)
	var workers sync.WaitGroup
	for i := 0; i < count; i++ {
		workers.Add(1)
		go func(i int) {
			defer workers.Done()
			original := fmt.Sprintf("https://example.com/%d", i)
			id, err := s.Shorten(original)
			if err != nil {
				t.Errorf("Shorten: %v", err)
				return
			}
			ids <- id
			got, err := s.Resolve(id)
			if err != nil || got != original {
				t.Errorf("Resolve(%q) = (%q, %v), want %q", id, got, err, original)
			}
		}(i)
	}
	workers.Wait()
	close(ids)
	seen := make(map[string]bool)
	for id := range ids {
		if seen[id] {
			t.Errorf("duplicate ID: %q", id)
		}
		seen[id] = true
	}
	if len(seen) != count {
		t.Errorf("got %d unique IDs, want %d", len(seen), count)
	}
}
