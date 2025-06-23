//go:build go1.10
// +build go1.10

package sqlmw

import (
	"context"
	"database/sql/driver"
)

type wrappedConnector struct {
	parent    driver.Connector
	driverRef *wrappedDriver
}

var (
	_ driver.Connector = wrappedConnector{}
)

// WrapConnector returns the supplied driver.Connector wrapped in a new object that has all of its calls intercepted by
// the Interceptor in the supplied driver.
func WrapConnector(driverRef *wrappedDriver, connector driver.Connector) wrappedConnector {
	return wrappedConnector{driverRef: driverRef, parent: connector}
}

func (c wrappedConnector) Connect(ctx context.Context) (conn driver.Conn, err error) {
	conn, err = c.driverRef.intr.ConnectorConnect(ctx, c.parent)
	if err != nil {
		return nil, err
	}

	return wrappedConn{intr: c.driverRef.intr, parent: conn}, nil
}

func (c wrappedConnector) Driver() driver.Driver {
	return c.driverRef
}

// dsnConnector is a fallback connector placed in position of wrappedConnector.parent
// when given Driver does not comply with DriverContext interface.
type dsnConnector struct {
	dsn    string
	driver driver.Driver
}

func (t dsnConnector) Connect(_ context.Context) (driver.Conn, error) {
	return t.driver.Open(t.dsn)
}

func (t dsnConnector) Driver() driver.Driver {
	return t.driver
}
