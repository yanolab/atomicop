package atomicop

import (
	"context"
	"database/sql"
	"errors"
	"testing"
)

func prepare(t *testing.T) (*RetryableOncer, func() error) {
	db, err := sql.Open("mysql", "root:pass@tcp(127.0.0.1:3306)/atomicop")
	if err != nil {
		t.Fatal(err)
	}

	// prepare for the test
	if _, err := db.Exec("DELETE FROM atomicop WHERE true"); err != nil {
		t.Error("failed to cleanup table")
	}

	insertSQLBuilder := func(key string) (string, []interface{}) {
		return "INSERT INTO atomicop(id) VALUE(?)", []interface{}{key}
	}
	getStateSQLBuilder := func(key string) (string, []interface{}) {
		return "SELECT state, attempts FROM atomicop WHERE id = ?", []interface{}{key}
	}
	stateBinder := func(rows *sql.Rows, state *State) error {
		return rows.Scan(&state.Value, &state.Attempts)
	}
	updateStateSQLBuilder := func(key string, state State) (string, []interface{}) {
		q := `INSERT INTO atomicop(id, state, attempts) VALUES(?, ?, ?) ON DUPLICATE KEY
			  UPDATE state = VALUES(state), attempts = VALUES(attempts)`
		args := []interface{}{key, state.Value, state.Attempts}
		return q, args
	}

	r := NewMySQLRepository(
		db,
		insertSQLBuilder,
		getStateSQLBuilder,
		stateBinder,
		updateStateSQLBuilder,
	)
	return NewRetryableOncer(5, NewOnce(r), r), db.Close
}

func Test_RetryableOncer_at_once_with_MySQL(t *testing.T) {
	retryableOncer, closer := prepare(t)
	defer closer()

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

func Test_RetryableOncer_retry_success_with_MySQL(t *testing.T) {
	retryableOncer, closer := prepare(t)
	defer closer()

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

func Test_RetryableOncer_retry_failed_with_MySQL(t *testing.T) {
	retryableOncer, closer := prepare(t)
	defer closer()

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

func Test_RetryableOncer_retry_limit_exceeded_with_MySQL(t *testing.T) {
	retryableOncer, closer := prepare(t)
	defer closer()

	counter := 0
	for i := 0; i < 10; i++ {
		retryableOncer.Do(context.TODO(), "retryableKey", func() error {
			counter++
			return &retryableError{errors.New("retry")}
		})
	}

	if counter != 5 {
		t.Errorf("unexpected execution time: %d", counter)
	}
}
