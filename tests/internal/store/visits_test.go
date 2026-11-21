package store_test

import (
	"context"
	"testing"
	"time"

	"github.com/Tvister7/aero/internal/domain"
	. "github.com/Tvister7/aero/internal/store"
)

func TestVisitsKeepsActiveAttendance(t *testing.T) {
	db := openTest(t)
	ctx := context.Background()
	now := time.Date(2026, 11, 19, 10, 0, 0, 0, domain.Tbilisi)
	trainer, err := db.CreateTrainer(ctx, now, "Лилия")
	if err != nil {
		t.Fatal(err)
	}
	kind := makeType(t, db, now, "Aero Yoga")
	start := now.Add(20 * time.Minute)
	class, err := db.CreateClass(ctx, now, ClassInput{
		TrainerID: trainer.ID, TypeID: kind.ID, StartsAt: domain.FormatISO(start),
	})
	if err != nil {
		t.Fatal(err)
	}
	user := User{TelegramID: 1001, Username: "anna"}
	booked, err := db.Book(ctx, now, user, class.ID, "Анна", "+995555000000")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Attend(ctx, start, booked.BookingID, true, domain.ActorAdmin, "100"); err != nil {
		t.Fatal(err)
	}
	visits, err := db.Visits(ctx, user)
	if err != nil || len(visits) != 1 || visits[0].Payment != "unpaid" || visits[0].TrainerName != "Лилия" {
		t.Fatalf("visits %+v %v", visits, err)
	}
	if _, err := db.Attend(ctx, start, booked.BookingID, false, domain.ActorAdmin, "100"); err != nil {
		t.Fatal(err)
	}
	visits, err = db.Visits(ctx, user)
	if err != nil || len(visits) != 0 {
		t.Fatalf("cleared %+v %v", visits, err)
	}
	if empty, err := db.Visits(ctx, User{TelegramID: 404}); err != nil || len(empty) != 0 {
		t.Fatalf("unknown %+v %v", empty, err)
	}
}
