package sqlite

import (
	"context"
	"os"
	"strings"
)

func (d *DB) Backup(ctx context.Context) error {
	tmp := d.path + ".bak.tmp"
	if err := os.Remove(tmp); err != nil && !os.IsNotExist(err) {
		return err
	}
	quoted := strings.ReplaceAll(tmp, "'", "''")
	if _, err := d.sql.ExecContext(ctx, `vacuum into '`+quoted+`'`); err != nil {
		return err
	}
	return os.Rename(tmp, d.path+".bak")
}
