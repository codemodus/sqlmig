# sqlmig

    go get -u github.com/codemodus/sqlmig

## Usage

```go
type MigrationGetter
type Migrator
type Regularizer
type Result
    func (r *Result) Directory() string
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
    func (m *SQLMig) AddMigrations(srcs ...MigrationGetter)
    func (m *SQLMig) AddRegularizations(regs ...Regularizer)
    func (m *SQLMig) Migrate() Results
    func (m *SQLMig) Regularize(ctx context.Context) error
    func (m *SQLMig) RollBack() Results
    func (m *SQLMig) RunMigrations(src MigrationGetter, up bool) *Result
```

```go
type MigrationGetter interface {
    MigrationName() string
    MigrationAssetDirectory() string
    MigrationAssetNames(dir string) (filenames []string, err error)
    MigrationAsset(filename string) (data []byte, err error)
}

type Migrator interface {
    MigrationGetter
    Regularizer
}

type Regularizer interface {
    Regularize(context.Context) error
}
```
