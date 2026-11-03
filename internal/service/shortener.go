package service

import (
	"fmt"
	"sync"
)

type Repository interface {
	Save(id, url string) error
	Get(id string) (string, error)
}

type Shortener struct {
	mu         sync.Mutex
	counter    uint64
	repository Repository
}

func NewShortener(repository Repository) *Shortener {
	return &Shortener{repository: repository}
}

func (s *Shortener) Shorten(url string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	id, err := generateID(s.counter)
	if err != nil {
		return "", err
	}
	if err := s.repository.Save(id, url); err != nil {
		return "", fmt.Errorf("save shortened URL: %w", err)
	}
	s.counter++
	return id, nil
}

func (s *Shortener) Resolve(short string) (string, error) {
	return s.repository.Get(short)
}
