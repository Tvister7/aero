package sqlite

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestImportSeed(t *testing.T) {
	db := openTest(t)
	now := time.Date(2026, 11, 18, 10, 0, 0, 0, time.UTC)
	counts, err := ImportSQL(context.Background(), db, sampleSeed, now)
	if err != nil {
		t.Fatal(err)
	}
	if counts.Trainers != 2 || counts.Types != 1 || counts.Templates != 1 {
		t.Fatalf("counts %+v", counts)
	}
	if _, err := ImportSQL(context.Background(), db, sampleSeed, now); err != nil {
		t.Fatal(err)
	}
	trainers, err := db.ListTrainers(context.Background())
	if err != nil || len(trainers) != 2 {
		t.Fatalf("trainers %+v %v", trainers, err)
	}
	types, err := db.ListTypes(context.Background())
	if err != nil || len(types) != 1 || types[0].PriceMin != "17.00" || types[0].PriceMax != "40.00" || types[0].Color != "#5c7a3a" {
		t.Fatalf("types %+v %v", types, err)
	}
	templates, err := db.ListTemplates(context.Background())
	if err != nil || len(templates) != 1 || templates[0].StartTime != "19:00" || templates[0].Weekday != 4 {
		t.Fatalf("templates %+v %v", templates, err)
	}
}

func TestImportStudioFile(t *testing.T) {
	path := filepath.Join("..", "..", "sql", "seed_data.sql")
	if _, err := os.Stat(path); err != nil {
		t.Skip("seed file is not in the tree")
	}
	db := openTest(t)
	counts, err := ImportFile(context.Background(), db, path, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if counts.Trainers != 5 || counts.Types != 6 || counts.Templates != 13 {
		t.Fatalf("counts %+v", counts)
	}
	again, err := ImportFile(context.Background(), db, path, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if again.Trainers != 5 || again.Templates != 13 {
		t.Fatalf("second import %+v", again)
	}
	trainers, err := db.ListTrainers(context.Background())
	if err != nil || len(trainers) != 5 {
		t.Fatalf("trainers %d %v", len(trainers), err)
	}
}

func TestBackup(t *testing.T) {
	db := openTest(t)
	if _, err := db.CreateTrainer(context.Background(), time.Now(), "Тая"); err != nil {
		t.Fatal(err)
	}
	if err := db.Backup(context.Background()); err != nil {
		t.Fatal(err)
	}
	copy, err := Open(db.Path() + ".bak")
	if err != nil {
		t.Fatal(err)
	}
	defer copy.Close()
	trainers, err := copy.ListTrainers(context.Background())
	if err != nil || len(trainers) != 1 || trainers[0].Name != "Тая" {
		t.Fatalf("backup %+v %v", trainers, err)
	}
}

func openTest(t *testing.T) *DB {
	t.Helper()
	db, err := Open(filepath.Join(t.TempDir(), "aero.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

const sampleSeed = `
insert into public.trainers (name, active)
values
  ('Жанна', true),
  ('Лилия', true)
on conflict (name) do update set active = excluded.active;

insert into public.training_types (title, category, duration_minutes, capacity, price_min, price_max, active)
values
  ('Aero Yoga', 'Занятия в гамаках', 60, 6, 17, 40, true)
on conflict (title) do update set category = excluded.category;

insert into public.class_templates (trainer_id, title, weekday, start_time, duration_minutes, capacity, active)
select t.id from (
  values
    ('Лилия', 'Aero Yoga', 4, '19:00:00', 60, 6)
) as v
on conflict (trainer_id, title, weekday, start_time) do update set capacity = excluded.capacity;
`
