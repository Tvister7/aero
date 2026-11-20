package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/Tvister7/aero/internal/domain"
)

func TestStudioKeepsOneRoom(t *testing.T) {
	db := openTest(t)
	ctx := context.Background()
	now := time.Date(2026, 11, 19, 8, 0, 0, 0, domain.Tbilisi)
	trainer, err := db.CreateTrainer(ctx, now, "Лилия")
	if err != nil {
		t.Fatal(err)
	}
	kind := makeType(t, db, now, "Aero Yoga")
	other, err := db.CreateTrainer(ctx, now, "Жанна")
	if err != nil {
		t.Fatal(err)
	}
	start := now.Add(4 * time.Hour)
	if _, err := db.CreateClass(ctx, now, ClassInput{TrainerID: trainer.ID, TypeID: kind.ID, StartsAt: domain.FormatISO(start)}); err != nil {
		t.Fatal(err)
	}
	_, err = db.CreateClass(ctx, now, ClassInput{TrainerID: other.ID, TypeID: kind.ID, StartsAt: domain.FormatISO(start.Add(30 * time.Minute))})
	if codeOf(err) != domain.StudioBusy {
		t.Fatalf("overlap %v", err)
	}
	hold, err := db.CreateHold(ctx, now, "Съёмка", domain.FormatISO(start.Add(3*time.Hour)), nil)
	if err != nil || hold.Kind != domain.KindHold || hold.Title != "Съёмка" {
		t.Fatalf("hold %+v %v", hold, err)
	}
	client, err := db.Me(ctx, now, User{TelegramID: 1001, Username: "anna"})
	if err != nil {
		t.Fatal(err)
	}
	individualAt := start.Add(5 * time.Hour)
	individual, err := db.CreateIndividual(ctx, now, trainer.ID, client.ID, domain.FormatISO(individualAt), nil)
	if err != nil || individual.Kind != domain.KindIndividual || individual.Capacity != 1 {
		t.Fatalf("individual %+v %v", individual, err)
	}
	day := "2026-11-19"
	clientView, err := db.Schedule(ctx, day, day, false)
	if err != nil {
		t.Fatal(err)
	}
	var sawHold, sawIndividual bool
	for _, item := range clientView {
		if item.Kind == domain.KindHold {
			sawHold = item.Title == "Студия занята"
		}
		if item.Kind == domain.KindIndividual {
			sawIndividual = true
		}
	}
	if !sawHold || sawIndividual {
		t.Fatalf("client schedule %+v", clientView)
	}
	staff, err := db.Schedule(ctx, day, day, true)
	if err != nil {
		t.Fatal(err)
	}
	var staffHold bool
	for _, item := range staff {
		if item.ID == hold.ID && item.Title == "Съёмка" {
			staffHold = true
		}
	}
	if !staffHold {
		t.Fatalf("admin schedule %+v", staff)
	}
	bookings, err := db.MyBookings(ctx, now, User{TelegramID: 1001, Username: "anna"})
	if err != nil || len(bookings) != 1 || bookings[0].Class.Title != "Индивидуальное" {
		t.Fatalf("bookings %+v %v", bookings, err)
	}
	if _, err := db.CancelBooking(ctx, now, User{TelegramID: 1001}, bookings[0].BookingID); err != nil {
		t.Fatal(err)
	}
	if _, err := db.CreateClass(ctx, now, ClassInput{TrainerID: other.ID, TypeID: kind.ID, StartsAt: domain.FormatISO(individualAt)}); err != nil {
		t.Fatalf("freed %v", err)
	}
	db.Exclusive = false
	if _, err := db.CreateClass(ctx, now, ClassInput{TrainerID: trainer.ID, TypeID: kind.ID, StartsAt: domain.FormatISO(start.Add(30 * time.Minute))}); err != nil {
		t.Fatalf("several rooms %v", err)
	}
}

func TestClassKindMigration(t *testing.T) {
	path := filepath.Join(t.TempDir(), "old.db")
	raw, err := sql.Open("sqlite", "file:"+path)
	if err != nil {
		t.Fatal(err)
	}
	_, err = raw.Exec(`create table classes (
		id text primary key,
		template_id text,
		trainer_id text not null,
		training_type_id text not null,
		title text not null,
		starts_at text not null,
		ends_at text not null,
		capacity integer not null,
		price_min text not null,
		price_max text not null,
		status text not null,
		created_at text not null
	)`)
	if err != nil {
		t.Fatal(err)
	}
	_, err = raw.Exec(`insert into classes values ('c1', null, 'tr', 'ty', 'Yoga', '2026-11-19T06:00:00Z', '2026-11-19T07:00:00Z', 6, '17.00', '40.00', 'scheduled', '2026-11-01T00:00:00Z')`)
	if err != nil {
		t.Fatal(err)
	}
	raw.Close()
	db, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	class, err := loadClass(context.Background(), db.sql, "c1")
	if err != nil || class.Kind != domain.KindGroup || class.Title != "Yoga" {
		t.Fatalf("migrated %+v %v", class, err)
	}
}

func codeOf(err error) string {
	var target *domain.Error
	if errors.As(err, &target) {
		return target.Code
	}
	if err == nil {
		return ""
	}
	return err.Error()
}
