package store_test

import (
	"context"
	"database/sql"
	"strings"
	"testing"
	"time"

	. "github.com/Tvister7/aero/internal/store"
)

func TestStudioStartsWithGuideAndPricing(t *testing.T) {
	db := openTest(t)
	studio, err := db.Studio(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(studio.Guide, "Как нас найти") {
		t.Fatalf("guide %s", studio.Guide)
	}
	if !strings.Contains(studio.Pricing, `"personal"`) || !strings.Contains(studio.Pricing, `"split":130`) {
		t.Fatalf("pricing %s", studio.Pricing)
	}
	saved, err := db.PutStudio(context.Background(), Studio{
		Address: "адрес", Phone: "+995", Guide: "", Pricing: `{"kids":{"8":90}}`,
	})
	if err != nil {
		t.Fatal(err)
	}
	if saved.Guide != "" || saved.Pricing != `{"kids":{"8":90}}` {
		t.Fatalf("saved %+v", saved)
	}
	again, err := db.Studio(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if again.Guide != "" {
		t.Fatalf("cleared guide came back %s", again.Guide)
	}
}

func TestPricingUpgradesTextAndRejectsFraction(t *testing.T) {
	db := openTest(t)
	ctx := context.Background()
	sqlDB, err := sql.Open("pgx", db.DSN())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })
	var poolSchema, sqlSchema string
	if err := db.QueryRow(ctx, `select current_schema()`).Scan(&poolSchema); err != nil {
		t.Fatal(err)
	}
	if err := sqlDB.QueryRowContext(ctx, `select current_schema()`).Scan(&sqlSchema); err != nil {
		t.Fatal(err)
	}
	if poolSchema != sqlSchema {
		t.Fatalf("schema %s vs %s", poolSchema, sqlSchema)
	}
	for _, stmt := range []string{
		`alter table studio drop constraint if exists studio_pricing_check`,
		`alter table studio alter column pricing drop default`,
		`alter table studio alter column pricing type text using pricing::text`,
		`update studio set pricing = ''`,
		`alter table studio alter column pricing set default ''`,
		`alter table studio alter column pricing set not null`,
	} {
		if _, err := sqlDB.ExecContext(ctx, stmt); err != nil {
			t.Fatal(stmt, err)
		}
	}
	again, err := Open(db.DSN())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = again.Close() })
	studio, err := again.Studio(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(studio.Pricing, `"split":130`) {
		t.Fatalf("pricing %s", studio.Pricing)
	}
	if _, err := db.PutStudio(ctx, Studio{Pricing: `{"base":{"1":35.5}}`}); err == nil {
		t.Fatal("fraction accepted")
	}
	if _, err := sqlDB.ExecContext(ctx, `update studio set pricing = $1::jsonb`, `{"base":{"1":35.5}}`); err == nil {
		t.Fatal("fraction stored")
	}
	if _, err := sqlDB.ExecContext(ctx, `update studio set pricing = $1::jsonb`, `{"base":{"1":35}}`); err != nil {
		t.Fatal(err)
	}
}

func TestPublicTrainerListsActiveTemplates(t *testing.T) {
	db := openTest(t)
	ctx := context.Background()
	now := time.Date(2026, 11, 21, 8, 0, 0, 0, time.UTC)
	lilia, err := db.CreateTrainer(ctx, now, "Лилия")
	if err != nil {
		t.Fatal(err)
	}
	hidden, err := db.CreateTrainer(ctx, now, "Жанна")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.PatchTrainer(ctx, hidden.ID, TrainerFields{Active: boolPtr(false)}); err != nil {
		t.Fatal(err)
	}
	about := "йога"
	phone := "+995555000111"
	if _, err := db.PatchTrainer(ctx, lilia.ID, TrainerFields{About: &about, Phone: &phone}); err != nil {
		t.Fatal(err)
	}
	kind := makeType(t, db, now, "Aero Yoga")
	weekday := 4
	start := "19:00"
	active := true
	if _, err := db.CreateTemplate(ctx, now, TemplateInput{
		TrainerID: &lilia.ID, TrainingTypeID: &kind.ID, Weekday: &weekday, StartTime: &start, Active: &active,
	}); err != nil {
		t.Fatal(err)
	}
	list, err := db.PublicTrainers(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 || list[0].Name != "Лилия" || list[0].Phone != phone || len(list[0].Classes) != 1 || list[0].Classes[0] != "Aero Yoga" {
		t.Fatalf("list %+v", list)
	}
	if _, err := db.PublicTrainer(ctx, hidden.ID); err == nil {
		t.Fatal("inactive trainer is visible")
	}
}

func boolPtr(value bool) *bool { return &value }
