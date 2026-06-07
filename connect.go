package connect

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/go-sql-driver/mysql"
)

// NewMySQLContext opens and configures a MySQL *sql.DB using the provided context and Config.
// It normalizes the Config, validates required fields, applies pool settings, and
// verifies the connection with PingContext. If ping fails, the DB is closed before returning the error.
func NewMySQLContext(ctx context.Context, cfg Config) (*sql.DB, error) {
	cfg = NormalizeConfig(cfg)
	if err := ValidateConfig(cfg); err != nil {
		return nil, err
	}

	mc, err := buildMySQLConfig(cfg)
	if err != nil {
		return nil, err
	}

	connector, err := mysql.NewConnector(mc)
	if err != nil {
		return nil, fmt.Errorf("connect: create connector: %w", err)
	}

	db := sql.OpenDB(connector)
	applyPool(db, cfg)

	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("connect: ping: %w", err)
	}

	return db, nil
}

// NewMySQL opens and configures a MySQL *sql.DB using context.Background().
// For explicit context control use NewMySQLContext.
func NewMySQL(cfg Config) (*sql.DB, error) {
	return NewMySQLContext(context.Background(), cfg)
}

// RetryConfig controls the retry behaviour for NewMySQLContextWithRetry.
// Retry is intended for startup only — use it once to establish the initial connection,
// not as a runtime reconnection mechanism.
//
// Attempts <= 1 performs a single attempt with no retry.
// MinDelay is the initial wait between attempts; it doubles after each failure (exponential backoff).
// MaxDelay caps the per-attempt wait. Both default to zero (no delay) when unset.
type RetryConfig struct {
	// Attempts is the maximum number of connection attempts. Values <= 1 mean a single attempt.
	Attempts int
	// MinDelay is the minimum (initial) delay between attempts.
	MinDelay time.Duration
	// MaxDelay is the maximum delay between attempts. Zero means no cap beyond MinDelay.
	MaxDelay time.Duration
}

// NewMySQLContextWithRetry opens a MySQL connection, retrying with exponential backoff
// on failure. Use this only for the initial startup connection.
//
// Example:
//
//	db, err := connect.NewMySQLContextWithRetry(ctx, cfg, connect.RetryConfig{
//	    Attempts: 5,
//	    MinDelay: 500 * time.Millisecond,
//	    MaxDelay: 5 * time.Second,
//	})
func NewMySQLContextWithRetry(ctx context.Context, cfg Config, retry RetryConfig) (*sql.DB, error) {
	return newMySQLContextWithRetryFunc(ctx, cfg, retry, NewMySQLContext)
}

// newMySQLContextWithRetryFunc is the testable implementation of NewMySQLContextWithRetry.
// openFunc is injected so that tests can provide a mock without a real MySQL server.
func newMySQLContextWithRetryFunc(
	ctx context.Context,
	cfg Config,
	retry RetryConfig,
	openFunc func(context.Context, Config) (*sql.DB, error),
) (*sql.DB, error) {
	if retry.Attempts <= 1 {
		return openFunc(ctx, cfg)
	}

	var lastErr error
	delay := retry.MinDelay

	for i := 0; i < retry.Attempts; i++ {
		if err := ctx.Err(); err != nil {
			return nil, fmt.Errorf("connect: retry cancelled: %w", err)
		}

		db, err := openFunc(ctx, cfg)
		if err == nil {
			return db, nil
		}
		lastErr = err

		if i == retry.Attempts-1 {
			break
		}

		wait := delay
		if retry.MaxDelay > 0 && wait > retry.MaxDelay {
			wait = retry.MaxDelay
		}
		if wait < 0 {
			wait = 0
		}

		if wait > 0 {
			select {
			case <-ctx.Done():
				return nil, fmt.Errorf("connect: retry cancelled: %w", ctx.Err())
			case <-time.After(wait):
			}
		}

		delay *= 2
		if delay < retry.MinDelay {
			delay = retry.MinDelay
		}
	}

	return nil, fmt.Errorf("connect: all %d attempts failed: %w", retry.Attempts, lastErr)
}

// PingContext verifies that the database connection is alive.
// Returns an error if db is nil.
func PingContext(ctx context.Context, db *sql.DB) error {
	if db == nil {
		return errors.New("connect: PingContext called on nil *sql.DB")
	}
	return db.PingContext(ctx)
}

// Close closes the database and releases all resources.
// Returns nil if db is nil.
func Close(db *sql.DB) error {
	if db == nil {
		return nil
	}
	return db.Close()
}

// applyPool applies connection pool settings to db.
func applyPool(db *sql.DB, cfg Config) {
	db.SetMaxOpenConns(cfg.MaxOpenConns)
	db.SetMaxIdleConns(cfg.MaxIdleConns)
	db.SetConnMaxLifetime(cfg.ConnMaxLifetime)
	db.SetConnMaxIdleTime(cfg.ConnMaxIdleTime)
}
