package atomicop

import (
	"context"
	"errors"
	"sync"
)

// SyncMapRepository implements SyncRepository interface using sync.Map
type SyncMapRepository struct {
	m sync.Map
}

// NewSyncMapRepository creates a SyncMapRepository instance
func NewSyncMapRepository() *SyncMapRepository {
	var m sync.Map
	return &SyncMapRepository{m: m}
}

// Store stores key.
// If key is already exists, this method returns duplicateError
func (r *SyncMapRepository) Store(ctx context.Context, key string) error {
	_, loaded := r.m.LoadOrStore(key, struct{}{})
	if loaded {
		return &duplicateError{errors.New("duplicated")}
	}

	return nil
}
