// +build go1.10

package instrumentedsql

import (
	"context"
	"database/sql/driver"
	"fmt"
	"testing"
)

func TestConnectorWithDriverContext(t *testing.T) {
	err := fmt.Errorf("a generic error")

	tests := []struct {
		name             string
		openConnectorErr error
		expectErr        bool
	}{
		{
			name: "should properly open connector and wrap it",
		},
		{
			name:             "should fail when calling OpenConnector",
			openConnectorErr: err,
			expectErr:        true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			d := WrappedDriver{parent: &driverContextMock{err: test.openConnectorErr}}
			conn, err := d.OpenConnector("some-dsn")
			if err != nil {
				if test.expectErr {
					return
				}
				t.Fatalf("unexpected error from wrapped OpenConnector impl: %+v\n", err)
			}

			wc, ok := conn.(wrappedConnector)
			if !ok {
				t.Fatal("expected wrapped OpenConnector to return wrappedConnector instance")
			}

			_, ok = wc.parent.(*connMock)
			if !ok {
				t.Error("expected wrappedConnector to have connMock as parent")
			}
		})
	}
}

func TestConnectorWithDriver(t *testing.T) {
	d := WrappedDriver{parent: &driverMock{}}
	conn, err := d.OpenConnector("some-dsn")
	if err != nil {
		t.Fatalf("unexpected error from wrapped OpenConnector impl: %+v\n", err)
	}

	wc, ok := conn.(wrappedConnector)
	if !ok {
		t.Fatal("expected wrapped OpenConnector to return wrappedConnector instance")
	}

	_, ok = wc.parent.(dsnConnector)
	if !ok {
		t.Error("expected wrappedConnector to have dsnConnector as parent")
	}
}

type driverMock struct{}

func (d *driverMock) Open(name string) (driver.Conn, error) {
	panic("not implemented")
}

type driverContextMock struct {
	err error
}

func (d *driverContextMock) Open(name string) (driver.Conn, error) {
	panic("not implemented")
}

func (d *driverContextMock) OpenConnector(name string) (driver.Connector, error) {
	return &connMock{}, d.err
}

type connMock struct{}

func (c *connMock) Connect(context.Context) (driver.Conn, error) {
	panic("not implemented")
}

func (c *connMock) Driver() driver.Driver {
	panic("not implemented")
}
