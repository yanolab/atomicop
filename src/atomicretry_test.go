package atomicop

import (
	"context"
	"errors"
	"sync"
	"testing"
)

type mockStateRepository struct {
	StateRepository

	MockGetState    func(ctx context.Context, key string) (*State, error)
	MockUpdateState func(ctx context.Context, key string, state State) error
}

func (r *mockStateRepository) GetState(ctx context.Context, key string) (*State, error) {
	return r.MockGetState(ctx, key)
}

func (r *mockStateRepository) UpdateState(ctx context.Context, key string, state State) error {
	return r.MockUpdateState(ctx, key, state)
}

func Test_RetryableOncer_at_once(t *testing.T) {
	var m sync.Map
	retryableOncer := NewRetryableOncer(5, NewOnce(NewSyncMapRepository()), &mockStateRepository{
		MockGetState: func(_ context.Context, key string) (*State, error) {
			st := &State{
				Attempts: 0,
				Value:    InitState,
			}
			act, _ := m.LoadOrStore(key, st)
			return act.(*State), nil
		},
		MockUpdateState: func(_ context.Context, key string, state State) error {
			m.Store(key, &state)
			return nil
		},
	})

	counter := 0
	for i := 0; i < 5; i++ {
		err := retryableOncer.Do(context.TODO(), "retryableKey", func() error {
			counter++
			return nil
		})
		if err != nil {
			t.Errorf("unexpected error: %s", err)
		}
	}

	if counter != 1 {
		t.Errorf("unexpected execution times: %d", counter)
	}
}

func Test_RetryableOncer_GetState(t *testing.T) {
	retryableOncer := NewRetryableOncer(5, NewOnce(NewSyncMapRepository()), &mockStateRepository{
		MockGetState: func(_ context.Context, key string) (*State, error) {
			return nil, errors.New("failed to get")
		},
		MockUpdateState: func(_ context.Context, key string, state State) error {
			return nil
		},
	})

	counter := 0
	for i := 0; i < 5; i++ {
		err := retryableOncer.Do(context.TODO(), "retryableKey", func() error {
			counter++
			return nil
		})
		if err == nil {
			t.Errorf("unexpected success")
		}
	}

	if counter != 0 {
		t.Errorf("unexpected execution times: %d", counter)
	}
}

func Test_RetryableOncer_retry_success(t *testing.T) {
	var m sync.Map
	retryableOncer := NewRetryableOncer(5, NewOnce(NewSyncMapRepository()), &mockStateRepository{
		MockGetState: func(_ context.Context, key string) (*State, error) {
			st := &State{
				Attempts: 0,
				Value:    InitState,
			}
			act, _ := m.LoadOrStore(key, st)
			return act.(*State), nil
		},
		MockUpdateState: func(_ context.Context, key string, state State) error {
			m.Store(key, &state)
			return nil
		},
	})

	counter := 0
	successAt := 3
	for i := 0; i < 5; i++ {
		err := retryableOncer.Do(context.TODO(), "retryableKey", func() error {
			counter++
			if counter < successAt {
				return &retryableError{errors.New("retry")}
			}
			return nil
		})
		if err != nil {
			if rerr, ok := err.(interface {
				CanRetry() bool
			}); ok && rerr.CanRetry() {
				continue
			}

			t.Fatalf("unexpected error: %s", err)
		}
	}

	if counter != successAt {
		t.Errorf("unexpected execution time: %d", counter)
	}
}

func Test_RetryableOncer_retry_failed(t *testing.T) {
	var m sync.Map
	retryableOncer := NewRetryableOncer(5, NewOnce(NewSyncMapRepository()), &mockStateRepository{
		MockGetState: func(_ context.Context, key string) (*State, error) {
			st := &State{
				Attempts: 0,
				Value:    InitState,
			}
			act, _ := m.LoadOrStore(key, st)
			return act.(*State), nil
		},
		MockUpdateState: func(_ context.Context, key string, state State) error {
			m.Store(key, &state)
			return nil
		},
	})

	counter := 0
	for i := 0; i < 10; i++ {
		retryableOncer.Do(context.TODO(), "retryableKey", func() error {
			counter++
			return errors.New("no retry")
		})
	}

	if counter != 1 {
		t.Errorf("unexpected execution time: %d", counter)
	}
}

func Test_RetryableOncer_retry_limit_exceeded(t *testing.T) {
	var m sync.Map
	retryLimit := 5
	retryableOncer := NewRetryableOncer(retryLimit, NewOnce(NewSyncMapRepository()), &mockStateRepository{
		MockGetState: func(_ context.Context, key string) (*State, error) {
			st := &State{
				Attempts: 0,
				Value:    InitState,
			}
			act, _ := m.LoadOrStore(key, st)
			return act.(*State), nil
		},
		MockUpdateState: func(_ context.Context, key string, state State) error {
			m.Store(key, &state)
			return nil
		},
	})

	counter := 0
	for i := 0; i < 10; i++ {
		retryableOncer.Do(context.TODO(), "retryableKey", func() error {
			counter++
			return &retryableError{errors.New("retry")}
		})
	}

	if counter != retryLimit {
		t.Errorf("unexpected execution time: %d", counter)
	}
}

func Test_retryableError(t *testing.T) {
	err := &retryableError{errors.New("error")}

	if err.CanRetry() != true {
		t.Errorf("unexpected retryable error: %s", err)
	}
	if err.Error() != "error" {
		t.Errorf("unexpected error: %s", err)
	}
}
