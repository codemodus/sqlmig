package sqlmig

import (
	"context"
	"database/sql"
	"fmt"
	"sync"

	migrate "github.com/rubenv/sql-migrate"
)

// Migrator ...
type Migrator interface {
	MigrationGetter
	Regularizer
}

// MigrationGetter ...
type MigrationGetter interface {
	MigrationName() string
	MigrationAssetDirectory() string
	MigrationAssetNames(dir string) (filenames []string, err error)
	MigrationAsset(filename string) (data []byte, err error)
}

// Regularizer ...
type Regularizer interface {
	Regularize(context.Context) error
}

// SQLMig ...
type SQLMig struct {
	*sql.DB
	drvr string
	mu   sync.Mutex
	srcs []MigrationGetter
	regs []Regularizer
}

// New ...
func New(db *sql.DB, driver string) (*SQLMig, error) {
	m := SQLMig{
		DB:   db,
		drvr: driver,
	}

	return &m, nil
}

// AddMigrations ...
func (m *SQLMig) AddMigrations(srcs ...MigrationGetter) {
	if srcs == nil || len(srcs) == 0 {
		return
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	m.srcs = append(m.srcs, srcs...)
}

// RunMigrations ...
func (m *SQLMig) RunMigrations(src MigrationGetter, up bool) *Result {
	dir := migrate.Up
	if !up {
		dir = migrate.Down
	}

	msrc := migrate.AssetMigrationSource{
		Asset:    src.MigrationAsset,
		AssetDir: src.MigrationAssetNames,
		Dir:      src.MigrationAssetDirectory(),
	}

	ct, err := migrate.Exec(m.DB, m.drvr, &msrc, dir)

	return &Result{
		name: src.MigrationName(),
		ct:   ct,
		dir:  src.MigrationAssetDirectory(),
		err:  err,
	}
}

func (m *SQLMig) migrate(up bool) Results {
	var rs Results

	m.mu.Lock()
	defer m.mu.Unlock()

	for _, src := range m.srcs {
		r := m.RunMigrations(src, up)
		rs = append(rs, r)
	}

	return rs
}

// Migrate ...
func (m *SQLMig) Migrate() Results {
	return m.migrate(true)
}

// RollBack ...
func (m *SQLMig) RollBack() Results {
	return m.migrate(false)
}

// AddRegularizations ...
func (m *SQLMig) AddRegularizations(regs ...Regularizer) {
	if regs == nil || len(regs) == 0 {
		return
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	m.regs = append(m.regs, regs...)
}

// Regularize ...
func (m *SQLMig) Regularize(ctx context.Context) error {
	for _, reg := range m.regs {
		if err := reg.Regularize(ctx); err != nil {
			return err
		}
	}
	return nil
}

// Result ...
type Result struct {
	name string
	ct   int
	dir  string
	err  error
}

// Total ...
func (r *Result) Total() int {
	return r.ct
}

// Directory ...
func (r *Result) Directory() string {
	return r.dir
}

func (r *Result) String() string {
	return fmt.Sprintf("%s (%d)", r.name, r.ct)
}

// Err ...
func (r *Result) Err() error {
	return r.err
}

// Results ...
type Results []*Result

// Total ...
func (rs Results) Total() int {
	var t int
	for _, r := range rs {
		t += r.ct
	}
	return t
}

func (rs Results) String() string {
	var names, sep string

	for _, r := range rs {
		if r.name == "" {
			continue
		}

		names += sep + r.name
		sep = ", "
	}

	return fmt.Sprintf("%s (%d)", names, rs.Total())
}

// HasError ...
func (rs Results) HasError() bool {
	for _, r := range rs {
		if r.err != nil {
			return true
		}
	}
	return false
}

// Errs ...
func (rs Results) Errs() []error {
	var errs []error
	for _, r := range rs {
		if r.err != nil {
			errs = append(errs, r.err)
		}
	}
	return errs
}

// ErrsErr ...
func (rs Results) ErrsErr() error {
	var e, sep string

	for _, err := range rs.Errs() {
		if err == nil {
			continue
		}

		e += sep + err.Error()
		sep = ", "
	}

	return fmt.Errorf(e)
}
