package sqlmw

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"reflect"
	"testing"
)

// TestDefaultParameterConversion ensures that
// driver.DefaultParameterConverter is used when neither stmt nor con
// implements any value converters.
func TestDefaultParameterConversion(t *testing.T) {
	driverNameWithSQLmw := t.Name() + "sqlmw"
	fakeStmt := fakeStmtWithValStore{}
	sql.Register(
		driverNameWithSQLmw,
		Driver(&fakeDriver{conn: &fakeConn{stmt: &fakeStmt}}, &NullInterceptor{}),
	)

	db, err := sql.Open(driverNameWithSQLmw, "")
	if err != nil {
		t.Fatalf("Failed to open: %v", err)
	}

	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Errorf("Failed to close db: %v", err)
		}
	})

	stmt, err := db.Prepare("")
	if err != nil {
		t.Fatalf("Prepare failed: %s", err)
	}

	// int32 values are converted by driver.DefaultParameterConverter to
	// int64
	queryVal := int32(1)
	_, err = stmt.Query(queryVal)
	if err != nil {
		t.Fatalf("Query failed: %s", err)
	}

	if len(fakeStmt.val) != 1 {
		t.Fatalf("fakestmt got %d values, expected %d", len(fakeStmt.val), 1)
	}

	switch v := fakeStmt.val[0].(type) {
	case int32:
		t.Errorf("int32 was not converted to int64 **without** using sqlmw")
	case int64:
		if v != int64(1) {
			t.Errorf("converted value is %d, passed value to Query was: %d", v, queryVal)
		}
	default:
		t.Errorf("converted value has type %T, expected int64", fakeStmt.val[0])
	}
}

func TestWrappedStmt_CheckNamedValue(t *testing.T) {
	tests := map[string]struct {
		fd       *fakeDriver
		expected struct {
			cc bool // Whether the fakeConn's CheckNamedValue was called
			sc bool // Whether the fakeStmt's CheckNamedValue was called
		}
	}{
		"When both conn and stmt implement CheckNamedValue": {
			fd: &fakeDriver{
				conn: &fakeConnWithCheckNamedValue{
					fakeConn: fakeConn{
						stmt: &fakeStmtWithCheckNamedValue{},
					},
				},
			},
			expected: struct {
				cc bool
				sc bool
			}{cc: false, sc: true},
		},
		"When only conn implements CheckNamedValue": {
			fd: &fakeDriver{
				conn: &fakeConnWithCheckNamedValue{
					fakeConn: fakeConn{
						stmt: &fakeStmtWithoutCheckNamedValue{},
					},
				},
			},
			expected: struct {
				cc bool
				sc bool
			}{cc: true, sc: false},
		},
		"When only stmt implements CheckNamedValue": {
			fd: &fakeDriver{
				conn: &fakeConnWithoutCheckNamedValue{
					fakeConn: fakeConn{
						stmt: &fakeStmtWithCheckNamedValue{},
					},
				},
			},
			expected: struct {
				cc bool
				sc bool
			}{cc: false, sc: true},
		},
		"When both stmt do not implement CheckNamedValue": {
			fd: &fakeDriver{
				conn: &fakeConnWithoutCheckNamedValue{
					fakeConn: fakeConn{
						stmt: &fakeStmtWithoutCheckNamedValue{},
					},
				},
			},
			expected: struct {
				cc bool
				sc bool
			}{cc: false, sc: false},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			sql.Register("fake-driver:"+name, Driver(test.fd, &fakeInterceptor{}))
			db, err := sql.Open("fake-driver:"+name, "dummy")
			if err != nil {
				t.Errorf("Failed to open: %v", err)
			}
			defer func() {
				if err := db.Close(); err != nil {
					t.Errorf("Failed to close db: %v", err)
				}
			}()

			stmt, err := db.Prepare("SELECT foo FROM bar Where 1 = ?")
			if err != nil {
				t.Errorf("Failed to prepare: %v", err)
			}

			if _, err := stmt.Query(1); err != nil {
				t.Errorf("Failed to query: %v", err)
			}

			conn := reflect.ValueOf(test.fd.conn).Elem()
			sc := conn.FieldByName("stmt").Elem().Elem().FieldByName("called").Bool()
			cc := conn.FieldByName("called").Bool()

			if test.expected.sc != sc {
				t.Errorf("sc mismatch.\n got: %#v\nwant: %#v", sc, test.expected.sc)
			}

			if test.expected.cc != cc {
				t.Errorf("cc mismatch.\n got: %#v\nwant: %#v", cc, test.expected.cc)
			}
		})
	}
}

type userWrappedStmt struct {
	driver.Stmt
	userData bool
}

type wrapStmtInterceptor struct {
	NullInterceptor
}

func (wrapStmtInterceptor) ConnPrepareContext(ctx context.Context, conn driver.ConnPrepareContext, query string) (driver.Stmt, error) {
	stmt, err := conn.PrepareContext(ctx, query)
	if err != nil {
		return nil, err
	}

	return &userWrappedStmt{Stmt: stmt, userData: true}, nil
}

func (wrapStmtInterceptor) StmtExecContext(ctx context.Context, stmt *Stmt, _ string, args []driver.NamedValue) (driver.Result, error) {
	ud, ok := stmt.Parent().(*userWrappedStmt)
	if !ok {
		return nil, fmt.Errorf("stmt.Parent() has type %t, expected *userWrappedStmt", stmt.Parent())
	}

	if ud == nil {
		return nil, errors.New("userData is nil")
	}

	if !ud.userData {
		return nil, errors.New("userData is false, expected true")
	}

	return stmt.ExecContext(ctx, args)
}

func (wrapStmtInterceptor) StmtQueryContext(ctx context.Context, stmt *Stmt, _ string, args []driver.NamedValue) (driver.Rows, error) {
	ud, ok := stmt.Parent().(*userWrappedStmt)
	if !ok {
		return nil, fmt.Errorf("stmt.Parent() has type %t, expected *userWrappedStmt", stmt.Parent())
	}

	if ud == nil {
		return nil, errors.New("userData is nil")
	}

	if !ud.userData {
		return nil, errors.New("userData is false, expected true")
	}

	return stmt.QueryContext(ctx, args)
}

func (wrapStmtInterceptor) StmtClose(ctx context.Context, stmt *Stmt) error {
	ud, ok := stmt.Parent().(*userWrappedStmt)
	if !ok {
		return fmt.Errorf("stmt.Parent() has type %t, expected *userWrappedStmt", stmt.Parent())
	}

	if ud == nil {
		return errors.New("userData is nil")
	}

	if !ud.userData {
		return errors.New("userData is false, expected true")
	}

	return stmt.Close()
}

func TestWrapStmt(t *testing.T) {
	driverNameWithSQLmw := t.Name() + "sqlmw"

	fakeStmt := fakeStmt{}
	sql.Register(
		driverNameWithSQLmw,
		Driver(&fakeDriver{conn: &fakeConn{stmt: &fakeStmt}}, &wrapStmtInterceptor{}),
	)

	db, err := sql.Open(driverNameWithSQLmw, "")
	if err != nil {
		t.Fatalf("Failed to open: %v", err)
	}

	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Errorf("Failed to close db: %v", err)
		}
	})

	stmt, err := db.Prepare("")
	if err != nil {
		t.Fatalf("Prepare failed: %s", err)
	}

	if _, err := stmt.ExecContext(context.Background(), ""); err != nil {
		t.Errorf("stmt.ExecContext failed: %s", err)
	}

	if _, err := stmt.Exec(""); err != nil {
		t.Errorf("stmt.Exec failed: %s", err)
	}

	if _, err := stmt.QueryContext(context.Background(), ""); err != nil {
		t.Errorf("stmt.QueryContext failed: %s", err)
	}

	if _, err := stmt.Query(""); err != nil {
		t.Errorf("stmt.Query failed: %s", err)
	}
}
