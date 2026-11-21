package store

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

func (d *DB) Backup(ctx context.Context) error {
	dir := os.Getenv("BACKUP_DIR")
	if dir == "" {
		dir = "/backups"
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	name := filepath.Join(dir, "aero-"+time.Now().UTC().Format("2006-01-02")+".sql")
	cmd := exec.CommandContext(ctx, "pg_dump", "--dbname", dumpDSN(d.dsn), "--schema", d.schema, "--file", name, "--no-owner", "--no-acl")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("pg_dump: %w: %s", err, out)
	}
	return nil
}

func dumpDSN(dsn string) string {
	u, err := url.Parse(dsn)
	if err != nil {
		return dsn
	}
	q := u.Query()
	q.Del("options")
	u.RawQuery = q.Encode()
	return u.String()
}
