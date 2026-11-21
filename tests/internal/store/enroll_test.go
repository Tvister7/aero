package store_test

import (
	"context"
	"testing"
	"time"

	"github.com/Tvister7/aero/internal/domain"
	. "github.com/Tvister7/aero/internal/store"
)

func TestEnrollWeekByPolicy(t *testing.T) {
	db := openTest(t)
	ctx := context.Background()
	now := time.Date(2026, 11, 18, 10, 0, 0, 0, domain.Tbilisi)
	trainer, err := db.CreateTrainer(ctx, now, "Лилия")
	if err != nil {
		t.Fatal(err)
	}
	kind, err := db.CreateType(ctx, now, TypeInput{
		Title: strPtr("Aero Yoga"), DurationMinutes: intPtr(60), Capacity: intPtr(6),
		PriceMin: strPtr("17"), PriceMax: strPtr("40"),
	})
	if err != nil {
		t.Fatal(err)
	}
	weekday := 1
	clock := "10:00"
	template, err := db.CreateTemplate(ctx, now, TemplateInput{
		TrainerID: &trainer.ID, TrainingTypeID: &kind.ID, Weekday: &weekday, StartTime: &clock,
	})
	if err != nil {
		t.Fatal(err)
	}
	postpay := User{TelegramID: 1001, Username: "anna"}
	restricted := User{TelegramID: 1002, Username: "maria"}
	if _, err := db.Me(ctx, now, postpay); err != nil {
		t.Fatal(err)
	}
	other, err := db.Me(ctx, now, restricted)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.PatchClient(ctx, now, other.ID, domain.Restricted); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ReplaceEnrollments(ctx, now, postpay, []string{template.ID}); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ReplaceEnrollments(ctx, now, restricted, []string{template.ID}); err != nil {
		t.Fatal(err)
	}
	if _, err := db.GenerateWeek(ctx, now, "2026-11-23"); err != nil {
		t.Fatal(err)
	}
	booked, err := db.MyBookings(ctx, now, postpay)
	if err != nil || len(booked) != 1 {
		t.Fatalf("postpay bookings %+v %v", booked, err)
	}
	skipped, err := db.MyBookings(ctx, now, restricted)
	if err != nil || len(skipped) != 0 {
		t.Fatalf("restricted bookings %+v %v", skipped, err)
	}
	notices, err := db.PendingBalanceNotices(ctx, now.Add(8*24*time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	var bookedNotice, skippedNotice int
	for _, notice := range notices {
		if notice.Kind == domain.NoticeBookedEmpty && notice.TelegramID == 1001 {
			bookedNotice++
		}
		if notice.Kind == domain.NoticeSkippedEmpty && notice.TelegramID == 1002 {
			skippedNotice++
		}
	}
	if bookedNotice != 1 || skippedNotice != 1 {
		t.Fatalf("notices %+v", notices)
	}
	if _, err := db.GenerateWeek(ctx, now, "2026-11-23"); err != nil {
		t.Fatal(err)
	}
	again, err := db.PendingBalanceNotices(ctx, now.Add(8*24*time.Hour))
	if err != nil || len(again) != len(notices) {
		t.Fatalf("duplicate notices %+v %v", again, err)
	}
	if _, err := db.Grant(ctx, now, other.ID, 1, "пакет", "1"); err != nil {
		t.Fatal(err)
	}
	if _, err := db.GenerateWeek(ctx, now, "2026-11-23"); err != nil {
		t.Fatal(err)
	}
	after, err := db.MyBookings(ctx, now, restricted)
	if err != nil || len(after) != 1 {
		t.Fatalf("restricted after grant %+v %v", after, err)
	}
}

func TestVacationSkipsTemplate(t *testing.T) {
	db := openTest(t)
	ctx := context.Background()
	now := time.Date(2026, 11, 18, 10, 0, 0, 0, domain.Tbilisi)
	trainer, err := db.CreateTrainer(ctx, now, "Тая")
	if err != nil {
		t.Fatal(err)
	}
	kind, err := db.CreateType(ctx, now, TypeInput{
		Title: strPtr("K-POP"), DurationMinutes: intPtr(60), Capacity: intPtr(8),
		PriceMin: strPtr("17"), PriceMax: strPtr("35"),
	})
	if err != nil {
		t.Fatal(err)
	}
	weekday := 1
	clock := "18:00"
	if _, err := db.CreateTemplate(ctx, now, TemplateInput{
		TrainerID: &trainer.ID, TrainingTypeID: &kind.ID, Weekday: &weekday, StartTime: &clock,
	}); err != nil {
		t.Fatal(err)
	}
	if _, _, err := db.AddAbsence(ctx, now, trainer.ID, AbsenceInput{StartsOn: "2026-11-23", EndsOn: "2026-11-23"}); err != nil {
		t.Fatal(err)
	}
	week, err := db.GenerateWeek(ctx, now, "2026-11-23")
	if err != nil {
		t.Fatal(err)
	}
	if week.Created != 0 {
		t.Fatalf("created during leave: %+v", week)
	}
}
