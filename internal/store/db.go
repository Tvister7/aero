package store

import (
	"context"
	"crypto/rand"
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/Tvister7/aero/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
)

type DB struct {
	sql       querier
	raw       *sql.DB
	dsn       string
	schema    string
	ids       func() string
	Exclusive bool
}

func Open(dsn string) (*DB, error) {
	if strings.TrimSpace(dsn) == "" {
		return nil, fmt.Errorf("DATABASE_URL is empty")
	}
	sqlDB, err := openSQL(dsn)
	if err != nil {
		return nil, err
	}
	sqlDB.SetMaxOpenConns(10)
	sqlDB.SetMaxIdleConns(10)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := sqlDB.PingContext(ctx); err != nil {
		sqlDB.Close()
		return nil, err
	}
	db := &DB{sql: dbBinder{sqlDB}, raw: sqlDB, dsn: dsn, schema: schemaOf(dsn), ids: newID, Exclusive: true}
	if err := db.migrate(context.Background()); err != nil {
		sqlDB.Close()
		return nil, err
	}
	return db, nil
}

func openSQL(dsn string) (*sql.DB, error) {
	cfg, err := pgx.ParseConfig(dsn)
	if err != nil {
		return nil, err
	}
	// Простой протокол передаёт аргументы бестиповыми литералами, и Postgres
	// сам приводит строку к uuid, enum, date и time.
	cfg.DefaultQueryExecMode = pgx.QueryExecModeSimpleProtocol
	return stdlib.OpenDB(*cfg), nil
}

func OpenTemp() (*DB, func() error, error) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = "postgres://aero:aero@127.0.0.1:5432/aero?sslmode=disable"
	}
	schema := "t_" + strings.ReplaceAll(newID(), "-", "")
	admin, err := openSQL(dsn)
	if err != nil {
		return nil, nil, err
	}
	if _, err := admin.Exec("create schema " + schema); err != nil {
		admin.Close()
		return nil, nil, err
	}
	admin.Close()
	testDSN, err := withSearchPath(dsn, schema)
	if err != nil {
		_ = dropSchema(dsn, schema)
		return nil, nil, err
	}
	db, err := Open(testDSN)
	if err != nil {
		_ = dropSchema(dsn, schema)
		return nil, nil, err
	}
	var current string
	if err := db.raw.QueryRow("select current_schema()").Scan(&current); err != nil || current != schema {
		db.Close()
		_ = dropSchema(dsn, schema)
		if err != nil {
			return nil, nil, err
		}
		return nil, nil, fmt.Errorf("search_path is %s", current)
	}
	cleanup := func() error {
		closeErr := db.Close()
		dropErr := dropSchema(dsn, schema)
		if closeErr != nil {
			return closeErr
		}
		return dropErr
	}
	return db, cleanup, nil
}

func (d *DB) Close() error { return d.raw.Close() }

func (d *DB) DSN() string { return d.dsn }

func (d *DB) QueryRow(ctx context.Context, query string, args ...any) *sql.Row {
	return d.sql.QueryRowContext(ctx, query, args...)
}

func (d *DB) migrate(ctx context.Context) error {
	for _, stmt := range strings.Split(schemaSQL, ";") {
		stmt = stripSQLComments(stmt)
		if stmt == "" {
			continue
		}
		if name, ok := enumName(stmt); ok {
			exists, err := d.enumExists(ctx, name)
			if err != nil {
				return err
			}
			if exists {
				continue
			}
		}
		if _, err := d.sql.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("schema: %w", err)
		}
	}
	return d.write(ctx, func(tx querier) error {
		return seedStudioDefaults(ctx, tx)
	})
}

func stripSQLComments(stmt string) string {
	var b strings.Builder
	for _, line := range strings.Split(stmt, "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "--") {
			continue
		}
		b.WriteString(line)
		b.WriteByte('\n')
	}
	return strings.TrimSpace(b.String())
}

func enumName(stmt string) (string, bool) {
	fields := strings.Fields(stmt)
	if len(fields) < 4 || !strings.EqualFold(fields[0], "create") || !strings.EqualFold(fields[1], "type") || !strings.EqualFold(fields[3], "as") {
		return "", false
	}
	return fields[2], true
}

func (d *DB) enumExists(ctx context.Context, name string) (bool, error) {
	var found bool
	err := d.sql.QueryRowContext(ctx, `
		select exists (
			select 1 from pg_type t
			join pg_namespace n on n.oid = t.typnamespace
			where n.nspname = current_schema() and t.typname = ?
		)`, name).Scan(&found)
	return found, err
}

func (d *DB) write(ctx context.Context, fn func(querier) error) error {
	tx, err := d.raw.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	if err = fn(txBinder{tx}); err != nil {
		_ = tx.Rollback()
		return mapWriteErr(err)
	}
	return tx.Commit()
}

func mapWriteErr(err error) error {
	var domainErr *domain.Error
	if errors.As(err, &domainErr) {
		return err
	}
	if strings.Contains(strings.ToLower(err.Error()), "unique") {
		return domain.Fail(domain.Validation)
	}
	return err
}

type dbBinder struct{ *sql.DB }

func (b dbBinder) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	return b.DB.ExecContext(ctx, Rebind(query), bindArgs(args)...)
}

func (b dbBinder) QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	return b.DB.QueryContext(ctx, Rebind(query), bindArgs(args)...)
}

func (b dbBinder) QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row {
	return b.DB.QueryRowContext(ctx, Rebind(query), bindArgs(args)...)
}

type txBinder struct{ *sql.Tx }

func (b txBinder) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	return b.Tx.ExecContext(ctx, Rebind(query), bindArgs(args)...)
}

func (b txBinder) QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	return b.Tx.QueryContext(ctx, Rebind(query), bindArgs(args)...)
}

func (b txBinder) QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row {
	return b.Tx.QueryRowContext(ctx, Rebind(query), bindArgs(args)...)
}

func bindArgs(args []any) []any {
	if len(args) == 0 {
		return nil
	}
	out := make([]any, len(args))
	for i, arg := range args {
		switch value := arg.(type) {
		case int:
			out[i] = int64(value)
		case int8:
			out[i] = int64(value)
		case int16:
			out[i] = int64(value)
		case int32:
			out[i] = int64(value)
		default:
			out[i] = arg
		}
	}
	return out
}

// Rebind меняет плейсхолдеры ? на $n и не трогает вопросительный знак внутри кавычек.
func Rebind(query string) string {
	var b strings.Builder
	b.Grow(len(query) + 8)
	n := 1
	inQuote := false
	for i := 0; i < len(query); i++ {
		char := query[i]
		if inQuote {
			b.WriteByte(char)
			if char == '\'' {
				if i+1 < len(query) && query[i+1] == '\'' {
					b.WriteByte('\'')
					i++
					continue
				}
				inQuote = false
			}
			continue
		}
		if char == '\'' {
			inQuote = true
			b.WriteByte(char)
			continue
		}
		if char == '?' {
			b.WriteByte('$')
			b.WriteString(strconv.Itoa(n))
			n++
			continue
		}
		b.WriteByte(char)
	}
	return b.String()
}

func newID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		panic(err)
	}
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
}

func utc(t time.Time) time.Time { return t.UTC() }

func dayText(t time.Time) string { return t.UTC().Format("2006-01-02") }

func nullable(value string) any {
	if value == "" {
		return nil
	}
	return value
}

func schemaOf(dsn string) string {
	u, err := url.Parse(dsn)
	if err != nil {
		return "public"
	}
	const key = "search_path="
	opt := u.Query().Get("options")
	i := strings.Index(opt, key)
	if i < 0 {
		return "public"
	}
	rest := opt[i+len(key):]
	if cut := strings.IndexAny(rest, " ,"); cut >= 0 {
		rest = rest[:cut]
	}
	if rest == "" {
		return "public"
	}
	return rest
}

func withSearchPath(dsn, schema string) (string, error) {
	u, err := url.Parse(dsn)
	if err != nil {
		return "", err
	}
	q := u.Query()
	q.Set("options", "-csearch_path="+schema)
	u.RawQuery = q.Encode()
	return u.String(), nil
}

func dropSchema(dsn, schema string) error {
	if !tempSchema(schema) {
		return fmt.Errorf("schema name")
	}
	db, err := openSQL(dsn)
	if err != nil {
		return err
	}
	defer db.Close()
	_, err = db.Exec("drop schema if exists " + schema + " cascade")
	return err
}

func tempSchema(name string) bool {
	if !strings.HasPrefix(name, "t_") || len(name) < 4 {
		return false
	}
	for _, char := range name[2:] {
		if (char < '0' || char > '9') && (char < 'a' || char > 'f') {
			return false
		}
	}
	return true
}
