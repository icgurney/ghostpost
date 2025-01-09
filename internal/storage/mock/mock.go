package mock

import (
	"context"
	"io"
	"sync"

	"ghostpost/internal/storage"
)

// Storage implements storage.Storage interface for testing
type Storage struct {
	storage.Storage
	mu          sync.RWMutex
	SavedEmails map[string]string
}

func NewStorage() *Storage {
	return &Storage{
		SavedEmails: make(map[string]string),
	}
}

func (m *Storage) SaveEmail(ctx context.Context, id string, body io.Reader) error {
	data, err := io.ReadAll(body)
	if err != nil {
		return err
	}

	m.mu.Lock()
	m.SavedEmails[id] = string(data)
	m.mu.Unlock()

	return nil
}

func (m *Storage) GetEmail(id string) (string, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	email, exists := m.SavedEmails[id]
	return email, exists
}
