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

type fakeTx struct{}

func (f fakeTx) Commit() error { return nil }

func (f fakeTx) Rollback() error { return nil }

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

func (s fakeStmt) QueryContext(_ context.Context, _ []driver.NamedValue) (driver.Rows, error) {
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

	//These are here so that we can check things have not been called
	hasNextResultSetCalled         bool // nolint:structcheck,unused // ignore unused warning, it is accessed via reflection
	nextResultSetCalled            bool // nolint:structcheck,unused // ignore unused warning, it is accessed via reflection
	columnTypeDatabaseNameCalled   bool // nolint:structcheck,unused // ignore unused warning, it is accessed via reflection
	columnTypeLengthCalled         bool // nolint:structcheck,unused // ignore unused warning, it is accessed via reflection
	columnTypePrecisionScaleCalled bool // nolint:structcheck,unused // ignore unused warning, it is accessed via reflection
	columnTypeNullable             bool // nolint:structcheck,unused // ignore unused warning, it is accessed via reflection
	columnTypeScanTypeCalled       bool // nolint:structcheck,unused // ignore unused warning, it is accessed via reflection
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
	r *fakeRows
}

func (f *fakeWithRowsNextResultSet) HasNextResultSet() bool {
	f.r.hasNextResultSetCalled = true
	return false
}

func (f *fakeWithRowsNextResultSet) NextResultSet() error {
	f.r.nextResultSetCalled = true
	return nil
}

type fakeWithColumnTypeDatabaseName struct {
	r     *fakeRows
	names []string
}

func (f *fakeWithColumnTypeDatabaseName) ColumnTypeDatabaseTypeName(index int) string {
	f.r.columnTypeDatabaseNameCalled = true
	return f.names[index]
}

type fakeWithColumnTypeLength struct {
	r       *fakeRows
	lengths []int64
	bools   []bool
}

func (f *fakeWithColumnTypeLength) ColumnTypeLength(index int) (length int64, ok bool) {
	f.r.columnTypeLengthCalled = true
	return f.lengths[index], f.bools[index]
}

type fakeWithColumnTypePrecisionScale struct {
	r                  *fakeRows
	precisions, scales []int64
	bools              []bool
}

func (f *fakeWithColumnTypePrecisionScale) ColumnTypePrecisionScale(index int) (precision, scale int64, ok bool) {
	f.r.columnTypePrecisionScaleCalled = true
	return f.precisions[index], f.scales[index], f.bools[index]
}

type fakeWithColumnTypeNullable struct {
	r         *fakeRows
	nullables []bool
	oks       []bool
}

func (f *fakeWithColumnTypeNullable) ColumnTypeNullable(index int) (nullable, ok bool) {
	f.r.columnTypeNullable = true
	return f.nullables[index], f.oks[index]
}

type fakeWithColumnTypeScanType struct {
	r         *fakeRows
	scanTypes []reflect.Type
}

func (f *fakeWithColumnTypeScanType) ColumnTypeScanType(index int) reflect.Type {
	f.r.columnTypeScanTypeCalled = true
	return f.scanTypes[index]
}

type fakeRowsLikeMysql struct {
	fakeRows
	fakeWithRowsNextResultSet
	fakeWithColumnTypeDatabaseName
	fakeWithColumnTypePrecisionScale
	fakeWithColumnTypeNullable
	fakeWithColumnTypeScanType
}

// The set of interfaces support by pgx and sqlite3
type fakeRowsLikePgx struct {
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
	tx              driver.Tx
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

func (c *fakeConn) ExecContext(_ context.Context, _ string, _ []driver.NamedValue) (driver.Result, error) {
	return nil, nil
}

func (c *fakeConn) Close() error { return nil }

func (c *fakeConn) Begin() (driver.Tx, error) { return c.tx, nil }

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
