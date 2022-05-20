package sqlmw

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"testing"
)

type stmtCtxKey string

const (
	stmtRowContextKey   stmtCtxKey = "rowcontext"
	stmtRowContextValue string     = "rowvalue"
)

type stmtTestInterceptor struct {
	T              *testing.T
	RowsNextValid  bool
	RowsCloseValid bool
	NullInterceptor
}

func (i *stmtTestInterceptor) StmtQueryContext(ctx context.Context, stmt driver.StmtQueryContext, _ string, args []driver.NamedValue) (context.Context, driver.Rows, error) {
	ctx = context.WithValue(ctx, stmtRowContextKey, stmtRowContextValue)

	r, err := stmt.QueryContext(ctx, args)
	return ctx, r, err
}

func (i *stmtTestInterceptor) RowsNext(ctx context.Context, rows driver.Rows, dest []driver.Value) error {
	if ctx.Value(stmtRowContextKey) == stmtRowContextValue {
		i.RowsNextValid = true
	}

	i.T.Log(ctx)

	return rows.Next(dest)
}

func (i *stmtTestInterceptor) RowsClose(ctx context.Context, rows driver.Rows) error {
	if ctx.Value(stmtRowContextKey) == stmtRowContextValue {
		i.RowsCloseValid = true
	}

	i.T.Log(ctx)

	return rows.Close()
}

func TestStmtQueryContext_PassWrappedRowContext(t *testing.T) {
	driverName := driverName(t)

	con := &fakeConn{}
	fakeStmt := &fakeStmt{
		rows: &fakeRows{
			con:  con,
			vals: [][]driver.Value{{}},
		},
	}
	con.stmt = fakeStmt

	ti := &stmtTestInterceptor{T: t}

	sql.Register(
		driverName,
		Driver(&fakeDriver{conn: con}, ti),
	)

	db, err := sql.Open(driverName, "")
	if err != nil {
		t.Fatalf("Failed to open: %v", err)
	}

	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Errorf("Failed to close db: %v", err)
		}
	})

	stmt, err := db.PrepareContext(context.Background(), "")
	if err != nil {
		t.Fatalf("Prepare failed: %s", err)
	}

	rows, err := stmt.Query("")
	if err != nil {
		t.Fatalf("Stmt query failed: %s", err)
	}

	rows.Next()
	rows.Close()
	stmt.Close()

	if !ti.RowsNextValid {
		t.Error("RowsNext context not valid")
	}
	if !ti.RowsCloseValid {
		t.Error("RowsClose context not valid")
	}
}
