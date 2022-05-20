package sqlmw

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"testing"
)

type connCtxKey string

const (
	connRowContextKey    connCtxKey = "context"
	connRowContextValue  string     = "value"
	connStmtContextKey   connCtxKey = "stmtcontext"
	connStmtContextValue string     = "stmtvalue"
	connTxContextKey     connCtxKey = "txcontext"
	connTxContextValue   string     = "txvalue"
)

type connTestInterceptor struct {
	T               *testing.T
	RowsNextValid   bool
	RowsCloseValid  bool
	StmtCloseValid  bool
	TxCommitValid   bool
	TxRollbackValid bool
	NullInterceptor
}

func (i *connTestInterceptor) ConnPrepareContext(ctx context.Context, conn driver.ConnPrepareContext, query string) (context.Context, driver.Stmt, error) {
	ctx = context.WithValue(ctx, connStmtContextKey, connStmtContextValue)

	s, err := conn.PrepareContext(ctx, query)
	return ctx, s, err
}

func (i *connTestInterceptor) ConnQueryContext(ctx context.Context, conn driver.QueryerContext, query string, args []driver.NamedValue) (context.Context, driver.Rows, error) {
	ctx = context.WithValue(ctx, connRowContextKey, connRowContextValue)

	r, err := conn.QueryContext(ctx, query, args)
	return ctx, r, err
}

func (i *connTestInterceptor) ConnBeginTx(ctx context.Context, conn driver.ConnBeginTx, txOpts driver.TxOptions) (context.Context, driver.Tx, error) {
	ctx = context.WithValue(ctx, connTxContextKey, connTxContextValue)

	t, err := conn.BeginTx(ctx, txOpts)
	return ctx, t, err
}

func (i *connTestInterceptor) RowsNext(ctx context.Context, rows driver.Rows, dest []driver.Value) error {
	if ctx.Value(connRowContextKey) == connRowContextValue {
		i.RowsNextValid = true
	}

	return rows.Next(dest)
}

func (i *connTestInterceptor) RowsClose(ctx context.Context, rows driver.Rows) error {
	if ctx.Value(connRowContextKey) == connRowContextValue {
		i.RowsCloseValid = true
	}

	return rows.Close()
}

func (i *connTestInterceptor) StmtClose(ctx context.Context, stmt driver.Stmt) error {
	if ctx.Value(connStmtContextKey) == connStmtContextValue {
		i.StmtCloseValid = true
	}

	i.T.Log(ctx)

	return stmt.Close()
}

func (i *connTestInterceptor) TxCommit(ctx context.Context, tx driver.Tx) error {
	if ctx.Value(connTxContextKey) == connTxContextValue {
		i.TxCommitValid = true
	}

	i.T.Log(ctx)

	return tx.Commit()
}

func (i *connTestInterceptor) TxRollback(ctx context.Context, tx driver.Tx) error {
	if ctx.Value(connTxContextKey) == connTxContextValue {
		i.TxRollbackValid = true
	}

	i.T.Log(ctx)

	return tx.Rollback()
}

func TestConnQueryContext_PassWrappedRowContext(t *testing.T) {
	driverName := driverName(t)

	con := &fakeConn{}

	ti := &connTestInterceptor{T: t}

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

	rows, err := db.QueryContext(context.Background(), "")
	if err != nil {
		t.Fatalf("Prepare failed: %s", err)
	}

	rows.Next()
	rows.Close()

	if !ti.RowsCloseValid {
		t.Error("RowsClose context not valid")
	}
	if !ti.RowsNextValid {
		t.Error("RowsNext context not valid")
	}
}

func TestConnPrepareContext_PassWrappedStmtContext(t *testing.T) {
	driverName := driverName(t)

	con := &fakeConn{}
	fakeStmt := &fakeStmt{
		rows: &fakeRows{
			con:  con,
			vals: [][]driver.Value{{}},
		},
	}
	con.stmt = fakeStmt

	ti := &connTestInterceptor{T: t}

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

	stmt.Close()

	if !ti.StmtCloseValid {
		t.Error("StmtClose context not valid")
	}
}

func TestConnBeginTx_PassWrappedTxContextCommit(t *testing.T) {
	driverName := driverName(t)

	con := &fakeConn{}
	fakeTx := &fakeTx{}
	con.tx = fakeTx

	ti := &connTestInterceptor{T: t}

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

	tx, err := db.BeginTx(context.Background(), &sql.TxOptions{})
	if err != nil {
		t.Fatalf("Prepare failed: %s", err)
	}

	err = tx.Commit()
	if err != nil {
		t.Fatalf("Commit failed: %s", err)
	}

	if !ti.TxCommitValid {
		t.Error("TxCommit context not valid")
	}
}
func TestConnBeginTx_PassWrappedTxContextRollback(t *testing.T) {
	driverName := driverName(t)

	con := &fakeConn{}
	fakeTx := &fakeTx{}
	con.tx = fakeTx

	ti := &connTestInterceptor{T: t}

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

	tx, err := db.BeginTx(context.Background(), &sql.TxOptions{})
	if err != nil {
		t.Fatalf("Prepare failed: %s", err)
	}

	err = tx.Rollback()
	if err != nil {
		t.Fatalf("Rollback failed: %s", err)
	}

	if !ti.TxRollbackValid {
		t.Error("TxRollback context not valid")
	}
}
