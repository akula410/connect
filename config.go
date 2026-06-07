// Package connect provides production-ready MySQL connection management for Go applications.
// It builds safe MySQL DSNs, opens and configures *sql.DB, applies connection pool settings,
// checks connection health via PingContext, and manages named connections through Manager.
// It works with database/sql directly and integrates with github.com/akula410/builder.
package connect

import (
	"errors"
	"fmt"
	"time"

	"github.com/go-sql-driver/mysql"
)

// Config contains MySQL connection and pool settings.
// User and DBName are required. All other fields have safe defaults applied by NormalizeConfig.
type Config struct {
	// User is the database username. Required.
	User string
	// Password is the database password. Not logged or included in error messages.
	Password string
	// Host is the MySQL server host. Defaults to "127.0.0.1".
	Host string
	// Port is the MySQL server port. Defaults to "3306".
	Port string
	// DBName is the database name. Required.
	DBName string

	// Charset is the connection character set. Defaults to "utf8mb4".
	// Important: the charset is not sent to the MySQL driver directly. The effective
	// character set is controlled by Collation — for example, "utf8mb4_unicode_ci"
	// implies charset utf8mb4. Charset is stored for documentation and validation only.
	Charset string
	// Collation is the connection collation sent in SET NAMES <charset> COLLATE <collation>.
	// This controls both the character set and the sort order. Defaults to "utf8mb4_unicode_ci".
	Collation string
	// ParseTime enables automatic parsing of DATE and DATETIME values into time.Time.
	// NormalizeConfig always defaults this to true. To use ParseTime=false, set it
	// explicitly on the Config after calling NormalizeConfig.
	ParseTime bool
	// Loc is the IANA timezone location name used for time values. Defaults to "Local".
	Loc string
	// Timeout is the dial timeout. Defaults to 5s.
	Timeout time.Duration
	// ReadTimeout is the I/O read timeout. Defaults to 30s.
	ReadTimeout time.Duration
	// WriteTimeout is the I/O write timeout. Defaults to 30s.
	WriteTimeout time.Duration
	// InterpolateParams enables client-side interpolation of query parameters.
	// Not compatible with unsafe collations such as big5 or gbk.
	InterpolateParams bool

	// MaxOpenConns is the maximum number of open connections in the pool. Defaults to 25.
	MaxOpenConns int
	// MaxIdleConns is the maximum number of idle connections in the pool. Defaults to 25.
	MaxIdleConns int
	// ConnMaxLifetime is the maximum lifetime of a pooled connection. Defaults to 5m.
	// Keep this below the MySQL wait_timeout to avoid stale connections.
	ConnMaxLifetime time.Duration
	// ConnMaxIdleTime is the maximum idle time of a pooled connection. Defaults to 5m.
	ConnMaxIdleTime time.Duration
}

// DefaultConfig returns a Config with safe production defaults for MySQL 8.
func DefaultConfig() Config {
	return Config{
		Host:            "127.0.0.1",
		Port:            "3306",
		Charset:         "utf8mb4",
		Collation:       "utf8mb4_unicode_ci",
		ParseTime:       true,
		Loc:             "Local",
		Timeout:         5 * time.Second,
		ReadTimeout:     30 * time.Second,
		WriteTimeout:    30 * time.Second,
		MaxOpenConns:    25,
		MaxIdleConns:    25,
		ConnMaxLifetime: 5 * time.Minute,
		ConnMaxIdleTime: 5 * time.Minute,
	}
}

// NormalizeConfig fills zero-value fields with safe defaults from DefaultConfig.
// User, Password, and DBName are never overwritten.
//
// ParseTime is always defaulted to true. Because false is the zero value for bool,
// NormalizeConfig cannot distinguish "not set" from "explicitly false".
// To use ParseTime=false, set cfg.ParseTime = false after calling NormalizeConfig.
func NormalizeConfig(cfg Config) Config {
	d := DefaultConfig()
	if cfg.Host == "" {
		cfg.Host = d.Host
	}
	if cfg.Port == "" {
		cfg.Port = d.Port
	}
	if cfg.Charset == "" {
		cfg.Charset = d.Charset
	}
	if cfg.Collation == "" {
		cfg.Collation = d.Collation
	}
	if !cfg.ParseTime {
		cfg.ParseTime = d.ParseTime
	}
	if cfg.Loc == "" {
		cfg.Loc = d.Loc
	}
	if cfg.Timeout == 0 {
		cfg.Timeout = d.Timeout
	}
	if cfg.ReadTimeout == 0 {
		cfg.ReadTimeout = d.ReadTimeout
	}
	if cfg.WriteTimeout == 0 {
		cfg.WriteTimeout = d.WriteTimeout
	}
	if cfg.MaxOpenConns == 0 {
		cfg.MaxOpenConns = d.MaxOpenConns
	}
	if cfg.MaxIdleConns == 0 {
		cfg.MaxIdleConns = d.MaxIdleConns
	}
	if cfg.ConnMaxLifetime == 0 {
		cfg.ConnMaxLifetime = d.ConnMaxLifetime
	}
	if cfg.ConnMaxIdleTime == 0 {
		cfg.ConnMaxIdleTime = d.ConnMaxIdleTime
	}
	return cfg
}

// ValidateConfig checks that all required fields are present.
// Call NormalizeConfig before ValidateConfig to apply defaults first.
func ValidateConfig(cfg Config) error {
	if cfg.User == "" {
		return errors.New("connect: User is required")
	}
	if cfg.DBName == "" {
		return errors.New("connect: DBName is required")
	}
	if cfg.Host == "" {
		return errors.New("connect: Host is required")
	}
	if cfg.Port == "" {
		return errors.New("connect: Port is required")
	}
	return nil
}

// DSN formats and returns the MySQL DSN string for the given Config.
// It applies NormalizeConfig and ValidateConfig before formatting.
// The returned string contains the password if set — do not log it.
func DSN(cfg Config) (string, error) {
	cfg = NormalizeConfig(cfg)
	if err := ValidateConfig(cfg); err != nil {
		return "", err
	}
	mc, err := buildMySQLConfig(cfg)
	if err != nil {
		return "", err
	}
	return mc.FormatDSN(), nil
}

// buildMySQLConfig converts Config into a mysql.Config used by the driver.
// cfg must already be normalized and validated.
// Note: cfg.Charset is not passed to the driver directly; the effective charset
// is derived from cfg.Collation by the MySQL driver.
func buildMySQLConfig(cfg Config) (*mysql.Config, error) {
	loc, err := time.LoadLocation(cfg.Loc)
	if err != nil {
		return nil, fmt.Errorf("connect: invalid Loc %q: %w", cfg.Loc, err)
	}

	mc := mysql.NewConfig()
	mc.User = cfg.User
	mc.Passwd = cfg.Password
	mc.Net = "tcp"
	mc.Addr = cfg.Host + ":" + cfg.Port
	mc.DBName = cfg.DBName
	mc.Collation = cfg.Collation
	mc.Loc = loc
	mc.ParseTime = cfg.ParseTime
	mc.InterpolateParams = cfg.InterpolateParams
	mc.Timeout = cfg.Timeout
	mc.ReadTimeout = cfg.ReadTimeout
	mc.WriteTimeout = cfg.WriteTimeout
	return mc, nil
}
