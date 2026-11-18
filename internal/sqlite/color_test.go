package sqlite

import (
	"context"
	"testing"
	"time"
)

func TestTypeColorStaysWithTheType(t *testing.T) {
	db := openTest(t)
	ctx := context.Background()
	now := time.Date(2026, 11, 18, 12, 0, 0, 0, time.UTC)
	yoga := makeType(t, db, now, "Aero Yoga")
	if yoga.Color != "#5c7a3a" {
		t.Fatalf("yoga color %s", yoga.Color)
	}
	pilates := makeType(t, db, now, "Пилатес")
	if pilates.Color == "" || pilates.Color == yoga.Color {
		t.Fatalf("pilates color %s", pilates.Color)
	}
	renamed := "Аэройога"
	patched, err := db.PatchType(ctx, yoga.ID, TypeInput{Title: &renamed})
	if err != nil {
		t.Fatal(err)
	}
	if patched.Color != yoga.Color {
		t.Fatalf("rename changed color %s", patched.Color)
	}
	third := makeType(t, db, now, "Барре")
	if third.Color == yoga.Color || third.Color == pilates.Color {
		t.Fatalf("third color collided %s", third.Color)
	}
	listed, err := db.ListTypes(ctx)
	if err != nil {
		t.Fatal(err)
	}
	colors := map[string]string{}
	for _, item := range listed {
		colors[item.Title] = item.Color
	}
	if colors["Аэройога"] != yoga.Color || colors["Пилатес"] != pilates.Color {
		t.Fatalf("colors shifted %+v", colors)
	}
	trainer, err := db.CreateTrainer(ctx, now, "Лилия")
	if err != nil {
		t.Fatal(err)
	}
	class, err := db.CreateClass(ctx, now, ClassInput{
		TrainerID: trainer.ID, TypeID: yoga.ID, StartsAt: "2026-11-19T10:00:00+04:00",
	})
	if err != nil {
		t.Fatal(err)
	}
	if class.Color != yoga.Color {
		t.Fatalf("class color %s", class.Color)
	}
}

func makeType(t *testing.T, db *DB, now time.Time, title string) TrainingType {
	t.Helper()
	item, err := db.CreateType(context.Background(), now, TypeInput{
		Title: &title, DurationMinutes: intPtr(60), Capacity: intPtr(6), PriceMin: strPtr("17"), PriceMax: strPtr("40"),
	})
	if err != nil {
		t.Fatal(err)
	}
	return item
}

func intPtr(value int) *int       { return &value }
func strPtr(value string) *string { return &value }
