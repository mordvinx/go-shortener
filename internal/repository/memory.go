package repository

import (
	"fmt"
	"sync"
)

type Memory struct {
	mu    sync.Mutex
	links map[string]string
}

func NewMemory() *Memory {
	return &Memory{links: make(map[string]string)}
}

func (m *Memory) Save(id, url string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.links[id] = url
	return nil
}

func (m *Memory) Get(id string) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	url, ok := m.links[id]
	if !ok {
		return "", fmt.Errorf("URL not found for short ID %q", id)
	}
	return url, nil
}
