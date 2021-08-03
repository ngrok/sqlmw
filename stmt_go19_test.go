package sqlmw

import (
	"database/sql"
	"reflect"
	"testing"
)

func TestWrappedStmt_CheckNamedValue(t *testing.T) {
	tests := map[string]struct {
		fd       *fakeDriver
		expected struct {
			cc  bool // Whether the fakeConn's CheckNamedValue was called
			sc  bool // Whether the fakeStmt's CheckNamedValue was called
			cci bool // Whether the fakeStmt's ColumnConverter was called

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
				cc  bool
				sc  bool
				cci bool
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
				cc  bool
				sc  bool
				cci bool
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
				cc  bool
				sc  bool
				cci bool
			}{cc: false, sc: true},
		},
		"When only stmt implements ColumnConverter": {
			fd: &fakeDriver{
				conn: &fakeConnWithoutCheckNamedValue{
					fakeConn: fakeConn{
						stmt: &fakeStmtWithColumnConverter{},
					},
				},
			},
			expected: struct {
				cc  bool
				sc  bool
				cci bool
			}{cci: true},
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
				cc  bool
				sc  bool
				cci bool
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
			sc := conn.FieldByName("stmt").Elem().Elem().FieldByName("checkNamedValueCalled").Bool()
			cc := conn.FieldByName("called").Bool()
			cci := conn.FieldByName("stmt").Elem().Elem().FieldByName("columnConverterCalled").Bool()

			if test.expected.sc != sc {
				t.Errorf("sc mismatch.\n got: %#v\nwant: %#v", sc, test.expected.sc)
			}

			if test.expected.cc != cc {
				t.Errorf("cc mismatch.\n got: %#v\nwant: %#v", cc, test.expected.cc)
			}

			if test.expected.cci != cci {
				t.Errorf("columnConverterCalled mismatch.\n got: %#v\nwant: %#v", cci, test.expected.cci)
			}
		})
	}
}
