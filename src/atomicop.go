package atomicop

import (
	"context"
)

// Oncer is a general interface of once executor
// If same key is gave, Do method assures that fn executes at once.
type Oncer interface {
	Do(ctx context.Context, key string, fn func() error) error
}

// Once is an implementation of Oncer
// Once uses a SyncRepository for atomic operation
type Once struct {
	Oncer
	r SyncRepository
}

// NewOnce creates an Once instance
func NewOnce(r SyncRepository) *Once {
	return &Once{r: r}
}

// SyncRepository is a repository
// This repository assure that key is already exists or not.
// If key is already exists, Store must return an error which has Duplicate() bool method.
type SyncRepository interface {
	Store(ctx context.Context, key string) error
}

type duplicateError struct {
	raw error
}

func (*duplicateError) Duplicate() bool {
	return true
}

func (err *duplicateError) Error() string {
	return err.raw.Error()
}

// Do execute function at once
// If function had been executed, This method do nothing.
// However the error has Propagate method and the method returns true,
// Do returns error. otherwise returns nil.
// If Store is failed or fn returned an error, Do will return error.
func (o *Once) Do(ctx context.Context, key string, fn func() error) error {
	err := o.r.Store(ctx, key)
	if err != nil {
		if v, ok := err.(interface {
			Duplicate() bool
		}); ok && v.Duplicate() {
			if v, ok := err.(interface {
				Propagate() bool
			}); ok && v.Propagate() {
				return err
			}
			return nil
		}

		return err
	}

	if err := fn(); err != nil {
		return err
	}

	return nil
}
