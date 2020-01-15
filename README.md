[![GoDoc](https://godoc.org/github.com/inconshreveable/sqlmw?status.svg)](https://godoc.org/github.com/inconshreveable/sqlmw)

# sqlmw
sqlmw provides an absurdly simple API that allows a caller to "wrap" another database/sql driver
with middleware.

Think grpc interceptors or http middleware but for the database/sql package. This allows a caller to implement
observability like tracing and logging easily. More importantly, it also enables far more powerful behaviors like
transparently modifying arguments, results or query execution strategy. This power allows programmers to implement
behaviors like automatic sharding, selective tracing, automatic caching, transparent query mirroring, retries, failover, and more.

## Usage

It's absurdly simple:

- Define a new type and embed the `sqlmw.NullInterceptor` type.
- Add a method you want to intercept from the `sqlmw.Interceptor` interface.
- Wrap the driver with your interceptor with `sqlmw.Driver` and then install it with `sql.Register`.

Here's a complete example:

```
func run(dsn string) {
        // install the wrapped driver
        sql.Register("postgres-mw", sqlmw.Driver(pq.Dirver{}, new(sqlInterceptor)))
        db, err := sql.Open("postgres-mw", dsn)
        ...
}

type sqlInterceptor struct {
        sqlmw.NullInterceptor
}

func (in *sqlInterceptor) StmtQueryContext(ctx context.Context, conn driver.StmtQueryContext, query string, args []driver.NamedValue) (driver.Rows, error) {
        startedAt := time.Now()
        rows, err := conn.QueryContext(ctx, args)
        log.Debug("executed sql query", "duration", time.Since(startedAt), "query", query, "args", args, "err", err)
        return rows, err
}
```

You may override any subset of methods to intercept in the `Interceptor` interface (https://godoc.org/github.com/inconshreveable/sqlmw#Interceptor):

```
type Interceptor interface {
    // Connection interceptors
    ConnBeginTx(context.Context, driver.ConnBeginTx, driver.TxOptions) (driver.Tx, error)
    ConnPrepareContext(context.Context, driver.ConnPrepareContext, string) (driver.Stmt, error)
    ConnPing(context.Context, driver.Pinger) error
    ConnExecContext(context.Context, driver.ExecerContext, string, []driver.NamedValue) (driver.Result, error)
    ConnQueryContext(context.Context, driver.QueryerContext, string, []driver.NamedValue) (driver.Rows, error)

    // Connector interceptors
    ConnectorConnect(context.Context, driver.Connector) (driver.Conn, error)

    // Results interceptors
    ResultLastInsertId(driver.Result) (int64, error)
    ResultRowsAffected(driver.Result) (int64, error)

    // Rows interceptors
    RowsNext(context.Context, driver.Rows, []driver.Value) error

    // Stmt interceptors
    StmtExecContext(context.Context, driver.StmtExecContext, string, []driver.NamedValue) (driver.Result, error)
    StmtQueryContext(context.Context, driver.StmtQueryContext, string, []driver.NamedValue) (driver.Rows, error)
    StmtClose(context.Context, driver.Stmt) error

    // Tx interceptors
    TxCommit(context.Context, driver.Tx) error
    TxRollback(context.Context, driver.Tx) error
}
```

Bear in mind that becase you are intercepting the calls entirely, that you are responsible for passing control up to the wrapped
driver in any function that you override, like so:

```
func (in *sqlInterceptor) ConnPing(ctx context.Context, conn driver.Pinger) error {
        return conn.Ping(ctx)
}
```

## Comaprison with similar projects

There are a number of other packages that allow the programmer to wrap a `database/sql/driver.Driver` to add logging or tracing.

Examples of tracing packages:
  - github.com/ExpansiveWorlds/instrumentedsql
  - contrib.go.opencensus.io/integrations/ocsql
  - gopkg.in/DataDog/dd-trace-go.v1/contrib/database/sql

A few provide a much more flexible setup of arbitrary before/after hooks to facilitate custom observability.

Pacakges that provide before/after hooks:
  - github.com/gchaincl/sqlhooks
  - github.com/shogo82148/go-sql-proxy

None of these packages provide an interface with sufficient power. `sqlmw` passes control of executing the
sql query to the caller which allows the caller to completely disintermediate the sql calls. This is what provides
the power to implement advanced behaviors like caching, sharding, retries, etc.

## Go version support

Go versions 1.9 and forward are supported.

## Fork

This project began by forking the code in github.com/luna-duclos/instrumentedsql, which itself is a fork of github.com/ExpansiveWorlds/instrumentedsql
