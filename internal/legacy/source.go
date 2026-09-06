// Package legacy implements a read-only importer for Lsky Pro (open source,
// https://github.com/lsky-org/lsky-pro) databases. Lsky Pro open source ships
// with MySQL 5.7+, PostgreSQL 9.6+, SQLite 3.8.8+ and SQL Server 2017+
// drivers; all four are supported as import sources. The source database is
// only ever SELECTed: MySQL sessions are forced read-only, SQLite files are
// opened with mode=ro, and PostgreSQL/SQL Server reads run inside read-only
// transactions.
package legacy

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	mysqldriver "github.com/go-sql-driver/mysql"

	_ "github.com/glebarez/go-sqlite"   // registers "sqlite" (read-only capable)
	_ "github.com/jackc/pgx/v5/stdlib"  // registers "pgx"
	_ "github.com/microsoft/go-mssqldb" // registers "sqlserver"
)

// Source describes the connection parameters of a Lsky Pro database.
// Type selects the access method:
//   - "mysql" (default): connect to a running MySQL/MariaDB server;
//   - "postgres": connect to a PostgreSQL server;
//   - "sqlserver": connect to a SQL Server instance;
//   - "sqlite": open the Lsky Pro SQLite database file read-only, located in
//     SQLiteDir (a directory containing the database file, or the file itself).
type Source struct {
	Type        string `json:"type"`
	Host        string `json:"host"`
	Port        string `json:"port"`
	Database    string `json:"database"`
	Username    string `json:"username"`
	Password    string `json:"password"`
	TablePrefix string `json:"tablePrefix"`
	// SQLiteDir 源数据库目录：SQLite 数据库文件所在目录（或直接给文件路径）。
	SQLiteDir string `json:"sqliteDir"`
}

const (
	SourceTypeMySQL     = "mysql"
	SourceTypeSQLite    = "sqlite"
	SourceTypePostgres  = "postgres"
	SourceTypeSQLServer = "sqlserver"
)

// NormalizeType returns the validated source type (mysql by default).
func NormalizeType(t string) string {
	switch strings.ToLower(strings.TrimSpace(t)) {
	case SourceTypeSQLite, "sqlite3":
		return SourceTypeSQLite
	case SourceTypePostgres, "postgresql", "pgsql", "pg":
		return SourceTypePostgres
	case SourceTypeSQLServer, "mssql", "ms sql", "sql server", "ms sqlserver":
		return SourceTypeSQLServer
	default:
		return SourceTypeMySQL
	}
}

// Sanitize trims whitespace around the fields.
func (s Source) Sanitize() Source {
	return Source{
		Type:        NormalizeType(s.Type),
		Host:        strings.TrimSpace(s.Host),
		Port:        strings.TrimSpace(s.Port),
		Database:    strings.TrimSpace(s.Database),
		Username:    strings.TrimSpace(s.Username),
		Password:    s.Password,
		TablePrefix: strings.TrimSpace(s.TablePrefix),
		SQLiteDir:   strings.TrimSpace(s.SQLiteDir),
	}
}

func (s Source) validate() error {
	switch s.Type {
	case SourceTypeSQLite:
		if s.SQLiteDir == "" {
			return fmt.Errorf("请填写源数据库目录（SQLite 数据库文件所在目录）")
		}
	default:
		if s.Host == "" || s.Port == "" || s.Database == "" || s.Username == "" {
			return fmt.Errorf("源数据库连接信息不完整（需要主机、端口、数据库名、用户名）")
		}
	}
	return nil
}

// quote quotes an identifier for the source dialect.
func (s Source) quote(ident string) string {
	switch s.Type {
	case SourceTypePostgres:
		return `"` + strings.ReplaceAll(ident, `"`, `""`) + `"`
	case SourceTypeSQLServer:
		return "[" + strings.ReplaceAll(ident, "]", "]]") + "]"
	default:
		// MySQL 与 SQLite 都接受反引号包裹的标识符。
		return "`" + strings.ReplaceAll(ident, "`", "``") + "`"
	}
}

// firstRowQuery builds a single-row SELECT; SQL Server uses TOP while the
// other dialects use LIMIT.
func (s Source) firstRowQuery(selectExpr, fromWhere string) string {
	if s.Type == SourceTypeSQLServer {
		return fmt.Sprintf("SELECT TOP (1) %s FROM %s", selectExpr, fromWhere)
	}
	return fmt.Sprintf("SELECT %s FROM %s LIMIT 1", selectExpr, fromWhere)
}

// ph returns the n-th (1-based) placeholder for the source dialect.
func (s Source) ph(n int) string {
	switch s.Type {
	case SourceTypePostgres:
		return fmt.Sprintf("$%d", n)
	case SourceTypeSQLServer:
		return fmt.Sprintf("@p%d", n)
	default:
		return "?"
	}
}

// pageQuery builds the paged SELECT used by the batch importer.
func (s Source) pageQuery(table, keyColumn string, columns []string, limit int) string {
	cols := make([]string, 0, len(columns))
	for _, c := range columns {
		cols = append(cols, s.quote(c))
	}
	key := s.quote(keyColumn)
	switch s.Type {
	case SourceTypeSQLServer:
		return fmt.Sprintf(
			"SELECT TOP (%d) %s FROM %s WHERE %s > %s ORDER BY %s ASC",
			limit, strings.Join(cols, ", "), s.quote(table), key, s.ph(1), key,
		)
	default:
		return fmt.Sprintf(
			"SELECT %s FROM %s WHERE %s > %s ORDER BY %s ASC LIMIT %d",
			strings.Join(cols, ", "), s.quote(table), key, s.ph(1), key, limit,
		)
	}
}

// dsn builds the connection string for network source dialects.
func (s Source) dsn(enforceReadOnly bool) string {
	switch s.Type {
	case SourceTypePostgres:
		dsn := fmt.Sprintf(
			"postgres://%s:%s@%s:%s/%s?sslmode=disable&application_name=skyimage-import",
			url.QueryEscape(s.Username),
			url.QueryEscape(s.Password),
			s.Host,
			s.Port,
			url.QueryEscape(s.Database),
		)
		if enforceReadOnly {
			// 服务器会话级默认只读：任何写入都会被 PostgreSQL 拒绝。
			dsn += "&options=" + url.QueryEscape("-c default_transaction_read_only=on")
		}
		return dsn
	case SourceTypeSQLServer:
		return fmt.Sprintf(
			"sqlserver://%s:%s@%s:%s?database=%s&encrypt=disable&app+name=skyimage-import",
			url.QueryEscape(s.Username),
			url.QueryEscape(s.Password),
			s.Host,
			s.Port,
			url.QueryEscape(s.Database),
		)
	default:
		dsn := fmt.Sprintf(
			"%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local&timeout=10s&readTimeout=120s&writeTimeout=120s&interpolateParams=true",
			url.QueryEscape(s.Username),
			url.QueryEscape(s.Password),
			s.Host,
			s.Port,
			url.QueryEscape(s.Database),
		)
		if enforceReadOnly {
			dsn += "&transaction_read_only=1"
		}
		return dsn
	}
}

// resolveSQLiteFile turns the 源数据库目录 into the database file path.
// A directory is scanned for *.sqlite / *.sqlite3 / *.db files; when several
// exist, database.sqlite / lsky.sqlite take precedence.
func resolveSQLiteFile(dir string) (string, error) {
	info, err := os.Stat(dir)
	if err != nil {
		return "", fmt.Errorf("源数据库目录不可访问: %w", err)
	}
	if !info.IsDir() {
		return dir, nil
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", fmt.Errorf("读取源数据库目录失败: %w", err)
	}
	var candidates []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		ext := strings.ToLower(filepath.Ext(e.Name()))
		if ext == ".sqlite" || ext == ".sqlite3" || ext == ".db" {
			candidates = append(candidates, e.Name())
		}
	}
	if len(candidates) == 0 {
		return "", fmt.Errorf("源数据库目录中未找到 SQLite 数据库文件（.sqlite/.sqlite3/.db）")
	}
	for _, preferred := range []string{"database.sqlite", "lsky.sqlite", "lsky-pro.sqlite"} {
		for _, name := range candidates {
			if strings.EqualFold(name, preferred) {
				return filepath.Join(dir, name), nil
			}
		}
	}
	if len(candidates) > 1 {
		return "", fmt.Errorf("源数据库目录中存在多个数据库文件，请只保留一个: %s", strings.Join(candidates, ", "))
	}
	return filepath.Join(dir, candidates[0]), nil
}

// OpenSource connects to the Lsky Pro database with read-only access.
//   - MySQL: every pooled connection runs `SET transaction_read_only = 1`
//     (old servers without the variable fall back to a plain session; the
//     importer still only ever runs SELECTs);
//   - SQLite: the file is opened with mode=ro;
//   - PostgreSQL / SQL Server: reads run inside read-only transactions
//     (see readOnlyRunner), opened eagerly here with the read-only default
//     applied at session level for PostgreSQL.
func OpenSource(ctx context.Context, src Source) (*sql.DB, bool, error) {
	src = src.Sanitize()
	if err := src.validate(); err != nil {
		return nil, false, err
	}

	switch src.Type {
	case SourceTypeSQLite:
		return openSQLite(ctx, src)
	case SourceTypePostgres:
		return openNetwork(ctx, src, "pgx")
	case SourceTypeSQLServer:
		return openNetwork(ctx, src, "sqlserver")
	default:
		return openMySQL(ctx, src)
	}
}

func openSQLite(ctx context.Context, src Source) (*sql.DB, bool, error) {
	file, err := resolveSQLiteFile(src.SQLiteDir)
	if err != nil {
		return nil, false, err
	}
	dsn := "file:" + filepath.ToSlash(file) + "?mode=ro"
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, false, err
	}
	db.SetMaxOpenConns(4)
	db.SetMaxIdleConns(2)
	pingCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	if err := db.PingContext(pingCtx); err != nil {
		_ = db.Close()
		return nil, false, fmt.Errorf("连接源数据库失败: %w", err)
	}
	// mode=ro 在驱动层面拒绝任何写入。
	return db, true, nil
}

func openNetwork(ctx context.Context, src Source, driverName string) (*sql.DB, bool, error) {
	db, err := sql.Open(driverName, src.dsn(true))
	if err != nil {
		return nil, false, err
	}
	db.SetMaxOpenConns(4)
	db.SetMaxIdleConns(2)
	db.SetConnMaxLifetime(10 * time.Minute)

	pingCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	if err := db.PingContext(pingCtx); err != nil {
		_ = db.Close()
		if src.Type == SourceTypePostgres && strings.Contains(err.Error(), "default_transaction_read_only") {
			// 旧版本服务器不认识该会话参数时降级重连（仍只执行 SELECT）。
			db2, err2 := sql.Open(driverName, src.dsn(false))
			if err2 != nil {
				return nil, false, err2
			}
			if err := db2.PingContext(pingCtx); err != nil {
				_ = db2.Close()
				return nil, false, fmt.Errorf("连接源数据库失败: %w", err)
			}
			return db2, false, nil
		}
		return nil, false, fmt.Errorf("连接源数据库失败: %w", err)
	}
	// PostgreSQL 会话已在服务器端强制只读；SQL Server 由 readOnlyRunner
	// 的只读事务保证。
	return db, true, nil
}

func openMySQL(ctx context.Context, src Source) (*sql.DB, bool, error) {
	cfg, err := mysqldriver.ParseDSN(src.dsn(true))
	if err != nil {
		return nil, false, fmt.Errorf("解析源数据库 DSN: %w", err)
	}

	db, err := sql.Open("mysql", cfg.FormatDSN())
	if err != nil {
		return nil, false, err
	}
	db.SetMaxOpenConns(4)
	db.SetMaxIdleConns(2)
	db.SetConnMaxLifetime(10 * time.Minute)

	pingCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	if err := db.PingContext(pingCtx); err != nil {
		_ = db.Close()
		// Retry without the read-only session variable for servers that do
		// not know it (the importer still only ever runs SELECTs).
		if strings.Contains(err.Error(), "Unknown system variable") {
			db2, err2 := openPlain(ctx, src, pingCtx)
			return db2, false, err2
		}
		return nil, false, fmt.Errorf("连接源数据库失败: %w", err)
	}

	if err := ensureSessionReadOnly(db); err != nil {
		_ = db.Close()
		db2, err2 := openPlain(ctx, src, pingCtx)
		return db2, false, err2
	}
	return db, true, nil
}

func openPlain(ctx context.Context, src Source, pingCtx context.Context) (*sql.DB, error) {
	cfg, err := mysqldriver.ParseDSN(src.dsn(false))
	if err != nil {
		return nil, fmt.Errorf("解析源数据库 DSN: %w", err)
	}
	db, err := sql.Open("mysql", cfg.FormatDSN())
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(4)
	db.SetMaxIdleConns(2)
	db.SetConnMaxLifetime(10 * time.Minute)
	if err := db.PingContext(pingCtx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("连接源数据库失败: %w", err)
	}
	return db, nil
}

// ensureSessionReadOnly double-checks that new connections really start in
// read-only transaction mode (defense in depth alongside SELECT-only code).
func ensureSessionReadOnly(db *sql.DB) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	var mode string
	if err := db.QueryRowContext(ctx, "SELECT @@session.transaction_read_only").Scan(&mode); err != nil {
		return err
	}
	if mode != "1" {
		return fmt.Errorf("源数据库会话未被置于只读模式")
	}
	return nil
}

// queryRunner abstracts *sql.DB and *sql.Tx for the import reads.
type queryRunner interface {
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

// readOnlyRunner wraps the source for reading. PostgreSQL / SQL Server reads
// run inside a read-only transaction; MySQL / SQLite rely on the session or
// file-level read-only mode established by OpenSource.
func (s Source) readOnlyRunner(ctx context.Context, db *sql.DB, openReadOnly bool) (queryRunner, bool, func()) {
	switch s.Type {
	case SourceTypePostgres, SourceTypeSQLServer:
		tx, err := db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
		if err == nil {
			return tx, true, func() { _ = tx.Rollback() }
		}
		// 只读事务不可用时退化为普通会话（导入代码仍只执行 SELECT）。
		return db, false, func() {}
	default:
		return db, openReadOnly, func() {}
	}
}

func (s Source) table(name string) string {
	prefix := strings.TrimSpace(s.TablePrefix)
	if prefix != "" && !strings.HasSuffix(prefix, "_") {
		prefix += "_"
	}
	return prefix + name
}

// lskyTables lists the tables required by the importer. personal_access_tokens
// and other framework tables are intentionally not imported.
var lskyTables = []string{"groups", "users", "albums", "strategies", "images", "configs"}

// Probe checks connectivity and reports what is available in the source
// database. It runs SELECT statements only.
func Probe(ctx context.Context, db *sql.DB, src Source, openReadOnly bool) (*ProbeResult, error) {
	src = src.Sanitize()
	runner, readOnly, cleanup := src.readOnlyRunner(ctx, db, openReadOnly)
	defer cleanup()

	res := &ProbeResult{Counts: map[string]int64{}}

	for _, name := range lskyTables {
		exists, err := tableExists(ctx, runner, src, name)
		if err != nil {
			return nil, err
		}
		if !exists {
			res.MissingTables = append(res.MissingTables, src.table(name))
			continue
		}
		var count int64
		if err := runner.QueryRowContext(ctx, "SELECT COUNT(*) FROM "+src.quote(src.table(name))).Scan(&count); err != nil {
			return nil, err
		}
		res.Counts[name] = count
	}

	if len(res.MissingTables) == 0 {
		res.AppName, res.AppVersion = probeAppVersion(ctx, runner, src)
	}
	res.ReadOnlyReady = readOnly
	return res, nil
}

// tableExists checks for one table in a dialect-agnostic way.
func tableExists(ctx context.Context, runner queryRunner, src Source, name string) (bool, error) {
	if src.Type == SourceTypeSQLite {
		var exists int
		query := "SELECT COUNT(*) FROM sqlite_master WHERE type = 'table' AND name = " + src.ph(1)
		if err := runner.QueryRowContext(ctx, query, src.table(name)).Scan(&exists); err != nil {
			return false, err
		}
		return exists > 0, nil
	}
	var exists int
	var query string
	switch src.Type {
	case SourceTypePostgres:
		query = "SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = current_schema() AND table_name = " + src.ph(1)
	case SourceTypeSQLServer:
		query = "SELECT COUNT(*) FROM INFORMATION_SCHEMA.TABLES WHERE TABLE_SCHEMA = SCHEMA_NAME() AND TABLE_NAME = " + src.ph(1)
	default:
		query = "SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = " + src.ph(1) + " AND table_name = " + src.ph(2)
	}
	args := []any{src.table(name)}
	if src.Type == SourceTypeMySQL {
		args = append([]any{src.Database}, args...)
	}
	if err := runner.QueryRowContext(ctx, query, args...).Scan(&exists); err != nil {
		return false, err
	}
	return exists > 0, nil
}

func probeAppVersion(ctx context.Context, runner queryRunner, src Source) (string, string) {
	var value string
	query := src.firstRowQuery(src.quote("value"),
		src.quote(src.table("configs"))+" WHERE "+src.quote("name")+" = 'app_version'")
	if err := runner.QueryRowContext(ctx, query).Scan(&value); err != nil {
		return "", ""
	}
	var name string
	queryName := src.firstRowQuery(src.quote("value"),
		src.quote(src.table("configs"))+" WHERE "+src.quote("name")+" = 'app_name'")
	_ = runner.QueryRowContext(ctx, queryName).Scan(&name)
	return name, value
}

// ProbeResult summarises a read-only inspection of the source database.
type ProbeResult struct {
	AppName       string           `json:"appName"`
	AppVersion    string           `json:"appVersion"`
	Counts        map[string]int64 `json:"counts"`
	MissingTables []string         `json:"missingTables,omitempty"`
	ReadOnlyReady bool             `json:"readOnlyReady"`
}

// OK reports whether the source looks like a Lsky Pro database.
func (p *ProbeResult) OK() bool {
	return len(p.MissingTables) == 0
}
