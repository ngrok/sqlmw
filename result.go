package sqlmw

import (
	"database/sql/driver"
)

type wrappedResult struct {
	intr   Interceptor
	parent driver.Result
}

func (r wrappedResult) LastInsertId() (id int64, err error) {
	return r.intr.ResultLastInsertId(r.parent)
}

func (r wrappedResult) RowsAffected() (num int64, err error) {
	return r.intr.ResultRowsAffected(r.parent)
}
