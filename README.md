# connect

Production-ready MySQL connection helper for Go.

## What this package does

- Builds MySQL DSNs safely via `github.com/go-sql-driver/mysql`.
- Opens and returns a configured `*sql.DB`.
- Applies connection pool settings (`MaxOpenConns`, `MaxIdleConns`, `ConnMaxLifetime`, `ConnMaxIdleTime`).
- Verifies the connection with `PingContext` on startup.
- Provides a concurrency-safe named connection manager (`Manager`).
- Supports startup retry with exponential backoff (`NewMySQLContextWithRetry`).
- Exposes pool statistics via `sql.DBStats`.
- Works with `database/sql` directly.
- Integrates with `github.com/akula410/builder` by supplying `*sql.DB`.

## What this package does NOT do

- Not an ORM.
- Does not build SQL queries.
- Does not scan rows into structs.
- Does not manage migrations.
- Does not implement its own connection pool (uses `database/sql`).
- Does not silently reconnect in a hidden background loop.
- Does not perform automatic failover.
- Does not replace MySQL monitoring.
- Does not auto-tune `MaxOpenConns`.

## Installation

```bash
go get github.com/akula410/connect/v2
```

```go
import "github.com/akula410/connect/v2"
```

## Basic usage

```go
db, err := connect.NewMySQLContext(ctx, connect.Config{
    User:     "app_user",
    Password: "app_password",
    Host:     "127.0.0.1",
    Port:     "3306",
    DBName:   "app_db",
})
if err != nil {
    log.Fatal(err)
}
defer db.Close()

// db is a *sql.DB — use it with database/sql normally.
var version string
if err := db.QueryRowContext(ctx, "SELECT VERSION()").Scan(&version); err != nil {
    log.Fatal(err)
}
```

Or use `DefaultConfig()` to start from sensible defaults:

```go
cfg := connect.DefaultConfig()
cfg.User = "app_user"
cfg.Password = "app_password"
cfg.DBName = "app_db"

db, err := connect.NewMySQLContext(ctx, cfg)
```

## Context-aware connection

Always use `NewMySQLContext` with a timeout for the startup ping:

```go
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()

db, err := connect.NewMySQLContext(ctx, cfg)
```

`NewMySQL` is a convenience wrapper that uses `context.Background()`.

## Pool configuration

`database/sql` manages the connection pool. `connect` applies your settings immediately after opening:

```go
cfg := connect.DefaultConfig()
cfg.User = "app_user"
cfg.DBName = "app_db"

cfg.MaxOpenConns    = 40
cfg.MaxIdleConns    = 20
cfg.ConnMaxLifetime = 5 * time.Minute
cfg.ConnMaxIdleTime = 3 * time.Minute
```

| Setting | Default | Description |
|---|---|---|
| `MaxOpenConns` | 25 | Maximum open connections. Zero means no limit (dangerous in production). |
| `MaxIdleConns` | 25 | Idle connections kept in the pool. Keep ≤ `MaxOpenConns`. |
| `ConnMaxLifetime` | 5m | Max age of a connection. Set below MySQL `wait_timeout` to avoid stale connections. |
| `ConnMaxIdleTime` | 5m | Max time a connection can sit idle before being closed. |

**Important:** create `*sql.DB` once at application startup and share it. Do not create a new connection per HTTP request.

## Connection pool sizing for high-load applications

`DefaultConfig` is safe as a starting point but is not a universal production setting. Tune it for your MySQL instance, number of application instances, and query patterns.

**Formula:**

```
MySQL max_connections      = 300
Reserved (admin/system)    = 30
Application instances      = 6

Available per instance:
(300 - 30) / 6 = 45

Recommended starting point:
MaxOpenConns = 40          (leave 5 headroom)
MaxIdleConns = 10–25       (adjust based on traffic pattern)
```

**Rules of thumb:**

- Set `MaxOpenConns` to the calculated available connections per instance.
- Set `MaxIdleConns` equal to `MaxOpenConns` for steady high-throughput workloads. For bursty traffic, a lower value (10–15) reduces idle resource usage.
- Keep `ConnMaxLifetime` well below MySQL `wait_timeout` (default 8 hours). 5 minutes is safe.
- Short-lived queries (< 1ms) → larger pool. Long-running queries (> 100ms) → smaller pool, or you will exhaust it.
- Monitor `sql.DBStats.WaitCount` and `WaitDuration`; high values indicate the pool is too small.

## Timeouts

```go
cfg.Timeout      = 3 * time.Second   // dial timeout
cfg.ReadTimeout  = 15 * time.Second  // I/O read timeout
cfg.WriteTimeout = 15 * time.Second  // I/O write timeout
```

Tight timeouts prevent cascading failures under load. Adjust based on your slowest expected query time.

## TLS

Pass a TLS configuration name to enable encrypted connections:

```go
cfg.TLSConfig = "true"         // require TLS, verify server certificate
cfg.TLSConfig = "skip-verify"  // require TLS, skip certificate verification
cfg.TLSConfig = "false"        // disable TLS
cfg.TLSConfig = "custom"       // a name registered via mysql.RegisterTLSConfig
```

Empty `TLSConfig` (the default) uses the driver default, which is no TLS for local connections.

```go
cfg := connect.DefaultConfig()
cfg.User = "app_user"
cfg.DBName = "app_db"
cfg.TLSConfig = "skip-verify"

db, err := connect.NewMySQLContext(ctx, cfg)
```

## Disabling ParseTime

`NormalizeConfig` defaults `ParseTime` to `true`, which enables automatic parsing of `DATE`/`DATETIME` columns into `time.Time`. To disable it:

```go
cfg := connect.DefaultConfig()
cfg.User = "app_user"
cfg.DBName = "app_db"
cfg.DisableParseTime = true   // results in parseTime=false in the DSN

db, err := connect.NewMySQLContext(ctx, cfg)
```

Setting `ParseTime: false` directly on Config is not sufficient because `NormalizeConfig` cannot distinguish `false` (zero value) from "not set".

## Retry on startup

Use `NewMySQLContextWithRetry` when the application may start before MySQL is ready:

```go
retry := connect.RetryConfig{
    Attempts: 5,
    MinDelay: 500 * time.Millisecond,
    MaxDelay: 5 * time.Second,
}

db, err := connect.NewMySQLContextWithRetry(ctx, cfg, retry)
if err != nil {
    log.Fatalf("could not connect after %d attempts: %v", retry.Attempts, err)
}
defer db.Close()
```

Retry uses exponential backoff: the delay starts at `MinDelay`, doubles after each failure, and is capped at `MaxDelay`. `Attempts <= 1` performs a single attempt with no retry. Retry stops immediately if the context is cancelled.

**This is a startup-only mechanism.** Do not use it for runtime reconnection.

## Manager for multiple connections

Use `Manager` when an application needs multiple MySQL connections (e.g., master + read replica):

```go
manager := connect.NewManager()
defer manager.CloseAll()

master, err := manager.Open(ctx, "master", connect.Config{
    User:   "app_user",
    DBName: "app_db",
})
if err != nil {
    log.Fatal(err)
}

replica, err := manager.Open(ctx, "replica", connect.Config{
    User:   "app_user",
    Host:   "replica.internal",
    DBName: "app_db",
})
if err != nil {
    log.Fatal(err)
}

// Retrieve a connection by name anywhere in your application.
db, ok := manager.Get("master")

// List all registered names (sorted).
names := manager.Names()
```

Reopening the same name with the same normalized Config returns the existing `*sql.DB`.
Reopening with a different Config returns an error.

Concurrent `Open` calls for the same name are safe: the mutex is released before the actual
connection and ping so other goroutines are not blocked. If two goroutines race to open the
same name, the slower one closes its extra connection and returns the winner's connection.

## Stats / observability

`*sql.DB` exposes pool statistics directly via `.Stats()`:

```go
stats := db.Stats()

fmt.Println(stats.OpenConnections)  // total open connections
fmt.Println(stats.InUse)            // connections currently executing a query
fmt.Println(stats.Idle)             // connections waiting in the pool
fmt.Println(stats.WaitCount)        // total goroutines that waited for a connection
fmt.Println(stats.WaitDuration)     // total time spent waiting
fmt.Println(stats.MaxIdleClosed)    // connections closed due to MaxIdleConns
fmt.Println(stats.MaxLifetimeClosed) // connections closed due to ConnMaxLifetime
```

For `Manager`, use `Stats(name)` or `StatsAll()`:

```go
// Single named connection.
if stats, ok := manager.Stats("master"); ok {
    fmt.Println("master open:", stats.OpenConnections)
}

// All connections at once.
for name, stats := range manager.StatsAll() {
    fmt.Printf("%s: open=%d inuse=%d idle=%d waitCount=%d\n",
        name, stats.OpenConnections, stats.InUse, stats.Idle, stats.WaitCount)
}
```

No external monitoring libraries are required. To integrate with Prometheus, read `db.Stats()` periodically in a goroutine and record the values as gauges.

## Graceful shutdown

```go
// Single connection.
db, err := connect.NewMySQLContext(ctx, cfg)
if err != nil {
    log.Fatal(err)
}
defer db.Close()   // release pool resources on exit

// Manager.
manager := connect.NewManager()
defer manager.CloseAll()
```

`db.Close()` waits for in-flight queries to complete and then closes all idle connections. Do not call `db.Close()` after every query — close only on application shutdown.

## Health checks

```go
func HealthHandler(db *sql.DB) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
        defer cancel()

        if err := connect.PingContext(ctx, db); err != nil {
            http.Error(w, "database unavailable", http.StatusServiceUnavailable)
            return
        }
        w.WriteHeader(http.StatusOK)
        _, _ = w.Write([]byte("ok\n"))
    }
}
```

## Usage with github.com/akula410/builder

`connect` returns a plain `*sql.DB`. Pass it directly to `sqlbuilder.NewExecutor`:

```go
db, err := connect.NewMySQLContext(ctx, cfg)
if err != nil {
    log.Fatal(err)
}
defer db.Close()

exec := sqlbuilder.NewExecutor(db)

rows, err := exec.QueryContext(ctx,
    sqlbuilder.Select("id", "name", "email").
        From("users").
        Where(sqlbuilder.Eq("status", "active")).
        OrderBy("id", sqlbuilder.Desc).
        Limit(10),
)
```

`connect` does not import `builder`. The integration lives entirely in application code:
`connect` provides `*sql.DB`, `builder` consumes it.

## Configuration reference

All fields are optional except `User` and `DBName`.

| Field | Type | Default | Description |
|---|---|---|---|
| `User` | `string` | — | **Required.** Database username. |
| `Password` | `string` | `""` | Database password. Never logged. |
| `Host` | `string` | `127.0.0.1` | MySQL server host. |
| `Port` | `string` | `3306` | MySQL server port. |
| `DBName` | `string` | — | **Required.** Database name. |
| `Charset` | `string` | `utf8mb4` | Informational only — not sent to the driver directly. The effective charset is determined by `Collation`. |
| `Collation` | `string` | `utf8mb4_unicode_ci` | Sent as `SET NAMES … COLLATE …`. Controls charset and sort order. |
| `ParseTime` | `bool` | `true` | Parse DATE/DATETIME as `time.Time`. Defaulted to `true` by `NormalizeConfig`. |
| `DisableParseTime` | `bool` | `false` | Explicitly set `parseTime=false`. Use instead of `ParseTime: false` which is indistinguishable from zero value. |
| `Loc` | `string` | `Local` | IANA timezone for time values (e.g., `"UTC"`, `"Europe/Moscow"`). |
| `Timeout` | `time.Duration` | `5s` | Connection dial timeout. |
| `ReadTimeout` | `time.Duration` | `30s` | I/O read timeout. |
| `WriteTimeout` | `time.Duration` | `30s` | I/O write timeout. |
| `InterpolateParams` | `bool` | `false` | Client-side query parameter interpolation. |
| `TLSConfig` | `string` | `""` | TLS configuration name: `"true"`, `"false"`, `"skip-verify"`, or a registered name. |
| `MaxOpenConns` | `int` | `25` | Max open connections. |
| `MaxIdleConns` | `int` | `25` | Max idle connections. |
| `ConnMaxLifetime` | `time.Duration` | `5m` | Max connection lifetime. |
| `ConnMaxIdleTime` | `time.Duration` | `5m` | Max connection idle time. |

## Examples

| Example | Description |
|---|---|
| `examples/basic/` | Simple single connection |
| `examples/manager/` | Multiple named connections with Manager |
| `examples/highload_config/` | Pool sizing for high-load workloads |
| `examples/stats/` | Observability via sql.DBStats |
| `examples/retry/` | Startup retry with exponential backoff |
| `examples/healthcheck/` | HTTP health check endpoint |
| `examples/with_builder/` | Integration with github.com/akula410/builder |

All examples read credentials from environment variables:

```bash
export MYSQL_USER=app_user
export MYSQL_PASSWORD=secret
export MYSQL_HOST=127.0.0.1
export MYSQL_PORT=3306
export MYSQL_DATABASE=app_db

go run examples/basic/main.go
```

## Migration from v1

v1 used a `MySql` struct with panic-based error handling and a global connection map.

```go
// v1 (panics, no context, utf8 default, global state)
ms := &connect.MySql{User: "u", DBName: "db"}
db := ms.Connect()
```

v2 replaces this with:

```go
// v2 (returns errors, context-aware, utf8mb4 default, no global state)
db, err := connect.NewMySQLContext(ctx, connect.Config{
    User:   "u",
    DBName: "db",
})
```

| v1 | v2 |
|---|---|
| `MySql` struct | `Config` struct |
| `Connect()` panics | `NewMySQLContext` returns `error` |
| `MaxOpenCoons` (typo) | `MaxOpenConns` |
| charset default `utf8` | charset default `utf8mb4` |
| global `map[string]*sql.DB` | concurrency-safe `Manager` |
| no context support | full `context.Context` support |
| no retry | `NewMySQLContextWithRetry` |
| no stats API | `Manager.Stats` / `Manager.StatsAll` |

## Security notes

- Passwords are never included in error messages or log output by this package.
- DSNs returned by `connect.DSN()` contain credentials — do not log them.
- Use least-privilege MySQL users: grant only the permissions the application needs.
- Prefer `InterpolateParams: false` (default) to avoid client-side escaping bypasses.

## License

MIT
