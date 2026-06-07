package connect

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/go-sql-driver/mysql"
)

// ---- DefaultConfig ----

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	checks := []struct {
		name string
		got  interface{}
		want interface{}
	}{
		{"Host", cfg.Host, "127.0.0.1"},
		{"Port", cfg.Port, "3306"},
		{"Charset", cfg.Charset, "utf8mb4"},
		{"Collation", cfg.Collation, "utf8mb4_unicode_ci"},
		{"Loc", cfg.Loc, "Local"},
		{"MaxOpenConns", cfg.MaxOpenConns, 25},
		{"MaxIdleConns", cfg.MaxIdleConns, 25},
	}
	for _, c := range checks {
		if c.got != c.want {
			t.Errorf("%s: got %v, want %v", c.name, c.got, c.want)
		}
	}
	if !cfg.ParseTime {
		t.Error("ParseTime: want true")
	}
	if cfg.DisableParseTime {
		t.Error("DisableParseTime: want false")
	}
	if cfg.Timeout != 5*time.Second {
		t.Errorf("Timeout: got %v, want 5s", cfg.Timeout)
	}
	if cfg.ReadTimeout != 30*time.Second {
		t.Errorf("ReadTimeout: got %v, want 30s", cfg.ReadTimeout)
	}
	if cfg.WriteTimeout != 30*time.Second {
		t.Errorf("WriteTimeout: got %v, want 30s", cfg.WriteTimeout)
	}
	if cfg.ConnMaxLifetime != 5*time.Minute {
		t.Errorf("ConnMaxLifetime: got %v, want 5m", cfg.ConnMaxLifetime)
	}
	if cfg.ConnMaxIdleTime != 5*time.Minute {
		t.Errorf("ConnMaxIdleTime: got %v, want 5m", cfg.ConnMaxIdleTime)
	}
}

// ---- NormalizeConfig ----

func TestNormalizeConfig_AppliesDefaults(t *testing.T) {
	cfg := NormalizeConfig(Config{User: "u", DBName: "db"})

	if cfg.Host != "127.0.0.1" {
		t.Errorf("Host: got %q", cfg.Host)
	}
	if cfg.Port != "3306" {
		t.Errorf("Port: got %q", cfg.Port)
	}
	if cfg.Charset != "utf8mb4" {
		t.Errorf("Charset: got %q", cfg.Charset)
	}
	if cfg.Collation != "utf8mb4_unicode_ci" {
		t.Errorf("Collation: got %q", cfg.Collation)
	}
	if !cfg.ParseTime {
		t.Error("ParseTime: want true")
	}
}

func TestNormalizeConfig_PreservesExplicitValues(t *testing.T) {
	cfg := NormalizeConfig(Config{
		User:     "u",
		Password: "secret",
		Host:     "db.internal",
		Port:     "3307",
		DBName:   "mydb",
		Charset:  "utf8",
		Loc:      "UTC",
	})

	if cfg.Host != "db.internal" {
		t.Errorf("Host: got %q, want db.internal", cfg.Host)
	}
	if cfg.Port != "3307" {
		t.Errorf("Port: got %q, want 3307", cfg.Port)
	}
	if cfg.Charset != "utf8" {
		t.Errorf("Charset: got %q, want utf8", cfg.Charset)
	}
	if cfg.Loc != "UTC" {
		t.Errorf("Loc: got %q, want UTC", cfg.Loc)
	}
}

func TestNormalizeConfig_ParseTimeDefaultsToTrue(t *testing.T) {
	cfg := NormalizeConfig(Config{User: "u", DBName: "db"})
	if !cfg.ParseTime {
		t.Error("ParseTime: NormalizeConfig should default to true")
	}
}

func TestNormalizeConfig_ParseTimeCanBeDisabledAfterNormalize(t *testing.T) {
	// ParseTime cannot be set to false through NormalizeConfig because false is
	// the zero value for bool. Disable it explicitly after normalization.
	cfg := NormalizeConfig(Config{User: "u", DBName: "db"})
	cfg.ParseTime = false
	if cfg.ParseTime {
		t.Error("ParseTime: should be false after explicit override")
	}
}

func TestNormalizeConfig_DisableParseTime(t *testing.T) {
	cfg := NormalizeConfig(Config{User: "u", DBName: "db", DisableParseTime: true})
	if cfg.ParseTime {
		t.Error("ParseTime: should be false when DisableParseTime=true")
	}
	if !cfg.DisableParseTime {
		t.Error("DisableParseTime: should remain true after NormalizeConfig")
	}
}

func TestNormalizeConfig_DisableParseTime_OverridesExplicitTrue(t *testing.T) {
	// DisableParseTime=true takes precedence even if ParseTime=true was also set.
	cfg := NormalizeConfig(Config{User: "u", DBName: "db", ParseTime: true, DisableParseTime: true})
	if cfg.ParseTime {
		t.Error("ParseTime: DisableParseTime=true should override ParseTime=true")
	}
}

func TestNormalizeConfig_CharsetIsInformational(t *testing.T) {
	// Charset is stored but not passed to the driver directly;
	// the effective charset is set by Collation.
	cfg := NormalizeConfig(Config{User: "u", DBName: "db", Charset: "latin1"})
	if cfg.Charset != "latin1" {
		t.Errorf("Charset: got %q, want latin1", cfg.Charset)
	}
	// Collation still defaults to utf8mb4_unicode_ci regardless of Charset.
	if cfg.Collation != "utf8mb4_unicode_ci" {
		t.Errorf("Collation: got %q, want utf8mb4_unicode_ci", cfg.Collation)
	}
}

func TestNormalizeConfig_TLSConfig_Preserved(t *testing.T) {
	cfg := NormalizeConfig(Config{User: "u", DBName: "db", TLSConfig: "skip-verify"})
	if cfg.TLSConfig != "skip-verify" {
		t.Errorf("TLSConfig: got %q, want skip-verify", cfg.TLSConfig)
	}
}

func TestNormalizeConfig_Timeouts(t *testing.T) {
	cfg := NormalizeConfig(Config{
		User:         "u",
		DBName:       "db",
		Timeout:      10 * time.Second,
		ReadTimeout:  60 * time.Second,
		WriteTimeout: 60 * time.Second,
	})
	if cfg.Timeout != 10*time.Second {
		t.Errorf("Timeout: got %v, want 10s", cfg.Timeout)
	}
	if cfg.ReadTimeout != 60*time.Second {
		t.Errorf("ReadTimeout: got %v, want 60s", cfg.ReadTimeout)
	}
	if cfg.WriteTimeout != 60*time.Second {
		t.Errorf("WriteTimeout: got %v, want 60s", cfg.WriteTimeout)
	}
}

// ---- ValidateConfig ----

func TestValidateConfig_EmptyUser(t *testing.T) {
	cfg := NormalizeConfig(Config{DBName: "db"})
	if err := ValidateConfig(cfg); err == nil {
		t.Error("want error for empty User")
	}
}

func TestValidateConfig_EmptyDBName(t *testing.T) {
	cfg := NormalizeConfig(Config{User: "u"})
	if err := ValidateConfig(cfg); err == nil {
		t.Error("want error for empty DBName")
	}
}

func TestValidateConfig_ValidAfterNormalize(t *testing.T) {
	cfg := NormalizeConfig(Config{User: "u", DBName: "db"})
	if err := ValidateConfig(cfg); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

// ---- DSN ----

func parseDSN(t *testing.T, dsn string) *mysql.Config {
	t.Helper()
	mc, err := mysql.ParseDSN(dsn)
	if err != nil {
		t.Fatalf("ParseDSN(%q): %v", dsn, err)
	}
	return mc
}

func TestDSN_BasicFields(t *testing.T) {
	dsn, err := DSN(Config{User: "alice", Password: "s3cr3t", DBName: "mydb"})
	if err != nil {
		t.Fatalf("DSN: %v", err)
	}
	mc := parseDSN(t, dsn)

	if mc.User != "alice" {
		t.Errorf("User: got %q, want alice", mc.User)
	}
	if mc.Passwd != "s3cr3t" {
		t.Errorf("Passwd: got %q, want s3cr3t", mc.Passwd)
	}
	if mc.DBName != "mydb" {
		t.Errorf("DBName: got %q, want mydb", mc.DBName)
	}
	if mc.Addr != "127.0.0.1:3306" {
		t.Errorf("Addr: got %q, want 127.0.0.1:3306", mc.Addr)
	}
}

func TestDSN_WithoutPassword(t *testing.T) {
	dsn, err := DSN(Config{User: "alice", DBName: "mydb"})
	if err != nil {
		t.Fatalf("DSN: %v", err)
	}
	mc := parseDSN(t, dsn)
	if mc.Passwd != "" {
		t.Errorf("Passwd should be empty, got %q", mc.Passwd)
	}
}

func TestDSN_PasswordWithSpecialChars(t *testing.T) {
	pass := "p@ss!w0rd#$%^&*()"
	dsn, err := DSN(Config{User: "u", Password: pass, DBName: "db"})
	if err != nil {
		t.Fatalf("DSN: %v", err)
	}
	mc := parseDSN(t, dsn)
	if mc.Passwd != pass {
		t.Errorf("Passwd round-trip: got %q, want %q", mc.Passwd, pass)
	}
}

func TestDSN_ParseTime(t *testing.T) {
	dsn, err := DSN(Config{User: "u", DBName: "db", ParseTime: true})
	if err != nil {
		t.Fatalf("DSN: %v", err)
	}
	if !strings.Contains(dsn, "parseTime=true") {
		t.Errorf("DSN missing parseTime=true: %q", dsn)
	}
}

func TestDSN_ParseTimeFalseViaDisableParseTime(t *testing.T) {
	dsn, err := DSN(Config{User: "u", DBName: "db", DisableParseTime: true})
	if err != nil {
		t.Fatalf("DSN: %v", err)
	}
	if strings.Contains(dsn, "parseTime=true") {
		t.Errorf("DSN should not contain parseTime=true when DisableParseTime=true: %q", dsn)
	}
}

func TestDSN_Collation(t *testing.T) {
	dsn, err := DSN(Config{User: "u", DBName: "db"})
	if err != nil {
		t.Fatalf("DSN: %v", err)
	}
	if !strings.Contains(dsn, "collation=utf8mb4_unicode_ci") {
		t.Errorf("DSN missing collation: %q", dsn)
	}
}

func TestDSN_InterpolateParams(t *testing.T) {
	dsn, err := DSN(Config{User: "u", DBName: "db", InterpolateParams: true})
	if err != nil {
		t.Fatalf("DSN: %v", err)
	}
	if !strings.Contains(dsn, "interpolateParams=true") {
		t.Errorf("DSN missing interpolateParams: %q", dsn)
	}
}

func TestDSN_Timeouts(t *testing.T) {
	dsn, err := DSN(Config{
		User:         "u",
		DBName:       "db",
		Timeout:      3 * time.Second,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	})
	if err != nil {
		t.Fatalf("DSN: %v", err)
	}
	for _, sub := range []string{"timeout=3s", "readTimeout=10s", "writeTimeout=10s"} {
		if !strings.Contains(dsn, sub) {
			t.Errorf("DSN missing %q in: %q", sub, dsn)
		}
	}
}

func TestDSN_TLSConfig(t *testing.T) {
	dsn, err := DSN(Config{User: "u", DBName: "db", TLSConfig: "skip-verify"})
	if err != nil {
		t.Fatalf("DSN: %v", err)
	}
	if !strings.Contains(dsn, "tls=skip-verify") {
		t.Errorf("DSN missing tls=skip-verify: %q", dsn)
	}
}

func TestDSN_TLSConfig_Empty(t *testing.T) {
	dsn, err := DSN(Config{User: "u", DBName: "db"})
	if err != nil {
		t.Fatalf("DSN: %v", err)
	}
	if strings.Contains(dsn, "tls=") {
		t.Errorf("DSN should not contain tls= when TLSConfig is empty: %q", dsn)
	}
}

func TestDSN_InvalidLoc(t *testing.T) {
	_, err := DSN(Config{User: "u", DBName: "db", Loc: "Not/A/Valid/Location"})
	if err == nil {
		t.Error("want error for invalid Loc")
	}
}

func TestDSN_EmptyUser(t *testing.T) {
	_, err := DSN(Config{DBName: "db"})
	if err == nil {
		t.Error("want error for empty User")
	}
}

func TestDSN_EmptyDBName(t *testing.T) {
	_, err := DSN(Config{User: "u"})
	if err == nil {
		t.Error("want error for empty DBName")
	}
}

// ---- Nil protection ----

func TestPingContext_NilDB(t *testing.T) {
	err := PingContext(context.Background(), nil)
	if err == nil {
		t.Error("PingContext(ctx, nil): want error, got nil")
	}
	if !errors.Is(err, err) { // just check it's a real error, not a panic
		t.Errorf("PingContext(ctx, nil): unexpected error type: %v", err)
	}
}

func TestClose_NilDB(t *testing.T) {
	if err := Close(nil); err != nil {
		t.Errorf("Close(nil): want nil, got %v", err)
	}
}

// ---- Manager helpers ----

// newTestManager creates a Manager that opens *sql.DB without pinging,
// so tests run without a real MySQL server.
func newTestManager() *Manager {
	m := NewManager()
	m.openFunc = func(_ context.Context, cfg Config) (*sql.DB, error) {
		dsn, err := DSN(cfg)
		if err != nil {
			return nil, err
		}
		return sql.Open("mysql", dsn)
	}
	return m
}

func validCfg() Config {
	return Config{User: "u", Password: "p", DBName: "db"}
}

// ---- Manager basic tests ----

func TestNewManager_NotNil(t *testing.T) {
	m := NewManager()
	if m == nil {
		t.Fatal("NewManager returned nil")
	}
}

func TestManager_Open_EmptyName(t *testing.T) {
	m := newTestManager()
	_, err := m.Open(context.Background(), "", validCfg())
	if err == nil {
		t.Error("want error for empty name")
	}
}

func TestManager_Get_Missing(t *testing.T) {
	m := newTestManager()
	_, ok := m.Get("nonexistent")
	if ok {
		t.Error("want false for missing connection")
	}
}

func TestManager_Open_StoreGet(t *testing.T) {
	m := newTestManager()
	db, err := m.Open(context.Background(), "main", validCfg())
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	if db == nil {
		t.Fatal("got nil *sql.DB")
	}

	got, ok := m.Get("main")
	if !ok {
		t.Fatal("Get: not found after Open")
	}
	if got != db {
		t.Error("Get returned different *sql.DB than Open")
	}
}

func TestManager_Open_SameConfigReturnsSameDB(t *testing.T) {
	m := newTestManager()
	cfg := validCfg()

	db1, err := m.Open(context.Background(), "main", cfg)
	if err != nil {
		t.Fatalf("first Open: %v", err)
	}
	db2, err := m.Open(context.Background(), "main", cfg)
	if err != nil {
		t.Fatalf("second Open: %v", err)
	}
	if db1 != db2 {
		t.Error("second Open with same config should return the same *sql.DB")
	}
}

func TestManager_Open_DifferentConfigReturnsError(t *testing.T) {
	m := newTestManager()

	cfg1 := Config{User: "u", Password: "p", DBName: "db1"}
	cfg2 := Config{User: "u", Password: "p", DBName: "db2"}

	if _, err := m.Open(context.Background(), "main", cfg1); err != nil {
		t.Fatalf("first Open: %v", err)
	}
	_, err := m.Open(context.Background(), "main", cfg2)
	if err == nil {
		t.Error("want error when reopening same name with different config")
	}
}

func TestManager_Open_ConcurrentSameName(t *testing.T) {
	// All goroutines open the same name with the same config concurrently.
	// All should succeed and get back the same *sql.DB pointer.
	m := newTestManager()

	const goroutines = 20
	var wg sync.WaitGroup
	var successCount int64
	var errCount int64

	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			_, err := m.Open(context.Background(), "concurrent", validCfg())
			if err != nil {
				atomic.AddInt64(&errCount, 1)
			} else {
				atomic.AddInt64(&successCount, 1)
			}
		}()
	}
	wg.Wait()

	if successCount != goroutines {
		t.Errorf("expected %d successes, got %d (errors: %d)", goroutines, successCount, errCount)
	}
	if names := m.Names(); len(names) != 1 {
		t.Errorf("expected 1 registered connection, got %d: %v", len(names), names)
	}
}

func TestManager_MustGet_Panics(t *testing.T) {
	m := newTestManager()
	defer func() {
		if r := recover(); r == nil {
			t.Error("MustGet should panic for missing connection")
		}
	}()
	m.MustGet("nothere")
}

func TestManager_MustGet_Returns(t *testing.T) {
	m := newTestManager()
	db, err := m.Open(context.Background(), "main", validCfg())
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	got := m.MustGet("main")
	if got != db {
		t.Error("MustGet returned different *sql.DB than Open")
	}
}

func TestManager_Close_RemovesConnection(t *testing.T) {
	m := newTestManager()
	if _, err := m.Open(context.Background(), "main", validCfg()); err != nil {
		t.Fatalf("Open: %v", err)
	}
	if err := m.Close("main"); err != nil {
		t.Fatalf("Close: %v", err)
	}
	_, ok := m.Get("main")
	if ok {
		t.Error("Get should return false after Close")
	}
}

func TestManager_Close_MissingReturnsError(t *testing.T) {
	m := newTestManager()
	if err := m.Close("nonexistent"); err == nil {
		t.Error("want error when closing missing connection")
	}
}

func TestManager_CloseAll_ClearsAll(t *testing.T) {
	m := newTestManager()
	for _, name := range []string{"a", "b", "c"} {
		cfg := Config{User: "u", Password: "p", DBName: name}
		if _, err := m.Open(context.Background(), name, cfg); err != nil {
			t.Fatalf("Open %q: %v", name, err)
		}
	}
	if err := m.CloseAll(); err != nil {
		t.Fatalf("CloseAll: %v", err)
	}
	if names := m.Names(); len(names) != 0 {
		t.Errorf("Names after CloseAll: got %v, want []", names)
	}
}

func TestManager_Names_Sorted(t *testing.T) {
	m := newTestManager()
	for _, name := range []string{"z", "a", "m"} {
		cfg := Config{User: "u", Password: "p", DBName: name}
		if _, err := m.Open(context.Background(), name, cfg); err != nil {
			t.Fatalf("Open %q: %v", name, err)
		}
	}
	names := m.Names()
	want := []string{"a", "m", "z"}
	if len(names) != len(want) {
		t.Fatalf("Names: got %v, want %v", names, want)
	}
	for i := range want {
		if names[i] != want[i] {
			t.Errorf("Names[%d]: got %q, want %q", i, names[i], want[i])
		}
	}
}

func TestManager_Names_EmptyManager(t *testing.T) {
	m := newTestManager()
	if names := m.Names(); len(names) != 0 {
		t.Errorf("Names on empty manager: got %v", names)
	}
}

func TestManager_ConcurrentAccess(t *testing.T) {
	m := newTestManager()

	if _, err := m.Open(context.Background(), "shared", validCfg()); err != nil {
		t.Fatalf("Open: %v", err)
	}

	var wg sync.WaitGroup
	const goroutines = 50
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _ = m.Get("shared")
			_ = m.Names()
		}()
	}
	wg.Wait()
}

// ---- Manager Stats ----

func TestManager_Stats_Found(t *testing.T) {
	m := newTestManager()
	if _, err := m.Open(context.Background(), "main", validCfg()); err != nil {
		t.Fatalf("Open: %v", err)
	}
	_, ok := m.Stats("main")
	if !ok {
		t.Error("Stats: want true for existing connection")
	}
}

func TestManager_Stats_NotFound(t *testing.T) {
	m := newTestManager()
	_, ok := m.Stats("nonexistent")
	if ok {
		t.Error("Stats: want false for missing connection")
	}
}

func TestManager_StatsAll(t *testing.T) {
	m := newTestManager()
	for _, name := range []string{"a", "b"} {
		cfg := Config{User: "u", Password: "p", DBName: name}
		if _, err := m.Open(context.Background(), name, cfg); err != nil {
			t.Fatalf("Open %q: %v", name, err)
		}
	}

	all := m.StatsAll()
	if len(all) != 2 {
		t.Errorf("StatsAll: got %d entries, want 2", len(all))
	}
	for _, key := range []string{"a", "b"} {
		if _, ok := all[key]; !ok {
			t.Errorf("StatsAll: missing key %q", key)
		}
	}
}

func TestManager_StatsAll_Empty(t *testing.T) {
	m := newTestManager()
	all := m.StatsAll()
	if len(all) != 0 {
		t.Errorf("StatsAll on empty manager: got %d entries, want 0", len(all))
	}
}

// ---- configsEqual ----

func TestConfigsEqual_SameConfig(t *testing.T) {
	cfg := NormalizeConfig(validCfg())
	if !configsEqual(cfg, cfg) {
		t.Error("configsEqual: same config should be equal")
	}
}

func TestConfigsEqual_DifferentDBName(t *testing.T) {
	a := NormalizeConfig(Config{User: "u", DBName: "db1"})
	b := NormalizeConfig(Config{User: "u", DBName: "db2"})
	if configsEqual(a, b) {
		t.Error("configsEqual: configs with different DBName should not be equal")
	}
}

func TestConfigsEqual_DifferentTLSConfig(t *testing.T) {
	a := NormalizeConfig(Config{User: "u", DBName: "db", TLSConfig: "true"})
	b := NormalizeConfig(Config{User: "u", DBName: "db", TLSConfig: "skip-verify"})
	if configsEqual(a, b) {
		t.Error("configsEqual: configs with different TLSConfig should not be equal")
	}
}

func TestConfigsEqual_DifferentDisableParseTime(t *testing.T) {
	a := NormalizeConfig(Config{User: "u", DBName: "db"})
	b := NormalizeConfig(Config{User: "u", DBName: "db", DisableParseTime: true})
	if configsEqual(a, b) {
		t.Error("configsEqual: configs with different DisableParseTime should not be equal")
	}
}

// ---- Retry ----

// mockDB returns a *sql.DB that is usable for tests without a real MySQL server.
func mockDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("mysql", "u:p@/db")
	if err != nil {
		t.Fatalf("sql.Open: %v", err)
	}
	return db
}

func TestRetry_SuccessFirstAttempt(t *testing.T) {
	calls := 0
	open := func(_ context.Context, _ Config) (*sql.DB, error) {
		calls++
		return mockDB(t), nil
	}

	db, err := newMySQLContextWithRetryFunc(context.Background(), validCfg(),
		RetryConfig{Attempts: 3}, open)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	db.Close()
	if calls != 1 {
		t.Errorf("expected 1 call, got %d", calls)
	}
}

func TestRetry_SuccessAfterFailures(t *testing.T) {
	calls := 0
	open := func(_ context.Context, _ Config) (*sql.DB, error) {
		calls++
		if calls < 3 {
			return nil, fmt.Errorf("connection refused (attempt %d)", calls)
		}
		return mockDB(t), nil
	}

	db, err := newMySQLContextWithRetryFunc(context.Background(), validCfg(),
		RetryConfig{Attempts: 5, MinDelay: 0, MaxDelay: 0}, open)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	db.Close()
	if calls != 3 {
		t.Errorf("expected 3 calls, got %d", calls)
	}
}

func TestRetry_StopsAfterMaxAttempts(t *testing.T) {
	calls := 0
	open := func(_ context.Context, _ Config) (*sql.DB, error) {
		calls++
		return nil, fmt.Errorf("connection refused")
	}

	_, err := newMySQLContextWithRetryFunc(context.Background(), validCfg(),
		RetryConfig{Attempts: 4, MinDelay: 0, MaxDelay: 0}, open)
	if err == nil {
		t.Fatal("expected error after all attempts")
	}
	if calls != 4 {
		t.Errorf("expected 4 calls, got %d", calls)
	}
}

func TestRetry_StopsOnContextAlreadyCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	calls := 0
	open := func(_ context.Context, _ Config) (*sql.DB, error) {
		calls++
		return nil, fmt.Errorf("connection refused")
	}

	_, err := newMySQLContextWithRetryFunc(ctx, validCfg(),
		RetryConfig{Attempts: 5, MinDelay: 0, MaxDelay: 0}, open)
	if err == nil {
		t.Fatal("expected error when context is cancelled")
	}
	// Context already cancelled before first attempt: should not retry.
	if calls > 1 {
		t.Errorf("expected at most 1 call with pre-cancelled context, got %d", calls)
	}
}

func TestRetry_StopsOnContextCancelledDuringRetry(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	calls := 0
	open := func(_ context.Context, _ Config) (*sql.DB, error) {
		calls++
		if calls == 2 {
			cancel() // cancel during second attempt
		}
		return nil, fmt.Errorf("connection refused")
	}

	// Use a tiny delay to ensure the select fires context cancellation.
	_, err := newMySQLContextWithRetryFunc(ctx, validCfg(),
		RetryConfig{Attempts: 10, MinDelay: time.Millisecond, MaxDelay: time.Millisecond}, open)
	if err == nil {
		t.Fatal("expected error when context is cancelled")
	}
	// Should stop well before 10 attempts.
	if calls >= 10 {
		t.Errorf("expected retry to stop early on ctx cancel, got %d calls", calls)
	}
}

func TestRetry_AttemptsOneOrLess(t *testing.T) {
	for _, attempts := range []int{-1, 0, 1} {
		calls := 0
		open := func(_ context.Context, _ Config) (*sql.DB, error) {
			calls++
			return nil, fmt.Errorf("connection refused")
		}

		_, err := newMySQLContextWithRetryFunc(context.Background(), validCfg(),
			RetryConfig{Attempts: attempts}, open)
		if err == nil {
			t.Fatalf("Attempts=%d: expected error", attempts)
		}
		if calls != 1 {
			t.Errorf("Attempts=%d: expected exactly 1 call, got %d", attempts, calls)
		}
	}
}

// ---- Integration tests (require CONNECT_MYSQL_DSN env var) ----

func TestIntegration_NewMySQLContext(t *testing.T) {
	dsn, ok := os.LookupEnv("CONNECT_MYSQL_DSN")
	if !ok || dsn == "" {
		t.Skip("CONNECT_MYSQL_DSN not set; skipping integration test")
	}

	mc, err := mysql.ParseDSN(dsn)
	if err != nil {
		t.Fatalf("parse CONNECT_MYSQL_DSN: %v", err)
	}

	// mc.Addr is "host:port"; split to avoid "host:port:3306" when Port is applied.
	host, port, err := net.SplitHostPort(mc.Addr)
	if err != nil {
		t.Fatalf("split host/port from %q: %v", mc.Addr, err)
	}

	cfg := Config{
		User:     mc.User,
		Password: mc.Passwd,
		Host:     host,
		Port:     port,
		DBName:   mc.DBName,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	db, err := NewMySQLContext(ctx, cfg)
	if err != nil {
		t.Fatalf("NewMySQLContext: %v", err)
	}
	defer db.Close()

	if err := db.PingContext(ctx); err != nil {
		t.Fatalf("Ping after connect: %v", err)
	}
}
