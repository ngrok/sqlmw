package sqlmw

import (
	"context"
	"database/sql/driver"
	"fmt"
	"io"
	"reflect"
)

type fakeDriver struct {
	conn driver.Conn
}

func (d *fakeDriver) Open(_ string) (driver.Conn, error) {
	return d.conn, nil
}

type fakeStmt struct {
	rows   driver.Rows
	called bool // nolint:structcheck // ignore unused warning, it is accessed via reflection
}

type fakeStmtWithCheckNamedValue struct {
	fakeStmt
}

type fakeStmtWithoutCheckNamedValue struct {
	fakeStmt
}

func (s fakeStmt) Close() error {
	return nil
}

func (s fakeStmt) NumInput() int {
	return 1
}

func (s fakeStmt) Exec(_ []driver.Value) (driver.Result, error) {
	return nil, nil
}

func (s fakeStmt) Query(_ []driver.Value) (driver.Rows, error) {
	return s.rows, nil
}

func (s *fakeStmtWithCheckNamedValue) CheckNamedValue(_ *driver.NamedValue) (err error) {
	s.called = true
	return
}

type fakeRows struct {
	con         *fakeConn
	vals        [][]driver.Value
	closeCalled bool // nolint:structcheck,unused // ignore unused warning, it is accessed via reflection
	nextCalled  bool // nolint:structcheck,unused // ignore unused warning, it is accessed via reflection
}

func (r *fakeRows) Close() error {
	r.con.rowsCloseCalled = true
	r.closeCalled = true
	return nil
}

func (r *fakeRows) Columns() []string {
	if len(r.vals) == 0 {
		return nil
	}

	var cols []string
	for i := range r.vals[0] {
		cols = append(cols, fmt.Sprintf("col%d", i))
	}
	return cols
}

func (r *fakeRows) Next(dest []driver.Value) error {
	r.nextCalled = true
	if len(r.vals) == 0 {
		return io.EOF
	}
	copy(dest, r.vals[0])
	r.vals = r.vals[1:]
	return nil
}

type fakeWithRowsNextResultSet struct {
	hasNextResultSetCalled bool // nolint:structcheck,unused // ignore unused warning, it is accessed via reflection
	nextResultSetCalled    bool // nolint:structcheck,unused // ignore unused warning, it is accessed via reflection
}

func (f *fakeWithRowsNextResultSet) HasNextResultSet() bool {
	f.hasNextResultSetCalled = true
	return false
}

func (f *fakeWithRowsNextResultSet) NextResultSet() error {
	f.nextResultSetCalled = true
	return nil
}

type fakeWithColumnTypeDatabaseName struct {
	columnTypeDatabaseNameCalled bool // nolint:structcheck,unused // ignore unused warning, it is accessed via reflection
}

func (f *fakeWithColumnTypeDatabaseName) ColumnTypeDatabaseTypeName(i int) string {
	f.columnTypeDatabaseNameCalled = true
	return "sometype"
}

type fakeWithColumnTypeLength struct {
	columnTypeLengthCalled bool // nolint:structcheck,unused // ignore unused warning, it is accessed via reflection
}

func (f *fakeWithColumnTypeLength) ColumnTypeLength(index int) (length int64, ok bool) {
	f.columnTypeLengthCalled = true
	return 4, true
}

type fakeWithColumnTypePrecisionScale struct {
	columnTypePrecisionScaleCalled bool // nolint:structcheck,unused // ignore unused warning, it is accessed via reflection
}

func (f *fakeWithColumnTypePrecisionScale) ColumnTypePrecisionScale(index int) (precision, scale int64, ok bool) {
	f.columnTypePrecisionScaleCalled = true
	return 0, 0, false
}

type fakeWithColumnTypeNullable struct {
	columnTypeNullable bool // nolint:structcheck,unused // ignore unused warning, it is accessed via reflection
}

func (f *fakeWithColumnTypeNullable) ColumnTypeNullable(i int) (nullable, ok bool) {
	f.columnTypeNullable = true
	return false, true
}

type fakeWithColumnTypeScanType struct {
	columnTypeScanTypeCalled bool // nolint:structcheck,unused // ignore unused warning, it is accessed via reflection
}

func (f *fakeWithColumnTypeScanType) ColumnTypeScanType(i int) reflect.Type {
	f.columnTypeScanTypeCalled = true
	return reflect.TypeOf("")
}

type fakeRowsLikeMysql struct {
	fakeRows
	fakeWithRowsNextResultSet
	fakeWithColumnTypeDatabaseName
	fakeWithColumnTypePrecisionScale
	fakeWithColumnTypeNullable
	fakeWithColumnTypeScanType
}

type fakeRowsLikePgx struct {
	fakeRows
	fakeWithColumnTypeDatabaseName
	fakeWithColumnTypeLength
	fakeWithColumnTypePrecisionScale
	fakeWithColumnTypeNullable
	fakeWithColumnTypeScanType
}

type fakeRowsLikeSqlite3 struct {
	fakeRows
	fakeWithColumnTypeDatabaseName
	fakeWithColumnTypeLength
	fakeWithColumnTypePrecisionScale
	fakeWithColumnTypeNullable
	fakeWithColumnTypeScanType
}

type fakeConn struct {
	called          bool // nolint:structcheck // ignore unused warning, it is accessed via reflection
	rowsCloseCalled bool
	stmt            driver.Stmt
}

type fakeConnWithCheckNamedValue struct {
	fakeConn
}

type fakeConnWithoutCheckNamedValue struct {
	fakeConn
}

func (c *fakeConn) Prepare(_ string) (driver.Stmt, error) {
	return nil, nil
}

func (c *fakeConn) PrepareContext(_ context.Context, _ string) (driver.Stmt, error) {
	return c.stmt, nil
}

func (c *fakeConn) Close() error { return nil }

func (c *fakeConn) Begin() (driver.Tx, error) { return nil, nil }

func (c *fakeConn) QueryContext(_ context.Context, _ string, nvs []driver.NamedValue) (driver.Rows, error) {
	if c.stmt == nil {
		return &fakeRows{con: c}, nil
	}

	var args []driver.Value
	for _, nv := range nvs {
		args = append(args, nv.Value)
	}

	return c.stmt.Query(args)
}

func (c *fakeConnWithCheckNamedValue) CheckNamedValue(_ *driver.NamedValue) (err error) {
	c.called = true
	return
}

type fakeInterceptor struct {
	NullInterceptor
}
