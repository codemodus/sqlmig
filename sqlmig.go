package sqlmig

import (
	"context"
	"database/sql"
	"fmt"
	"sync"

	migrate "github.com/rubenv/sql-migrate"
)

// QueryingMigrator ...
type QueryingMigrator interface {
	MigrationName() string
	MigrationIDs(_ string) ([]string, error)
	MigrationData(id string) ([]byte, error)
}

// Regularizer ...
type Regularizer interface {
	Regularize(context.Context) error
}

// SQLMig ...
type SQLMig struct {
	*sql.DB
	drv string
	mu  sync.Mutex
	qms []QueryingMigrator
	rs  []Regularizer
}

// New ...
func New(db *sql.DB, driver string) (*SQLMig, error) {
	m := SQLMig{
		DB:  db,
		drv: driver,
	}

	return &m, nil
}

// AddQueryingMigs ...
func (m *SQLMig) AddQueryingMigs(qms ...QueryingMigrator) {
	if qms == nil || len(qms) == 0 {
		return
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	m.qms = append(m.qms, qms...)
}

func (m *SQLMig) runQueryingMig(qm QueryingMigrator, up bool) *Result {
	dir := migrate.Up
	if !up {
		dir = migrate.Down
	}

	msrc := migrate.AssetMigrationSource{
		Asset:    qm.MigrationData,
		AssetDir: qm.MigrationIDs,
		Dir:      "",
	}

	ct, err := migrate.Exec(m.DB, m.drv, &msrc, dir)

	return &Result{
		name: qm.MigrationName(),
		ct:   ct,
		err:  err,
	}
}

func (m *SQLMig) migrate(up bool) Results {
	var rs Results

	m.mu.Lock()
	defer m.mu.Unlock()

	for _, qm := range m.qms {
		r := m.runQueryingMig(qm, up)
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
