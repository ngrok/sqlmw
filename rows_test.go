package sqlmw

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"log"
	"testing"
)

type rowsCloseInterceptor struct {
	NullInterceptor

	rowsCloseCalled  bool
	rowsCloseLastCtx context.Context
}

func (r *rowsCloseInterceptor) RowsClose(ctx context.Context, rows driver.Rows) error {
	r.rowsCloseCalled = true
	r.rowsCloseLastCtx = ctx

	return rows.Close()
}

func TestRowsClose(t *testing.T) {
	driverName := t.Name()
	interceptor := rowsCloseInterceptor{}

	con := fakeConn{}
	sql.Register(driverName, Driver(&fakeDriver{conn: &con}, &interceptor))

	db, err := sql.Open(driverName, "")
	if err != nil {
		t.Fatalf("opening db failed: %s", err)
	}

	ctx := context.Background()
	ctxKey := "ctxkey"
	ctxVal := "1"

	ctx = context.WithValue(ctx, ctxKey, ctxVal) // nolint: staticcheck // not using a custom type for the ctx key is not an issue here

	rows, err := db.QueryContext(ctx, "", "")
	if err != nil {
		t.Fatalf("db.Query failed: %s", err)
	}

	err = rows.Close()
	if err != nil {
		t.Errorf("rows Close failed: %s", err)
	}

	if !interceptor.rowsCloseCalled {
		t.Error("interceptor rows.Close was not called")
	}

	if interceptor.rowsCloseLastCtx == nil {
		t.Fatal("rows close ctx is nil")
	}

	v := interceptor.rowsCloseLastCtx.Value(ctxKey)
	if v == nil {
		t.Fatalf("ctx is different, missing value for key: %s", ctxKey)
	}

	vStr, ok := v.(string)
	if !ok {
		t.Fatalf("ctx is different, value for key: %s, has type %t, expected string", ctxKey, v)
	}

	if ctxVal != vStr {
		t.Errorf("ctx is different, value for key: %s, is %q, expected %q", ctxKey, vStr, ctxVal)
	}

	if !con.rowsCloseCalled {
		t.Fatalf("rows close of driver was not called")
	}
}

type rowsNextInterceptor struct {
	NullInterceptor

	rowsNextCalled  bool
	rowsNextLastCtx context.Context
}

func (r *rowsNextInterceptor) RowsNext(ctx context.Context, rows driver.Rows, dest []driver.Value) error {
	r.rowsNextCalled = true
	r.rowsNextLastCtx = ctx
	return rows.Next(dest)
}

func TestRowsNext(t *testing.T) {
	con := &fakeConn{}
	rows := &fakeRows{vals: [][]driver.Value{{"hello", "world"}}, con: con}
	stmt := fakeStmt{
		rows: rows,
	}
	con.stmt = stmt
	driverName := t.Name()
	interceptor := rowsNextInterceptor{}

	sql.Register(
		driverName,
		Driver(&fakeDriver{conn: con}, &interceptor),
	)

	db, err := sql.Open(driverName, "")
	if err != nil {
		t.Fatalf("opening db failed: %s", err)
	}

	ctx := context.Background()
	ctxKey := "ctxkey"
	ctxVal := "1"

	ctx = context.WithValue(ctx, ctxKey, ctxVal) // nolint: staticcheck // not using a custom type for the ctx key is not an issue here

	rs, err := db.QueryContext(ctx, "", "")
	if err != nil {
		t.Fatalf("db.Query failed: %s", err)
	}

	var id, name string
	for rs.Next() {
		err := rs.Scan(&id, &name)
		if err != nil {
			log.Fatal(err)
		}
	}

	err = rs.Close()
	if err != nil {
		t.Errorf("rows Close failed: %s", err)
	}

	if !rows.nextCalled {
		t.Error("driver rows.Next was not called")
	}

	if !interceptor.rowsNextCalled {
		t.Error("interceptor rows.Next was not called")
	}

	if !interceptor.rowsNextCalled {
		t.Error("interceptor rows.Next was not called")
	}

	if interceptor.rowsNextLastCtx == nil {
		t.Fatal("rows close ctx is nil")
	}

	v := interceptor.rowsNextLastCtx.Value(ctxKey)
	if v == nil {
		t.Fatalf("ctx is different, missing value for key: %s", ctxKey)
	}

	vStr, ok := v.(string)
	if !ok {
		t.Fatalf("ctx is different, value for key: %s, has type %t, expected string", ctxKey, v)
	}

	if ctxVal != vStr {
		t.Errorf("ctx is different, value for key: %s, is %q, expected %q", ctxKey, vStr, ctxVal)
	}
}
