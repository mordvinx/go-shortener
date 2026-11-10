package repository

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strconv"
	"sync"
)

type fileRecord struct {
	UUID        string `json:"uuid"`
	ShortURL    string `json:"short_url"`
	OriginalURL string `json:"original_url"`
}

type File struct {
	mu      sync.Mutex
	path    string
	records []fileRecord
	links   map[string]string
	next    uint64
}

func NewFile(path string) (*File, error) {
	f := &File{path: path, records: make([]fileRecord, 0), links: make(map[string]string)}
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return f, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read storage: %w", err)
	}
	if len(bytes.TrimSpace(data)) == 0 {
		return f, nil
	}
	if err := json.Unmarshal(data, &f.records); err != nil {
		return nil, fmt.Errorf("decode storage: %w", err)
	}
	for _, record := range f.records {
		number, err := strconv.ParseUint(record.UUID, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("parse storage sequence: %w", err)
		}
		f.links[record.ShortURL] = record.OriginalURL
		if number > f.next {
			f.next = number
		}
	}
	return f, nil
}

// NextCounter is passed to the service at startup, before serving requests.
// UUIDs start at 1, while generator counters start at 0.
func (f *File) NextCounter() uint64 {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.next
}

func (f *File) Save(id, url string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if _, exists := f.links[id]; exists {
		return fmt.Errorf("save short ID %q: %w", id, ErrAlreadyExists)
	}
	record := fileRecord{UUID: strconv.FormatUint(f.next+1, 10), ShortURL: id, OriginalURL: url}
	records := append(f.records, record)
	data, err := json.Marshal(records)
	if err != nil {
		return fmt.Errorf("encode storage: %w", err)
	}
	if err := os.WriteFile(f.path, data, 0600); err != nil {
		return fmt.Errorf("write storage: %w", err)
	}
	f.records = records
	f.links[id] = url
	f.next++
	return nil
}

func (f *File) Get(id string) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	url, ok := f.links[id]
	if !ok {
		return "", fmt.Errorf("URL not found for short ID %q", id)
	}
	return url, nil
}
