package atomicop

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
)

func Test_SyncMapRepository(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		n        int
		parallel bool
	}{
		"sequential": {
			n: 100,
		},
		"parallel": {
			n:        100,
			parallel: true,
		},
	}

	for tn, tc := range tests {
		t.Run(tn, func(t *testing.T) {
			oncer := NewOnce(NewSyncMapRepository())

			var counter = int32(0)
			fn := func() error {
				atomic.AddInt32(&counter, 1)
				return nil
			}

			var wg sync.WaitGroup
			var sem chan struct{}
			if tc.parallel {
				sem = make(chan struct{}, tc.n)
			} else {
				sem = make(chan struct{}, 1)
			}

			for i := 0; i < tc.n; i++ {
				wg.Add(1)
				go func() {
					sem <- struct{}{}
					defer func() {
						wg.Done()
						<-sem
					}()
					err := oncer.Do(context.TODO(), "same", fn)
					if (err != nil) != false {
						t.Errorf("unexpected error: %v", err)
					}
				}()
			}

			wg.Wait()
			close(sem)

			if counter != 1 {
				t.Errorf("unexpected execution times: %d", counter)
			}
		})
	}
}
