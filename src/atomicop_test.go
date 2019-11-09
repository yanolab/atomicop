package atomicop

import (
	"context"
	"errors"
	"testing"
)

type mockSyncRepository struct {
	SyncRepository
	MockStore func(ctx context.Context, key string) error
}

func (r *mockSyncRepository) Store(ctx context.Context, key string) error {
	return r.MockStore(ctx, key)
}

type duplicateAndPropagateError struct {
}

func (e *duplicateAndPropagateError) Error() string {
	return "duplicate and propagate error"
}

func (e *duplicateAndPropagateError) Duplicate() bool {
	return true
}

func (e *duplicateAndPropagateError) Propagate() bool {
	return true
}

func Test_OnceDo(t *testing.T) {
	t.Parallel()

	type args struct {
		key string
		fn  func() error
	}
	tests := map[string]struct {
		args    args
		mock    func(ctx context.Context, key string) error
		wantErr bool
	}{
		"Success": {
			args: args{
				fn: func() error { return nil },
			},
			mock: func(_ context.Context, _ string) error {
				return nil
			},
		},
		"Success already exists": {
			args: args{
				fn: func() error { return nil },
			},
			mock: func(_ context.Context, _ string) error {
				return &duplicateError{errors.New("duplicated")}
			},
		},
		"Fail due to propagate error": {
			args: args{
				fn: func() error { return nil },
			},
			mock: func(_ context.Context, _ string) error {
				return &duplicateAndPropagateError{}
			},
			wantErr: true,
		},
		"Fail due to fn error": {
			args: args{
				fn: func() error { return errors.New("error") },
			},
			mock: func(_ context.Context, _ string) error {
				return nil
			},
			wantErr: true,
		},
		"Fail due to repository": {
			mock: func(_ context.Context, _ string) error {
				return errors.New("internal error")
			},
			wantErr: true,
		},
	}

	for tn, tc := range tests {
		t.Run(tn, func(t *testing.T) {
			oncer := NewOnce(&mockSyncRepository{
				MockStore: tc.mock,
			})
			err := oncer.Do(context.TODO(), tc.args.key, tc.args.fn)
			if (err != nil) != tc.wantErr {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func Test_duplicateError(t *testing.T) {
	err := &duplicateError{errors.New("error")}

	if err.Duplicate() != true {
		t.Errorf("unexpected duplicate error: %s", err)
	}
	if err.Error() != "error" {
		t.Errorf("unexpected error: %s", err)
	}
}
