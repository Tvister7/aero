package store_test

import (
	"context"
	"testing"

	. "github.com/Tvister7/aero/internal/store"
)

func TestNativeColumnTypes(t *testing.T) {
	db := openTest(t)
	checks := []struct {
		table, column, dataType, udt string
	}{
		{"trainers", "id", "uuid", "uuid"},
		{"trainers", "active", "boolean", "bool"},
		{"trainers", "created_at", "timestamp with time zone", "timestamptz"},
		{"classes", "status", "USER-DEFINED", "class_status"},
		{"class_templates", "start_time", "time without time zone", "time"},
		{"trainer_absences", "starts_on", "date", "date"},
		{"studio", "pricing", "jsonb", "jsonb"},
	}
	for _, item := range checks {
		var dataType, udt string
		err := db.QueryRow(context.Background(), `
			select data_type, udt_name from information_schema.columns
			where table_schema = current_schema() and table_name = ? and column_name = ?`,
			item.table, item.column).Scan(&dataType, &udt)
		if err != nil {
			t.Fatal(item.table, item.column, err)
		}
		if dataType != item.dataType || udt != item.udt {
			t.Fatalf("%s.%s is %s/%s", item.table, item.column, dataType, udt)
		}
	}
	again, err := Open(db.DSN())
	if err != nil {
		t.Fatal(err)
	}
	again.Close()
}
