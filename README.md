# sqlmig

    go get -u github.com/codemodus/sqlmig

## Usage

```go
type QueryingMigrator
type Regularizer
type Result
    func (r *Result) Err() error
    func (r *Result) String() string
    func (r *Result) Total() int
type Results
    func (rs Results) Errs() []error
    func (rs Results) ErrsErr() error
    func (rs Results) HasError() bool
    func (rs Results) String() string
    func (rs Results) Total() int
type SQLMig
    func New(db *sql.DB, driver string) (*SQLMig, error)
    func (m *SQLMig) AddQueryingMigs(qms ...QueryingMigrator)
    func (m *SQLMig) AddRegularizers(rs ...Regularizer)
    func (m *SQLMig) Migrate() Results
    func (m *SQLMig) Regularize(ctx context.Context) error
    func (m *SQLMig) RollBack() Results
```

```go
type QueryingMigrator interface {
    MigrationName() string
    MigrationIDs(_ string) ([]string, error)
    MigrationData(id string) ([]byte, error)
}

type Regularizer interface {
    Regularize(context.Context) error
}
```
