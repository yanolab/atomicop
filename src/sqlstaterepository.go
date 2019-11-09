package atomicop

import (
	"context"
	"database/sql"
	"fmt"
)

// NewSQLStateRepository creates a SQLStateRepository instance.
func NewSQLStateRepository(
	db *sql.DB,
	getStateSQLBuilder GetStateSQLBuilder,
	stateBinder StateBinder,
	updateStateSQLBuilder UpdateStateSQLBuilder,
) *SQLStateRepository {
	return &SQLStateRepository{
		db:                    db,
		getStateSQLBuilder:    getStateSQLBuilder,
		stateBinder:           stateBinder,
		updateStateSQLBuilder: updateStateSQLBuilder,
	}
}

// GetStateSQLBuilder is an interface for building SQL query
type GetStateSQLBuilder func(key string) (query string, args []interface{})

// UpdateStateSQLBuilder is an interface for building SQL query
type UpdateStateSQLBuilder func(key string, state State) (query string, args []interface{})

// StateBinder binds state from rows
type StateBinder func(rows *sql.Rows, state *State) error

// SQLStateRepository implements StateRepository interface using SQL DB
type SQLStateRepository struct {
	db                    *sql.DB
	getStateSQLBuilder    GetStateSQLBuilder
	stateBinder           StateBinder
	updateStateSQLBuilder UpdateStateSQLBuilder
}

func (r *SQLStateRepository) GetState(ctx context.Context, key string) (state *State, err error) {
	opts := &sql.TxOptions{ReadOnly: true}
	tx, err := r.db.BeginTx(ctx, opts)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err != nil {
			tx.Rollback()
		}
	}()

	q, args := r.getStateSQLBuilder(key)
	rows, err := tx.QueryContext(ctx, q, args...)
	defer rows.Close()

	if err != nil {
		return nil, err
	}
	if !rows.Next() {
		return &State{0, InitState}, nil
	}

	state = new(State)
	if err := r.stateBinder(rows, state); err != nil {
		return nil, err
	}

	return state, nil
}

func (r *SQLStateRepository) UpdateState(ctx context.Context, key string, state State) error {
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

	q, args := r.updateStateSQLBuilder(key, state)
	result, err := tx.ExecContext(ctx, q, args...)
	if err != nil {
		return err
	}
	n, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if n < 1 {
		return fmt.Errorf("unexpected rows affected: %d", n)
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	return nil
}
