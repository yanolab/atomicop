package atomicop

import (
	"context"
	"database/sql"
	"sync"
	"sync/atomic"
	"testing"
)

func Test_MySQLSyncRepository(t *testing.T) {
	t.Parallel()

	db, err := sql.Open("mysql", "root:pass@tcp(127.0.0.1:3306)/atomicop")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	// prepare for the test
	if _, err := db.Exec("DELETE FROM atomicop WHERE true"); err != nil {
		t.Error("failed to cleanup table")
	}

	insertSQLBuilder := func(key string) (string, []interface{}) {
		return "INSERT INTO atomicop(id) VALUE(?)", []interface{}{key}
	}

	r := NewMySQLRepository(db, insertSQLBuilder, nil, nil, nil)
	tests := map[string]struct {
		n        int
		key      string
		parallel bool
	}{
		"sequential": {
			n:   100,
			key: "seqKey",
		},
		"parallel": {
			n:        100,
			key:      "parKey",
			parallel: true,
		},
	}

	for tn, tc := range tests {
		t.Run(tn, func(t *testing.T) {
			oncer := NewOnce(r)

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
					err := oncer.Do(context.TODO(), tc.key, fn)
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
