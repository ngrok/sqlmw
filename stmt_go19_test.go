package sqlmw

import (
	"database/sql"
	"database/sql/driver"
	"reflect"
	"testing"
)

// TestDefaultParameterConversion ensures that
// driver.DefaultParameterConverter is used when neither stmt nor con
// implements any value converters.
func TestDefaultParameterConversion(t *testing.T) {
	driverName := driverName(t)

	expectVal := int64(1)
	con := &fakeConn{}
	fakeStmt := &fakeStmt{
		rows: &fakeRows{
			con:  con,
			vals: [][]driver.Value{{expectVal}},
		},
	}
	con.stmt = fakeStmt

	sql.Register(
		driverName,
		Driver(&fakeDriver{conn: con}, &NullInterceptor{}),
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

	stmt, err := db.Prepare("")
	if err != nil {
		t.Fatalf("Prepare failed: %s", err)
	}

	// int32 values are converted by driver.DefaultParameterConverter to
	// int64
	queryVal := int32(1)
	rows, err := stmt.Query(queryVal)
	if err != nil {
		t.Fatalf("Query failed: %s", err)
	}

	count := 0
	for rows.Next() {
		var v int64
		err := rows.Scan(&v)
		if err != nil {
			t.Fatalf("rows.Scan failed, %v", err)
		}
		if v != 1 {
			t.Errorf("converted value is %d, passed value to Query was: %d", v, expectVal)
		}
		count++
	}

	if count != 1 {
		t.Fatalf("got too many rows, expected 1, got %d ", 1)
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
			driverName := driverName(t)
			sql.Register(driverName, Driver(test.fd, &fakeInterceptor{}))
			db, err := sql.Open(driverName, "dummy")
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
