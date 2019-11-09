package atomicop

import (
	"context"
	"fmt"
)

// RetryableOncer supports retring for oncer.
type RetryableOncer struct {
	limit int
	oncer Oncer
	r     StateRepository
}

// StateValue is state
type StateValue int

const (
	// InitState indicates initial
	InitState StateValue = 1
	// DoneState indicates done
	DoneState = 2
	// FailedState indicate failed
	FailedState = 3
	// RetryState indicate need to retry
	RetryState = 4
)

// State is execution state
type State struct {
	Attempts int
	Value    StateValue
}

// NewRetryableOncer creates an instance for RetryableOncer
func NewRetryableOncer(limit int, oncer Oncer, r StateRepository) *RetryableOncer {
	return &RetryableOncer{
		limit: limit,
		oncer: oncer,
		r:     r,
	}
}

// StateRepository is an interface for retryable oncer
type StateRepository interface {
	// GetState gets state.
	GetState(ctx context.Context, key string) (*State, error)
	// UpdateState update state.
	UpdateState(ctx context.Context, key string, state State) error
}

// updateOnece update state at once using oncer.
func (r *RetryableOncer) updateOnce(ctx context.Context, idempotencyKey, updateKey string, attenmpts int, state StateValue) error {
	return r.oncer.Do(ctx, idempotencyKey, func() error {
		return r.r.UpdateState(ctx, updateKey, State{
			Attempts: attenmpts,
			Value:    state,
		})
	})
}

type retryableError struct {
	raw error
}

func (*retryableError) CanRetry() bool {
	return true
}

func (err *retryableError) Error() string {
	return err.raw.Error()
}

// Do execute function at once.
// If function had been executed and succeeded, This method do nothing.
// If function return retryable error which has to have CanRetry() bool method,
// it is saved to repository with state.
// Then Do is able to be executed again.
//
// Attention:
//   `key-\d+` is reserved by system. Therefore, You can not use that key.
func (r *RetryableOncer) Do(ctx context.Context, key string, fn func() error) error {
	current, err := r.r.GetState(ctx, key)
	if err != nil {
		// GetState failed would be retryable
		return &retryableError{err}
	}

	if current.Value == DoneState || current.Value == FailedState {
		return nil
	}

	current.Attempts++
	id := fmt.Sprintf("%s-%d", key, current.Attempts)

	// under here, retry or init state
	if current.Attempts > r.limit {
		r.updateOnce(ctx, id, key, current.Attempts, FailedState)
		return nil
	}

	return r.oncer.Do(ctx, id, func() error {
		err := fn()
		if retryableErr, ok := err.(interface {
			CanRetry() bool
		}); ok && retryableErr.CanRetry() {
			r.r.UpdateState(ctx, key, State{
				Attempts: current.Attempts,
				Value:    RetryState,
			})
			return err
		}
		if err != nil {
			return r.r.UpdateState(ctx, key, State{
				Attempts: current.Attempts,
				Value:    FailedState,
			})
		}

		return r.r.UpdateState(ctx, key, State{
			Attempts: current.Attempts,
			Value:    DoneState,
		})
	})
}
