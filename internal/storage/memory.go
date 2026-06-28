package storage

import (
	"context"
	"sync"

	"github.com/emityoffwhite/url-shortener/internal/model"
)

// MemoryStorage - потокобезопасное in-memory хранилище ссылок.
// Использует RWMutex: чтения (Get) намного чаще записей (Save),
// поэтому RWMutex эффективнее обычного Mutex - несколько чтений могут идти параллельно.
type MemoryStorage struct {
	mu   sync.RWMutex
	data map[string]*model.URL
}

// NewMemoryStorage создаёт новое in-memory хранилище.
func NewMemoryStorage() *MemoryStorage {
	return &MemoryStorage{
		data: make(map[string]*model.URL),
	}
}

func (s *MemoryStorage) Save(_ context.Context, u *model.URL) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.data[u.ShortCode]; exists {
		return ErrAlreadyExists
	}

	// Сохраняем копию, чтобы внешние изменения структуры не влияли на хранилище.
	copied := *u
	s.data[u.ShortCode] = &copied
	return nil
}

func (s *MemoryStorage) Get(_ context.Context, shortCode string) (*model.URL, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	u, ok := s.data[shortCode]
	if !ok {
		return nil, ErrNotFound
	}

	copied := *u
	return &copied, nil
}

func (s *MemoryStorage) IncrementClicks(_ context.Context, shortCode string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	u, ok := s.data[shortCode]
	if !ok {
		return ErrNotFound
	}
	u.Clicks++
	return nil
}

func (s *MemoryStorage) Delete(_ context.Context, shortCode string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.data[shortCode]; !ok {
		return ErrNotFound
	}
	delete(s.data, shortCode)
	return nil
}
