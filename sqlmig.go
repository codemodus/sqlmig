package sqlmig

import (
	"context"
	"database/sql"
	"fmt"
	"sync"

	migrate "github.com/rubenv/sql-migrate"
)

// DataProvider ...
type DataProvider interface {
	MigrationData() (name string, data map[string][]byte)
}

// Regularizer ...
type Regularizer interface {
	Regularize(context.Context) error
}

// SQLMig ...
type SQLMig struct {
	*sql.DB
	drv string
	tp  string
	mu  sync.Mutex
	ps  []DataProvider
	rs  []Regularizer
}

// New ...
func New(db *sql.DB, driver, tablePrefix string) (*SQLMig, error) {
	m := SQLMig{
		DB:  db,
		drv: driver,
		tp:  tablePrefix,
	}

	return &m, nil
}

// AddDataProviders ...
func (m *SQLMig) AddDataProviders(ps ...DataProvider) {
	if ps == nil || len(ps) == 0 {
		return
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	m.ps = append(m.ps, ps...)
}

func (m *SQLMig) runMigration(p DataProvider, up bool) *Result {
	dir := migrate.Up
	if !up {
		dir = migrate.Down
	}

	name, data := p.MigrationData()

	msrc := newAssetMigrationSource(data)
	migrate.SetTable(m.tp + "_" + name)
	ct, err := migrate.Exec(m.DB, m.drv, msrc, dir)

	return &Result{
		name: name,
		ct:   ct,
		err:  err,
	}
}

func (m *SQLMig) migrate(up bool) Results {
	var rs Results

	m.mu.Lock()
	defer m.mu.Unlock()

	for _, p := range m.ps {
		r := m.runMigration(p, up)
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

// AddRegularizers ...
func (m *SQLMig) AddRegularizers(rs ...Regularizer) {
	if rs == nil || len(rs) == 0 {
		return
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	m.rs = append(m.rs, rs...)
}

// Regularize ...
func (m *SQLMig) Regularize(ctx context.Context) error {
	for _, r := range m.rs {
		if err := r.Regularize(ctx); err != nil {
			return err
		}
	}
	return nil
}

// Result ...
type Result struct {
	name string
	ct   int
	err  error
}

// Total ...
func (r *Result) Total() int {
	return r.ct
}

func (r *Result) String() string {
	name := r.name
	if name == "" {
		name = "-unnamed-"
	}
	return fmt.Sprintf("%s (%d)", name, r.ct)
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
	var s, sep string

	for _, r := range rs {
		s += sep + r.String()
		sep = ", "
	}

	div := " - "
	if len(s) == 0 {
		div = ""
	}

	return fmt.Sprintf("%s%stotal: %d", s, div, rs.Total())
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

func newAssetMigrationSource(m map[string][]byte) *migrate.AssetMigrationSource {
	if m == nil || len(m) == 0 {
		return &migrate.AssetMigrationSource{}
	}

	assetDirFn := func(_ string) ([]string, error) {
		var ss []string
		for s := range m {
			ss = append(ss, s)
		}

		return ss, nil
	}

	assetFn := func(key string) ([]byte, error) {
		if d, ok := m[key]; ok {
			return d, nil
		}

		return nil, fmt.Errorf("cannot find data %q", key)
	}

	return &migrate.AssetMigrationSource{
		Asset:    assetFn,
		AssetDir: assetDirFn,
		Dir:      "",
	}
}
