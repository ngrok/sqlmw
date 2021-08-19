package sqlmw

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"testing"
)

type rowsCloseInterceptor struct {
	NullInterceptor

	rowsCloseCalled bool
}

func (r *rowsCloseInterceptor) RowsClose(rows driver.Rows) error {
	r.rowsCloseCalled = true

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

	rows, err := db.QueryContext(context.Background(), "", "")
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

	if !con.rowsCloseCalled {
		t.Fatalf("rows close of driver was not called")
	}
}
