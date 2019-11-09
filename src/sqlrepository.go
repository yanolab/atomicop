package atomicop

import (
	"context"
	"database/sql"
	"fmt"

	_ "github.com/go-sql-driver/mysql"
)

// SQLErrorConverter is funcation for converting error
type SQLErrorConverter func(err error) error

// SQLBuilder is an interface for building SQL query
type SQLBuilder func(key string) (query string, args []interface{})

// SQLRepository implements SyncRepository interface using SQL DB
type SQLRepository struct {
	db      *sql.DB
	builder SQLBuilder
	conv    SQLErrorConverter
}

// NewSQLRepository creates a SQLRepository instance.
func NewSQLRepository(db *sql.DB, builder SQLBuilder, conv SQLErrorConverter) *SQLRepository {
	return &SQLRepository{
		db:      db,
		builder: builder,
		conv:    conv,
	}
}

func (r *SQLRepository) save(ctx context.Context, key string) (err error) {
	opts := &sql.TxOptions{ReadOnly: false}
	tx, err := r.db.BeginTx(ctx, opts)
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			tx.Rollback()
		}
	}()

	q, args := r.builder(key)
	result, err := tx.ExecContext(ctx, q, args...)
	if err != nil {
		return err
	}
	n, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if n != 1 {
		return fmt.Errorf("unexpected rows affected: %d", n)
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	return nil
}

// Store stores key using SQL DB.
func (r *SQLRepository) Store(ctx context.Context, key string) error {
	return r.conv(r.save(ctx, key))
}
