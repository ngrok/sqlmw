package sqlmw

import (
	"database/sql/driver"
)

type wrappedTx struct {
	intr   Interceptor
	parent driver.Tx
}

// Compile time validation that our types implement the expected interfaces
var (
	_ driver.Tx = wrappedTx{}
)

func (t wrappedTx) Commit() (err error) {
	return t.intr.TxCommit(t.parent)
}

func (t wrappedTx) Rollback() (err error) {
	return t.intr.TxRollback(t.parent)
}
