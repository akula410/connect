package connect

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

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
