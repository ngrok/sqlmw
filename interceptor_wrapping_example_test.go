package sqlmw_test

import (
	"context"
	"database/sql/driver"
	"fmt"
	"time"

	"github.com/ngrok/sqlmw"
)

type MyInterceptor struct {
	sqlmw.NullInterceptor
}

// MyStmt wraps a Stmt and annotates it with additional user-defined data.
type MyStmt struct {
	driver.Stmt
	startTime time.Time
	query     string
}

func (m *MyInterceptor) ConnPrepareContext(ctx context.Context, conn driver.ConnPrepareContext, query string) (driver.Stmt, error) {
	stmt, err := conn.PrepareContext(ctx, query)
	if err != nil {
		return nil, err
	}

	// wrap the original Stmt and store additional a query string and a
	// timestamp
	return &MyStmt{Stmt: stmt, startTime: time.Now(), query: query}, nil
}

func (m *MyInterceptor) StmtQueryContext(ctx context.Context, stmt *sqlmw.Stmt, _ string, args []driver.NamedValue) (driver.Rows, error) {
	// convert stmt to our StmtWithTs and access the data that we stored
	// in the ConnPrepareContext() call.
	if sq, ok := stmt.Parent().(*MyStmt); ok {
		fmt.Printf("running StmtQueryContext for query: %s, start time: %s", sq.query, sq.startTime)
	}

	return stmt.QueryContext(ctx, args)
}

// Example demonstrates how data that is created in a ConnPrepareContext()
// call can be made available on methods of a Stmt.
// This work analagous for Rows and Tx objects.
func Example() {}
