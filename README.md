# connect

Production-ready MySQL connection helper for Go.

## What this package does

- Builds MySQL DSNs safely via `github.com/go-sql-driver/mysql`.
- Opens and returns a configured `*sql.DB`.
- Applies connection pool settings (`MaxOpenConns`, `MaxIdleConns`, `ConnMaxLifetime`, `ConnMaxIdleTime`).
- Verifies the connection with `PingContext` on startup.
- Provides a concurrency-safe named connection manager.
- Works with `database/sql` directly.
- Integrates with `github.com/akula410/builder` by supplying `*sql.DB`.

## What this package does NOT do

- Not an ORM.
- Does not build SQL queries.
- Does not scan rows into structs.
- Does not manage migrations.
- Does not replace `database/sql`.
- Does not silently reconnect in a hidden loop.

## Installation

```bash
go get github.com/akula410/connect/v2
```

```go
import "github.com/akula410/connect/v2"
```

## Basic usage

```go
package main

import (
    "context"
    "log"
    "time"

    "github.com/akula410/connect/v2"
)

func main() {
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()

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
    log.Println("MySQL version:", version)
}
```

## Usage with github.com/akula410/builder

`connect` returns a `*sql.DB`. Pass it directly to `sqlbuilder.NewExecutor`:

```go
package main

import (
    "context"
    "log"
    "time"

    sqlbuilder "github.com/akula410/builder"
    "github.com/akula410/connect/v2"
)

func main() {
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()

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

    // connect returns *sql.DB — pass it to the builder executor.
    exec := sqlbuilder.NewExecutor(db)

    rows, err := exec.QueryContext(ctx,
        sqlbuilder.Select("id", "name", "email").
            From("users").
            Where(sqlbuilder.Eq("status", "active")).
            OrderBy("id", sqlbuilder.Desc).
            Limit(10),
    )
    if err != nil {
        log.Fatal(err)
    }
    defer rows.Close()

    for rows.Next() {
        var id int64
        var name, email string
        if err := rows.Scan(&id, &name, &email); err != nil {
            log.Fatal(err)
        }
        log.Printf("user id=%d name=%s email=%s", id, name, email)
    }
}
```

The `connect` package does not import `builder`. The integration is purely in application code:
`connect` provides `*sql.DB`, `builder` consumes it via `sqlbuilder.NewExecutor(db)`.

## Named connections

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

_, _ = master, replica
```

Reopening the same name with the same normalized Config returns the existing `*sql.DB`.
Reopening with a different Config returns an error.

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

## Configuration

All fields are optional except `User` and `DBName`. `Host` and `Port` default to `127.0.0.1:3306`.

| Field | Type | Default | Description |
|---|---|---|---|
| `User` | `string` | — | **Required.** Database username. |
| `Password` | `string` | `""` | Database password. Never logged. |
| `Host` | `string` | `127.0.0.1` | MySQL server host. |
| `Port` | `string` | `3306` | MySQL server port. |
| `DBName` | `string` | — | **Required.** Database name. |
| `Charset` | `string` | `utf8mb4` | Informational only — not sent to the driver directly. The effective charset is determined by `Collation` (e.g., `utf8mb4_unicode_ci` → charset `utf8mb4`). |
| `Collation` | `string` | `utf8mb4_unicode_ci` | Sent as `SET NAMES <charset> COLLATE <collation>` on connect. Controls both the charset and sort order. |
| `ParseTime` | `bool` | `true` | Parse DATE/DATETIME as `time.Time`. **`NormalizeConfig` always defaults this to `true`**; to use `ParseTime=false`, set it explicitly after calling `NormalizeConfig`. |
| `Loc` | `string` | `Local` | IANA timezone for time values (e.g., `"UTC"`, `"Europe/Moscow"`). |
| `Timeout` | `time.Duration` | `5s` | Connection dial timeout. |
| `ReadTimeout` | `time.Duration` | `30s` | I/O read timeout. |
| `WriteTimeout` | `time.Duration` | `30s` | I/O write timeout. |
| `InterpolateParams` | `bool` | `false` | Client-side query parameter interpolation. |
| `MaxOpenConns` | `int` | `25` | Max open connections in the pool. |
| `MaxIdleConns` | `int` | `25` | Max idle connections in the pool. |
| `ConnMaxLifetime` | `time.Duration` | `5m` | Max lifetime of a pooled connection. |
| `ConnMaxIdleTime` | `time.Duration` | `5m` | Max idle time of a pooled connection. |

Call `connect.DefaultConfig()` to get all defaults as a value you can modify:

```go
cfg := connect.DefaultConfig()
cfg.User = "app_user"
cfg.DBName = "app_db"
```

## Pool settings

Connection pooling is managed by `database/sql`. `connect` applies your settings immediately after opening the DB:

```go
db.SetMaxOpenConns(cfg.MaxOpenConns)
db.SetMaxIdleConns(cfg.MaxIdleConns)
db.SetConnMaxLifetime(cfg.ConnMaxLifetime)
db.SetConnMaxIdleTime(cfg.ConnMaxIdleTime)
```

**`MaxOpenConns`** — maximum number of open connections to the database. Zero means no limit. Default 25 works for most web services.

**`MaxIdleConns`** — maximum number of idle connections retained in the pool. Keep it ≤ `MaxOpenConns`. Default 25.

**`ConnMaxLifetime`** — maximum time a connection may be reused. Set this below MySQL's `wait_timeout` (default 8 hours) to avoid "packets out of order" errors on long-lived applications. Default 5m.

**`ConnMaxIdleTime`** — maximum time a connection may sit idle before being closed. Default 5m.

## Production recommendations

- Use `context.WithTimeout` for the initial connection check.
- Set `ConnMaxLifetime` below MySQL `wait_timeout` (default 8h). 5 minutes is safe.
- Use `utf8mb4` and `utf8mb4_unicode_ci` — they support all Unicode characters including emoji.
- Never log `connect.DSN(cfg)` output or full DSN strings; they contain the password.
- Close the DB gracefully on shutdown: `defer db.Close()`.
- Tune `MaxOpenConns` based on your MySQL `max_connections` and number of application instances.
- Do not share a single `*sql.DB` across services or processes — create one per application.

## Migrating from v1

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

Key changes:

| v1 | v2 |
|---|---|
| `MySql` struct | `Config` struct |
| `Connect()` panics | `NewMySQLContext` returns `error` |
| `MaxOpenCoons` (typo) | `MaxOpenConns` |
| charset default `utf8` | charset default `utf8mb4` |
| global `map[string]*sql.DB` | concurrency-safe `Manager` |
| no context support | full `context.Context` support |

## Security notes

- Passwords are never included in error messages or log output by this package.
- DSNs returned by `connect.DSN()` contain credentials — do not log them.
- Use least-privilege MySQL users: grant only the permissions the application needs.
- Prefer `InterpolateParams: false` (default) to avoid client-side escaping bypasses.

## License

MIT
