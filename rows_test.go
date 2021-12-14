package sqlmw

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"fmt"
	"reflect"
	"sync/atomic"
	"testing"
)

var driverCount = int32(0)

func driverName(t *testing.T) string {
	c := atomic.LoadInt32(&driverCount)
	name := fmt.Sprintf("driver-%s-%d", t.Name(), c)
	c++
	atomic.StoreInt32(&driverCount, c)

	return name
}

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
	driverName := driverName(t)
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
	driverName := driverName(t)
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
			t.Fatal(err)
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

func TestRows_LikePGX(t *testing.T) {
	strType := reflect.TypeOf("")
	con := &fakeConn{}
	rs := fakeRows{vals: [][]driver.Value{{"hello", "world"}}, con: con}
	rows := &fakeRowsLikePgx{
		fakeRows:                       rs,
		fakeWithColumnTypeDatabaseName: fakeWithColumnTypeDatabaseName{r: &rs, names: []string{"CUSTOMVARCHAR", "CUSTOMVARCHAR"}},
		fakeWithColumnTypeScanType:     fakeWithColumnTypeScanType{r: &rs, scanTypes: []reflect.Type{strType, strType}},
		fakeWithColumnTypeNullable:     fakeWithColumnTypeNullable{r: &rs, nullables: []bool{false, false}, oks: []bool{true, true}},
		fakeWithColumnTypeLength:       fakeWithColumnTypeLength{r: &rs, lengths: []int64{5, 5}, bools: []bool{true, true}},
		fakeWithColumnTypePrecisionScale: fakeWithColumnTypePrecisionScale{
			r:          &rs,
			precisions: []int64{0, 0},
			scales:     []int64{0, 0},
			bools:      []bool{false, false},
		},
	}

	stmt := fakeStmt{
		rows: rows,
	}
	con.stmt = stmt
	driverName := driverName(t)
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

	qrs, err := db.QueryContext(ctx, "", "")
	if err != nil {
		t.Fatalf("db.Query failed: %s", err)
	}

	names, err := qrs.Columns()
	if err != nil {
		t.Errorf("error calling Columns, %v", err)
	}

	cts, err := qrs.ColumnTypes()
	if err != nil {
		t.Errorf("error calling ColumnTypes, %v", err)
	}

	if len(names) != 2 || len(names) != len(cts) {
		t.Errorf("wrong column name or types count")
	}

	// There's no real way these can be called, but we'll test in case the
	// test implementation changes
	if rs.hasNextResultSetCalled {
		t.Errorf("HasNextResultSetCalled, on non-supporting type, %v", err)
	}
	if rs.nextResultSetCalled {
		t.Errorf("NextResultSetCalled, on non-supporting type, %v", err)
	}

	if !rs.columnTypeDatabaseNameCalled {
		t.Errorf("ColumnTypeDatabaseName not called, %v", err)
	}

	if !rs.columnTypeLengthCalled {
		t.Errorf("ColumnTypeTypeLenght not called, %v", err)
	}

	if !rs.columnTypeNullable {
		t.Errorf("ColumnTypeTypeLenght not called, %v", err)
	}

	if !rs.columnTypePrecisionScaleCalled {
		t.Errorf("ColumnTypePrecisionScale not called, %v", err)
	}

	var id, name string
	for qrs.Next() {
		err := qrs.Scan(&id, &name)
		if err != nil {
			t.Fatal(err)
		}
	}

	err = qrs.Close()
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

func TestWrapRows(t *testing.T) {
	ctx := context.Background()
	tt := []struct {
		name string
		rows driver.Rows
	}{
		{
			name: "vanilla",
			rows: &fakeRows{},
		},
		{
			name: "pgx",
			rows: &fakeRowsLikePgx{},
		},
		{
			name: "mysql",
			rows: &fakeRowsLikeMysql{},
		},
	}

	for _, st := range tt {
		st := st
		t.Run(st.name, func(t *testing.T) {
			rows := st.rows
			wr := wrapRows(ctx, nil, rows)

			_, rok := rows.(driver.RowsNextResultSet)
			_, wok := wr.(driver.RowsNextResultSet)
			if rok != wok {
				t.Fatalf("inconsistent support for driver.RowsNextResultSet")
			}

			_, rok = rows.(driver.RowsColumnTypeDatabaseTypeName)
			_, wok = wr.(driver.RowsColumnTypeDatabaseTypeName)
			if rok != wok {
				t.Fatalf("inconsinstent support for driver.RowsColumnTypeDatabaseTypeName")
			}

			_, rok = rows.(driver.RowsColumnTypeLength)
			_, wok = wr.(driver.RowsColumnTypeLength)
			if rok != wok {
				t.Fatalf("inconsinstent support for driver.RowsColumnTypeLength")
			}

			_, rok = rows.(driver.RowsColumnTypeNullable)
			_, wok = wr.(driver.RowsColumnTypeNullable)
			if rok != wok {
				t.Fatalf("inconsinstent support for driver.RowsColumnTypeNullable")
			}

			_, rok = rows.(driver.RowsColumnTypeScanType)
			_, wok = wr.(driver.RowsColumnTypeScanType)
			if rok != wok {
				t.Fatalf("inconsinstent support for driver.RowsColumnTypeScanType")
			}

			_, rok = rows.(driver.RowsColumnTypePrecisionScale)
			_, wok = wr.(driver.RowsColumnTypePrecisionScale)
			if rok != wok {
				t.Fatalf("inconsinstent support for driver.RowsColumnTypePrecisionScale")
			}
		})
	}
}
