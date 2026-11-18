package sqlite

import (
	"context"
	"crypto/rand"
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Tvister7/aero/internal/domain"
	_ "modernc.org/sqlite"
)

type DB struct {
	sql  *sql.DB
	path string
	ids  func() string
}

func Open(path string) (*DB, error) {
	if path == "" || strings.Contains(path, "?") {
		return nil, fmt.Errorf("DATA_PATH is empty or contains '?'")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}
	dsn := "file:" + path + "?" + url.Values{
		"_pragma": {"foreign_keys(1)", "journal_mode(WAL)", "busy_timeout(5000)"},
	}.Encode()
	sqlDB, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}
	sqlDB.SetMaxOpenConns(1)
	sqlDB.SetMaxIdleConns(1)
	db := &DB{sql: sqlDB, path: path, ids: newID}
	if err := db.migrate(context.Background()); err != nil {
		sqlDB.Close()
		return nil, err
	}
	var foreignKeys int
	if err := sqlDB.QueryRow("pragma foreign_keys").Scan(&foreignKeys); err != nil {
		sqlDB.Close()
		return nil, err
	}
	if foreignKeys != 1 {
		sqlDB.Close()
		return nil, fmt.Errorf("foreign keys are off")
	}
	return db, nil
}

func (d *DB) Close() error { return d.sql.Close() }

func (d *DB) Path() string { return d.path }

func (d *DB) migrate(ctx context.Context) error {
	for _, stmt := range strings.Split(schemaSQL, ";") {
		stmt = strings.TrimSpace(stmt)
		if stmt == "" {
			continue
		}
		if _, err := d.sql.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("schema: %w", err)
		}
	}
	return nil
}

func (d *DB) write(ctx context.Context, fn func(*sql.Tx) error) error {
	tx, err := d.sql.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	if err = fn(tx); err != nil {
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

func newID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		panic(err)
	}
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
}

func utc(t time.Time) string { return t.UTC().Format(time.RFC3339) }

func parseTime(value string) (time.Time, error) { return time.Parse(time.RFC3339, value) }

func nullable(value string) any {
	if value == "" {
		return nil
	}
	return value
}

func bit(value bool) int {
	if value {
		return 1
	}
	return 0
}
