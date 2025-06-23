//go:build go1.10
// +build go1.10

package sqlmw

import "database/sql/driver"

var _ driver.DriverContext = wrappedDriver{}

func (d wrappedDriver) OpenConnector(name string) (driver.Connector, error) {
	driver, ok := d.parent.(driver.DriverContext)
	if !ok {
		return WrapConnector(&d, dsnConnector{dsn: name, driver: d.parent}), nil
	}
	conn, err := driver.OpenConnector(name)
	if err != nil {
		return nil, err
	}
	return WrapConnector(&d, conn), nil
}
