package atomicop

import (
	"database/sql"

	"github.com/go-sql-driver/mysql"
)

func mysqlErrorConvert(err error) error {
	if myerr, ok := err.(*mysql.MySQLError); ok {
		// duplicate entry
		if myerr.Number == 1062 {
			return &duplicateError{myerr}
		}
		return err
	}

	return err
}

// NewMySQLRepository creates an instance for MySQLRepository
func NewMySQLRepository(
	db *sql.DB,
	builder SQLBuilder,
	getStateSQLBuilder GetStateSQLBuilder,
	stateBinder StateBinder,
	updateStateSQLBuilder UpdateStateSQLBuilder,
) *MySQLRepository {
	return &MySQLRepository{
		SQLRepository: NewSQLRepository(
			db,
			builder,
			mysqlErrorConvert,
		),
		SQLStateRepository: NewSQLStateRepository(
			db,
			getStateSQLBuilder,
			stateBinder,
			updateStateSQLBuilder,
		),
	}
}

// MySQLRepository implements SyncRepository and StateRepository interface using MySQL
type MySQLRepository struct {
	*SQLRepository
	*SQLStateRepository
}
