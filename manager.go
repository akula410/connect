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
// Use NewManager to create a Manager. Zero value is not usable.
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
// If a connection with this name already exists and the normalized Config is equivalent,
// the existing *sql.DB is returned. If the Config differs, an error is returned.
// Returns an error if name is empty.
func (m *Manager) Open(ctx context.Context, name string, cfg Config) (*sql.DB, error) {
	if name == "" {
		return nil, errors.New("connect: connection name must not be empty")
	}

	cfg = NormalizeConfig(cfg)

	m.mu.Lock()
	defer m.mu.Unlock()

	if existing, ok := m.conns[name]; ok {
		if !configsEqual(m.cfgs[name], cfg) {
			return nil, fmt.Errorf("connect: connection %q already registered with a different config", name)
		}
		return existing, nil
	}

	db, err := m.openFunc(ctx, cfg)
	if err != nil {
		return nil, err
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
