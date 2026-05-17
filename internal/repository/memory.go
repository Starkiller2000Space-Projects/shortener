package repository

import (
	"context"
	"sync"
)

type memStorage struct {
	mu   sync.RWMutex
	data map[string]urlInfo
}

// Create memory storage
func NewMemStorage() (*memStorage, error) {
	return &memStorage{
		data: make(map[string]urlInfo),
	}, nil // error not possible, but output is consistent with other storage types
}

// add url into storage and return generated id
func (m *memStorage) add(info Row) error {
	m.data[info.ID] = urlInfo{originalURL: info.OriginalURL, userID: info.UserID}
	return nil
}

// add url into storage and return generated id
func (m *memStorage) Add(ctx context.Context, info Row) error {
	select {
	case <-ctx.Done(): // cancel or deadline
		return ctx.Err()
	default:
	}
	m.mu.Lock()         // lock storage for writing
	defer m.mu.Unlock() // unlock storage for writing after function completion
	return m.add(info)
}

// get url by id from storage if exists
func (m *memStorage) Get(ctx context.Context, id string) (string, error) {
	select {
	case <-ctx.Done(): // cancel or deadline
		return "", ctx.Err()
	default:
	}
	m.mu.RLock()         // lock storage for reading
	defer m.mu.RUnlock() // unlock storage for reading after function completion
	info, ok := m.data[id]
	if !ok {
		return "", ErrNotFound
	}
	if info.deleted {
        return "", ErrGone
    }
	return info.originalURL, nil
}

// ping storage and check if it is available
func (m *memStorage) Ping(ctx context.Context) error {
	return nil // always available
}

// add multiple values as batch
func (m *memStorage) addBatch(items []Row) error {
	for _, item := range items {
		m.data[item.ID] = urlInfo{originalURL: item.OriginalURL, userID: item.UserID}
	}
	return nil
}

// add multiple values as batch with mutex
func (m *memStorage) AddBatch(ctx context.Context, items []Row) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}
	m.mu.Lock()         // lock storage for writing
	defer m.mu.Unlock() // unlock storage for writing after function completion
	return m.addBatch(items)
}

// get all urls added by current user
func (m *memStorage) GetUserURLs(ctx context.Context, userID string) ([]UserURL, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}
	m.mu.RLock()         // lock storage for reading
	defer m.mu.RUnlock() // unlock storage for reading after function completion
	result := []UserURL{}
	for id, info := range m.data {
		if info.userID == userID {
			result = append(result, UserURL{
				ShortURL:    id,
				OriginalURL: info.originalURL,
			})
		}
	}
	return result, nil
}

// delete batch by user id and short ids
func (m *memStorage) deleteBatch(userID string, shortIDs []string) error {
	for _, id := range shortIDs {
		if info, ok := m.data[id]; ok && info.userID == userID {
			info.deleted = true
			m.data[id] = info
		}
	}
	return nil
}

// delete batch by user id and short ids with mutex
func (m *memStorage) DeleteBatch(ctx context.Context, userID string, shortIDs []string) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}
	if len(shortIDs) == 0 {
		return nil
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.deleteBatch(userID, shortIDs)
}
