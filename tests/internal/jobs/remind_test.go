package jobs_test

import (
	"context"
	"io"
	"testing"
	"time"

	"github.com/Tvister7/aero/internal/domain"
	"github.com/Tvister7/aero/internal/jobs"
	"github.com/Tvister7/aero/internal/store"
)

type failBot struct{}

func (failBot) Send(context.Context, int64, string) error { return io.EOF }

func TestFailedReminderStaysPending(t *testing.T) {
	db, cleanup, err := store.OpenTemp()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := cleanup(); err != nil {
			t.Error(err)
		}
	})
	now := time.Date(2026, 11, 18, 10, 0, 0, 0, domain.Tbilisi)
	trainer, err := db.CreateTrainer(context.Background(), now, "Лилия")
	if err != nil {
		t.Fatal(err)
	}
	kind, err := db.CreateType(context.Background(), now, store.TypeInput{
		Title: str("Aero Yoga"), DurationMinutes: num(60), Capacity: num(6), PriceMin: str("17"), PriceMax: str("40"),
	})
	if err != nil {
		t.Fatal(err)
	}
	class, err := db.CreateClass(context.Background(), now, store.ClassInput{
		TrainerID: trainer.ID, TypeID: kind.ID, StartsAt: "2026-11-19T10:00:00+04:00",
	})
	if err != nil {
		t.Fatal(err)
	}
	booked, err := db.Book(context.Background(), now, store.User{TelegramID: 1001, Username: "anna"}, class.ID, "Анна", "+995555000000")
	if err != nil {
		t.Fatal(err)
	}
	var scheduler jobs.Scheduler
	scheduler.Tick(context.Background(), db, failBot{}, now)
	state, err := db.ReminderState(context.Background(), booked.BookingID)
	if err != nil || state != domain.ReminderPending {
		t.Fatalf("state %s %v", state, err)
	}
}

func str(value string) *string { return &value }

func num(value int) *int { return &value }
