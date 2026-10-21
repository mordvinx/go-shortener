package service

import (
	"fmt"
	"sync"
)

type Shortener struct {
	RWMutex sync.RWMutex
	Counter uint64
	Store   map[string]string
}

func (s *Shortener) Shorten(url string) (string, error) {
	s.RWMutex.Lock()
	defer s.RWMutex.Unlock()

	if s.Store == nil {
		s.Store = make(map[string]string)
	}

	id, err := GenerateID(s.Counter)

	if err == nil {
		s.Store[id] = url
		s.Counter++
	}

	return id, err
}

func (s *Shortener) Resolve(short string) (string, error) {
	s.RWMutex.RLock()
	defer s.RWMutex.RUnlock()

	url, ok := s.Store[short]

	if !ok {
		return "", fmt.Errorf("URL not found for short ID %q", short)
	}

	return url, nil
}
