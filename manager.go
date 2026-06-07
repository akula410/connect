package connect

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sort"
	"sync"
)

// Manager is a concurrency-safe registry of named MySQL connections.
// Use NewManager to create a Manager. The zero value is not usable.
type Manager struct {
	mu       sync.RWMutex
	conns    map[string]*sql.DB
	cfgs     map[string]Config
	openFunc func(ctx context.Context, cfg Config) (*sql.DB, error)
}

// NewManager creates an empty, ready-to-use Manager.
func NewManager() *Manager {
	return &Manager{
		conns:    make(map[string]*sql.DB),
		cfgs:     make(map[string]Config),
		openFunc: NewMySQLContext,
	}
}

// Open opens a new connection with the given name and stores it in the manager.
//
// If a connection with this name already exists and the normalized Config is equivalent,
// the existing *sql.DB is returned. If the Config differs, an error is returned.
// Returns an error if name is empty.
//
// The mutex is not held during the actual connection or ping, so concurrent Open calls
// for different names proceed in parallel. If two goroutines race to open the same name,
// the slower one closes its extra connection and returns the one saved by the faster goroutine.
func (m *Manager) Open(ctx context.Context, name string, cfg Config) (*sql.DB, error) {
	if name == "" {
		return nil, errors.New("connect: connection name must not be empty")
	}

	cfg = NormalizeConfig(cfg)

	// Fast path: connection already registered.
	m.mu.RLock()
	if existing, ok := m.conns[name]; ok {
		storedCfg := m.cfgs[name]
		m.mu.RUnlock()
		if !configsEqual(storedCfg, cfg) {
			return nil, fmt.Errorf("connect: connection %q already registered with a different config", name)
		}
		return existing, nil
	}
	m.mu.RUnlock()

	// Open the connection outside the lock so other goroutines are not blocked.
	db, err := m.openFunc(ctx, cfg)
	if err != nil {
		return nil, err
	}

	// Write lock: check for a concurrent Open that beat us, then save.
	m.mu.Lock()
	defer m.mu.Unlock()

	if existing, ok := m.conns[name]; ok {
		// Another goroutine registered this name while we were connecting.
		// Close the extra connection we just opened and return the winner's connection.
		_ = db.Close()
		if !configsEqual(m.cfgs[name], cfg) {
			return nil, fmt.Errorf("connect: connection %q already registered with a different config", name)
		}
		return existing, nil
	}

	m.conns[name] = db
	m.cfgs[name] = cfg
	return db, nil
}

// Get returns the *sql.DB registered under name and true, or nil and false if not found.
func (m *Manager) Get(name string) (*sql.DB, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	db, ok := m.conns[name]
	return db, ok
}

// MustGet returns the *sql.DB registered under name or panics if not found.
func (m *Manager) MustGet(name string) *sql.DB {
	db, ok := m.Get(name)
	if !ok {
		panic(fmt.Sprintf("connect: connection %q not found", name))
	}
	return db
}

// Stats returns the sql.DBStats for the connection registered under name.
// Returns (sql.DBStats{}, false) if the name is not found.
// Use this to inspect OpenConnections, InUse, Idle, WaitCount, and other pool metrics.
func (m *Manager) Stats(name string) (sql.DBStats, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	db, ok := m.conns[name]
	if !ok {
		return sql.DBStats{}, false
	}
	return db.Stats(), true
}

// StatsAll returns a snapshot of sql.DBStats for every registered connection,
// keyed by connection name. The returned map is a new allocation on each call.
func (m *Manager) StatsAll() map[string]sql.DBStats {
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make(map[string]sql.DBStats, len(m.conns))
	for name, db := range m.conns {
		result[name] = db.Stats()
	}
	return result
}

// Ping pings the connection registered under name.
func (m *Manager) Ping(ctx context.Context, name string) error {
	db, ok := m.Get(name)
	if !ok {
		return fmt.Errorf("connect: connection %q not found", name)
	}
	return db.PingContext(ctx)
}

// Close closes the connection registered under name and removes it from the manager.
// Returns an error if the name is not found.
func (m *Manager) Close(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	db, ok := m.conns[name]
	if !ok {
		return fmt.Errorf("connect: connection %q not found", name)
	}

	err := db.Close()
	delete(m.conns, name)
	delete(m.cfgs, name)
	return err
}

// CloseAll closes all registered connections and clears the manager.
// It attempts to close every connection and returns a combined error for any failures.
func (m *Manager) CloseAll() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	var errs []error
	for name, db := range m.conns {
		if err := db.Close(); err != nil {
			errs = append(errs, fmt.Errorf("connect: close %q: %w", name, err))
		}
	}

	m.conns = make(map[string]*sql.DB)
	m.cfgs = make(map[string]Config)

	return errors.Join(errs...)
}

// Names returns the names of all registered connections in sorted order.
func (m *Manager) Names() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	names := make([]string, 0, len(m.conns))
	for name := range m.conns {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// configsEqual reports whether two normalized Configs are equivalent.
func configsEqual(a, b Config) bool {
	return a.User == b.User &&
		a.Password == b.Password &&
		a.Host == b.Host &&
		a.Port == b.Port &&
		a.DBName == b.DBName &&
		a.Charset == b.Charset &&
		a.Collation == b.Collation &&
		a.ParseTime == b.ParseTime &&
		a.DisableParseTime == b.DisableParseTime &&
		a.TLSConfig == b.TLSConfig &&
		a.Loc == b.Loc &&
		a.Timeout == b.Timeout &&
		a.ReadTimeout == b.ReadTimeout &&
		a.WriteTimeout == b.WriteTimeout &&
		a.InterpolateParams == b.InterpolateParams &&
		a.MaxOpenConns == b.MaxOpenConns &&
		a.MaxIdleConns == b.MaxIdleConns &&
		a.ConnMaxLifetime == b.ConnMaxLifetime &&
		a.ConnMaxIdleTime == b.ConnMaxIdleTime
}
