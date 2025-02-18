package sqlx

import (
	"context"
	"database/sql"

	"github.com/kunlun-qilian/sqlx/v3/scanner"
)

type ScanIterator = scanner.ScanIterator

func Scan(rows *sql.Rows, v interface{}) error {
	if err := scanner.Scan(context.Background(), rows, v); err != nil {
		if err == scanner.RecordNotFound {
			return NewSqlError(sqlErrTypeNotFound, "record is not found")
		}
		return err
	}
	return nil
}
