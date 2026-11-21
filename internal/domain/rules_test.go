package domain

import (
	"testing"
	"time"
)

func at(y int, m time.Month, d, h, min int) time.Time {
	return time.Date(y, m, d, h, min, 0, 0, Tbilisi)
}

func yogaAt(start time.Time) Class {
	return Class{
		ID: "class-1", TrainerID: "tr", TypeID: "type", Title: "Aero Yoga",
		StartsAt: start, EndsAt: start.Add(time.Hour), Capacity: 6,
		PriceMin: "17.00", PriceMax: "40.00", Status: Scheduled,
	}
}

func clientRestricted(balance int) Client {
	return Client{ID: "anna", TelegramID: 1001, Username: "anna", Name: "Анна", Phone: "+995555112233", Policy: Restricted}
}

func TestFirstClassNotices(t *testing.T) {
	starts := at(2026, 11, 21, 10, 0)
	if PlaceLink(" https://maps.example/aero ", "https://aero.example") != "https://maps.example/aero" {
		t.Fatal("map link")
	}
	if PlaceLink("", "https://aero.example/") != "https://aero.example/#/studio/guide" {
		t.Fatal("guide link")
	}
	if PlaceLink("  ", "") != "" {
		t.Fatal("empty link")
	}
	client := FirstClassText("Aero Yoga", starts, "https://aero.example/#/studio/guide")
	if client != "Вы записаны на первое занятие: Aero Yoga, 21.11 10:00.\nКак нас найти: https://aero.example/#/studio/guide\nВ день занятия администратор напишет вам код от подъезда." {
		t.Fatalf("client %q", client)
	}
	admin := AdminFirstBookingText(" Анна ", "@anna", " +995555112233 ", "Aero Yoga", starts)
	if admin != "Первая запись: Анна, @anna, +995555112233.\nAero Yoga, 21.11 10:00.\nНапишите код от подъезда в день занятия." {
		t.Fatalf("admin %q", admin)
	}
	if AdminFirstBookingText("", "", "", "Aero Yoga", starts) != "Первая запись: без имени.\nAero Yoga, 21.11 10:00.\nНапишите код от подъезда в день занятия." {
		t.Fatal("nameless")
	}
}

func TestBookRestrictedWithoutCredits(t *testing.T) {
	_, err := Book(BookRequest{
		Now: at(2026, 11, 18, 10, 0), Class: yogaAt(at(2026, 11, 19, 10, 0)),
		Client: clientRestricted(0), Balance: 0, Name: "Анна", Phone: "+995555112233",
	})
	if code(err) != NoCredits {
		t.Fatalf("got %v, want NO_CREDITS", err)
	}
}

func TestBookPostpayWithDebtDoesNotTouchBalance(t *testing.T) {
	out, err := Book(BookRequest{
		Now: at(2026, 11, 18, 10, 0), Class: yogaAt(at(2026, 11, 19, 10, 0)),
		Client:    Client{ID: "nina", TelegramID: 5, Policy: Postpay, Name: "Нино"},
		Balance:   -2,
		Name:      "Нино",
		Phone:     "+995555112233",
		BookingID: "b1",
		EventID:   "e1",
	})
	if err != nil {
		t.Fatal(err)
	}
	if out.Booking.Status != Booked || out.Booking.ReminderState != ReminderPending {
		t.Fatalf("booking %+v", out.Booking)
	}
	wantRemind := at(2026, 11, 18, 10, 0)
	if !out.Booking.ReminderAt.Equal(wantRemind) {
		t.Fatalf("reminder %s", out.Booking.ReminderAt)
	}
	if out.Ledger != nil {
		t.Fatal("booking must not write a ledger row")
	}
	if out.Event.Type != EventCreated || out.Event.NewClassID != "class-1" {
		t.Fatalf("event %+v", out.Event)
	}
}

func TestBookEmptyNameKeepsStored(t *testing.T) {
	out, err := Book(BookRequest{
		Now: at(2026, 11, 18, 10, 0), Class: yogaAt(at(2026, 11, 19, 10, 0)),
		Client: Client{ID: "anna", Name: "Анна", Phone: "+995", Policy: Postpay},
		Name:   "  ", Phone: "", Username: "anna",
		BookingID: "b", EventID: "e",
	})
	if err != nil {
		t.Fatal(err)
	}
	if out.Client.Name != "Анна" || out.Client.Phone != "+995" || out.Client.Username != "anna" {
		t.Fatalf("profile %+v", out.Client)
	}
}

func TestBookRejects(t *testing.T) {
	start := at(2026, 11, 19, 10, 0)
	now := at(2026, 11, 18, 10, 0)
	base := BookRequest{
		Now: now, Class: yogaAt(start),
		Client: Client{ID: "anna", Policy: Restricted}, Balance: 3,
		Name: "Анна", Phone: "+995", BookingID: "b", EventID: "e",
	}
	started := base
	started.Class = yogaAt(now)
	full := base
	full.BookedCount = 6
	again := base
	again.AlreadyBooked = true
	cancelled := base
	cancelled.Class.Status = Cancelled
	both := base
	both.BookedCount = 6
	both.Balance = 0
	cases := []struct {
		name string
		in   BookRequest
		want string
	}{
		{"started", started, ClassStarted},
		{"full", full, NoPlaces},
		{"again", again, AlreadyBooked},
		{"cancelled", cancelled, ClassNotFound},
		{"full before credits", both, NoPlaces},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := Book(tc.in)
			if code(err) != tc.want {
				t.Fatalf("got %v, want %s", err, tc.want)
			}
		})
	}
}

func TestCancelCutoff(t *testing.T) {
	start := at(2026, 11, 19, 10, 0)
	booking := Booking{ID: "b", ClassID: "class-1", ClientID: "anna", Status: Booked, ReminderState: ReminderPending}
	class := yogaAt(start)
	if _, _, err := Cancel(at(2026, 11, 19, 4, 0), booking, class, "e"); code(err) != CutoffExceeded {
		t.Fatalf("exactly 6h: %v", err)
	}
	out, event, err := Cancel(at(2026, 11, 19, 3, 59), booking, class, "e")
	if err != nil {
		t.Fatal(err)
	}
	if out.Status != BookingCancelled || out.ReminderState != ReminderCancelled {
		t.Fatalf("booking %+v", out)
	}
	if event.Type != EventCancelled || event.OldClassID != "class-1" {
		t.Fatalf("event %+v", event)
	}
}

func TestCancelForeignBooking(t *testing.T) {
	booking := Booking{ID: "b", Status: BookingCancelled}
	_, _, err := Cancel(at(2026, 11, 18, 10, 0), booking, yogaAt(at(2026, 11, 19, 10, 0)), "e")
	if code(err) != BookingNotFound {
		t.Fatalf("got %v", err)
	}
}

func TestRescheduleCutoffAndSameClass(t *testing.T) {
	start := at(2026, 11, 19, 10, 0)
	booking := Booking{ID: "b", ClassID: "class-1", ClientID: "anna", Status: Booked, ReminderState: ReminderSent}
	current := yogaAt(start)
	_, _, err := Reschedule(RescheduleRequest{
		Now: at(2026, 11, 19, 5, 0), Booking: booking, Current: current, Next: yogaAt(start.Add(24 * time.Hour)),
	})
	if code(err) != CutoffExceeded {
		t.Fatalf("cutoff: %v", err)
	}
	next := yogaAt(start.Add(48 * time.Hour))
	next.ID = "class-1"
	_, _, err = Reschedule(RescheduleRequest{
		Now: at(2026, 11, 18, 10, 0), Booking: booking, Current: current, Next: next,
	})
	if code(err) != SameClass {
		t.Fatalf("same: %v", err)
	}
}

func TestRescheduleMovesReminder(t *testing.T) {
	start := at(2026, 11, 19, 10, 0)
	nextStart := at(2026, 11, 21, 12, 0)
	next := yogaAt(nextStart)
	next.ID = "class-2"
	booking := Booking{ID: "b", ClassID: "class-1", ClientID: "anna", Status: Booked, ReminderState: ReminderSent}
	out, event, err := Reschedule(RescheduleRequest{
		Now: at(2026, 11, 18, 10, 0), Booking: booking, Current: yogaAt(start), Next: next, EventID: "e",
	})
	if err != nil {
		t.Fatal(err)
	}
	if out.ClassID != "class-2" || out.ReminderState != ReminderPending || !out.ReminderAt.Equal(nextStart.Add(-24*time.Hour)) {
		t.Fatalf("booking %+v", out)
	}
	if event.Type != EventRescheduled || event.OldClassID != "class-1" || event.NewClassID != "class-2" {
		t.Fatalf("event %+v", event)
	}
}

func TestRescheduleRejectsTemplateAndOtherType(t *testing.T) {
	start := at(2026, 11, 19, 10, 0)
	current := yogaAt(start)
	current.TemplateID = "tpl"
	next := yogaAt(start.Add(48 * time.Hour))
	next.ID = "class-2"
	booking := Booking{ID: "b", ClassID: current.ID, Status: Booked}
	_, _, err := Reschedule(RescheduleRequest{Now: at(2026, 11, 18, 10, 0), Booking: booking, Current: current, Next: next})
	if code(err) != Validation {
		t.Fatalf("template: %v", err)
	}
	current.TemplateID = ""
	next.TypeID = "other"
	_, _, err = Reschedule(RescheduleRequest{Now: at(2026, 11, 18, 10, 0), Booking: booking, Current: current, Next: next})
	if code(err) != NewClassNotFound {
		t.Fatalf("type: %v", err)
	}
}

func TestAllowanceAndEnrollment(t *testing.T) {
	if Allowance(Restricted, 2, 1) != 1 || Allowance(Postpay, 0, 0) != 1 || Allowance(Postpay, 0, 1) != 0 || Allowance(Postpay, -1, 0) != 0 || Allowance(Restricted, 0, 0) != 0 {
		t.Fatal(Allowance(Restricted, 2, 1), Allowance(Postpay, 0, 0), Allowance(Postpay, 0, 1))
	}
	open := yogaAt(at(2026, 11, 20, 10, 0))
	full := yogaAt(at(2026, 11, 21, 10, 0))
	full.Capacity = 1
	decisions := DecideEnrollments(Postpay, 0, 1, []SlotChoice{
		{Class: open, BookedCount: 0},
		{Class: full, BookedCount: 1},
	})
	if !decisions[0].Book || decisions[0].Notice != NoticeBookedEmpty || decisions[1].Book || decisions[1].Notice != "" {
		t.Fatalf("postpay %+v", decisions)
	}
	skipped := DecideEnrollments(Restricted, 0, 0, []SlotChoice{{Class: open}})
	if skipped[0].Book || skipped[0].Notice != NoticeSkippedEmpty {
		t.Fatalf("restricted %+v", skipped)
	}
}

func TestAttendanceIdempotentAndReversal(t *testing.T) {
	start := at(2026, 11, 19, 10, 0)
	class := yogaAt(start)
	booking := Booking{ID: "b", ClassID: class.ID, ClientID: "anna", Status: Booked}
	tooSoon := at(2026, 11, 19, 9, 29)
	_, err := Attend(AttendRequest{Now: tooSoon, Class: class, Booking: booking, Arrived: true, Balance: 2})
	if code(err) != AttendanceTooEarly {
		t.Fatalf("early: %v", err)
	}
	now := at(2026, 11, 19, 9, 30)
	first, err := Attend(AttendRequest{
		Now: now, Class: class, Booking: booking, Arrived: true, Balance: 2,
		ActorTelegramID: "7", AttendanceID: "a1", LedgerID: "l1",
	})
	if err != nil {
		t.Fatal(err)
	}
	if first.Inserted == nil || !first.Inserted.Active || first.Inserted.Payment != Paid || first.Ledger == nil || first.Ledger.Delta != -1 {
		t.Fatalf("first %+v ledger %+v", first.Inserted, first.Ledger)
	}
	again, err := Attend(AttendRequest{Now: now, Class: class, Booking: booking, Arrived: true, Balance: 1, Active: first.Inserted})
	if err != nil {
		t.Fatal(err)
	}
	if again.Inserted != nil || again.Ledger != nil || again.Current == nil || again.Current.ID != "a1" {
		t.Fatal("repeat must not write")
	}
	cleared, err := Attend(AttendRequest{Now: now, Class: class, Booking: booking, Arrived: false, Balance: 1, Active: first.Inserted, LedgerID: "l2"})
	if err != nil {
		t.Fatal(err)
	}
	if cleared.Current != nil || cleared.Updated == nil || cleared.Updated.Active || cleared.Ledger == nil || cleared.Ledger.Delta != 1 || cleared.Ledger.Reason != ReasonReversal {
		t.Fatalf("clear %+v", cleared)
	}
	idle, err := Attend(AttendRequest{Now: now, Class: class, Booking: booking, Arrived: false, Balance: 2})
	if err != nil || idle.Ledger != nil || idle.Updated != nil {
		t.Fatalf("second clear %+v %v", idle, err)
	}
	debt, err := Attend(AttendRequest{
		Now: now, Class: class, Booking: booking, Arrived: true, Balance: 0,
		AttendanceID: "a2", LedgerID: "l3", ActorTelegramID: "7",
	})
	if err != nil {
		t.Fatal(err)
	}
	if debt.Inserted.Payment != Unpaid || debt.Ledger.Delta != -1 {
		t.Fatalf("unpaid %+v", debt.Inserted)
	}
}

func TestGenerateWeekIdempotentAndSkips(t *testing.T) {
	monday := at(2026, 11, 16, 0, 0)
	now := at(2026, 11, 18, 12, 0)
	templates := []TemplateSlot{
		{ID: "tpl-tue", TrainerID: "tanya", TypeID: "stretch", Title: "Aero Stretching", Weekday: 2, StartTime: "10:00", Capacity: 6, DurationMinutes: 60, PriceMin: "17.00", PriceMax: "40.00", Active: true, TrainerActive: true},
		{ID: "tpl-thu", TrainerID: "tanya", TypeID: "stretch", Title: "Aero Stretching", Weekday: 4, StartTime: "10:00", Capacity: 6, DurationMinutes: 60, PriceMin: "17.00", PriceMax: "40.00", Active: true, TrainerActive: true},
		{ID: "tpl-off", TrainerID: "zhanna", TypeID: "heels", Title: "HIGH HEELS", Weekday: 5, StartTime: "11:30", Capacity: 9, DurationMinutes: 60, Active: false, TrainerActive: true},
	}
	var n int
	ids := func() string { n++; return "new" }
	first := GenerateWeek(now, monday, templates, nil, nil, ids, true)
	if first.Created != 1 || first.Updated != 0 || len(first.Upserts) != 1 {
		t.Fatalf("first %+v", first)
	}
	got := first.Upserts[0]
	wantStart := at(2026, 11, 19, 10, 0)
	if !got.StartsAt.Equal(wantStart) || got.TemplateID != "tpl-thu" || got.Status != Scheduled || got.Capacity != 6 {
		t.Fatalf("slot %+v", got)
	}
	existing := []ClassSnapshot{{Class: got, Booked: 8}}
	existing[0].Class.Capacity = 6
	second := GenerateWeek(now, monday, templates, existing, nil, ids, true)
	if second.Created != 0 || second.Updated != 1 || second.Upserts[0].Capacity != 8 || second.Upserts[0].ID != got.ID {
		t.Fatalf("second %+v", second)
	}
	cancelled := got
	cancelled.Status = Cancelled
	third := GenerateWeek(now, monday, templates, []ClassSnapshot{{Class: cancelled, Booked: 0}}, nil, ids, true)
	away := GenerateWeek(now, monday, templates, nil, []Absence{{TrainerID: "tanya", StartsOn: "2026-11-19", EndsOn: "2026-11-19"}}, ids, true)
	if away.Created != 0 || len(away.Upserts) != 0 {
		t.Fatalf("leave created %+v", away)
	}
	if third.Created != 0 || third.Updated != 0 || len(third.Upserts) != 0 {
		t.Fatalf("cancelled revived: %+v", third)
	}
}

func TestNextMonday(t *testing.T) {
	if got := NextMonday(at(2026, 11, 16, 0, 10)); !got.Equal(at(2026, 11, 23, 0, 0)) {
		t.Fatalf("monday 00:10 -> %s", got)
	}
	if got := NextMonday(at(2026, 11, 18, 12, 0)); !got.Equal(at(2026, 11, 23, 0, 0)) {
		t.Fatalf("wednesday -> %s", got)
	}
	wed := at(2026, 11, 18, 15, 0)
	if got := MondayOf(wed); !got.Equal(at(2026, 11, 16, 0, 0)) {
		t.Fatalf("monday of wednesday %s", got)
	}
}

func TestMoveReminderAndCapacity(t *testing.T) {
	start := at(2026, 11, 19, 10, 0)
	class := yogaAt(start)
	sent := Booking{ID: "b", ClassID: class.ID, ClientID: "anna", Status: Booked, ReminderState: ReminderSent, ReminderAt: start.Add(-24 * time.Hour)}
	now := at(2026, 11, 19, 9, 0)
	newStart := at(2026, 11, 21, 18, 0)
	out, err := MoveClass(MoveRequest{Now: now, Class: class, Booked: 2, Bookings: []Booking{sent}, StartsAt: &newStart})
	if err != nil {
		t.Fatal(err)
	}
	if !out.Class.EndsAt.Equal(newStart.Add(time.Hour)) || out.Bookings[0].ReminderState != ReminderPending {
		t.Fatalf("move %+v", out)
	}
	soon := now.Add(10 * time.Hour)
	held, err := MoveClass(MoveRequest{Now: now, Class: class, Booked: 2, Bookings: []Booking{sent}, StartsAt: &soon})
	if err != nil {
		t.Fatal(err)
	}
	if held.Bookings[0].ReminderState != ReminderSent {
		t.Fatal("sent reminder inside 24h must stay sent")
	}
	low := 1
	_, err = MoveClass(MoveRequest{Now: now, Class: class, Booked: 2, Capacity: &low})
	if code(err) != Validation {
		t.Fatalf("capacity: %v", err)
	}
	class.Status = Cancelled
	_, err = MoveClass(MoveRequest{Now: now, Class: class, Booked: 0})
	if code(err) != ClassNotFound {
		t.Fatalf("cancelled: %v", err)
	}
}

func TestCancelClass(t *testing.T) {
	start := at(2026, 11, 19, 10, 0)
	class := yogaAt(start)
	booking := Booking{ID: "b", ClassID: class.ID, ClientID: "anna", Status: Booked, ReminderState: ReminderPending}
	var n int
	out, err := CancelClass(class, []Booking{booking}, func() string { n++; return "e1" })
	if err != nil {
		t.Fatal(err)
	}
	if out.Class.Status != Cancelled || out.Bookings[0].Status != BookingCancelled || out.Bookings[0].ReminderState != ReminderCancelled {
		t.Fatalf("%+v", out)
	}
	if len(out.Events) != 1 || out.Events[0].Type != EventCancelled {
		t.Fatalf("events %+v", out.Events)
	}
	_, err = CancelClass(out.Class, nil, func() string { return "x" })
	if code(err) != ClassNotFound {
		t.Fatalf("second cancel %v", err)
	}
}

func TestReminders(t *testing.T) {
	now := at(2026, 11, 18, 10, 0)
	start := at(2026, 11, 19, 10, 0)
	due := ReminderCandidate{BookingID: "b", State: ReminderPending, ReminderAt: now.Add(-time.Minute), ClassStatus: Scheduled, StartsAt: start}
	late := due
	late.BookingID = "late"
	late.StartsAt = now.Add(-time.Hour)
	cancelled := due
	cancelled.BookingID = "c"
	cancelled.ClassStatus = Cancelled
	waiting := due
	waiting.BookingID = "w"
	waiting.ReminderAt = now.Add(time.Hour)
	got := Reminders(now, []ReminderCandidate{due, late, cancelled, waiting})
	if len(got.Send) != 1 || got.Send[0].BookingID != "b" {
		t.Fatalf("send %+v", got.Send)
	}
	if len(got.Skip) != 2 {
		t.Fatalf("skip %+v", got.Skip)
	}
}

func TestNoShow(t *testing.T) {
	end := at(2026, 11, 18, 11, 0)
	class := yogaAt(end.Add(-time.Hour))
	class.EndsAt = end
	booking := Booking{Status: Booked}
	if !IsNoShow(end, class, booking, false) {
		t.Fatal("ended booked class without attendance is a no-show")
	}
	if IsNoShow(end, class, booking, true) {
		t.Fatal("active attendance is not a no-show")
	}
	class.Status = Cancelled
	if IsNoShow(end.Add(time.Hour), class, booking, false) {
		t.Fatal("cancelled class is not a no-show")
	}
}

func TestMoneyAndPriceOrder(t *testing.T) {
	min, max, err := NormalizePrice("17", "40.5")
	if err != nil || min != "17.00" || max != "40.50" {
		t.Fatalf("%s %s %v", min, max, err)
	}
	_, _, err = NormalizePrice("40", "17")
	if code(err) != Validation {
		t.Fatalf("order %v", err)
	}
}

func TestGenerateWeekSkipsOccupiedHour(t *testing.T) {
	monday := at(2026, 11, 16, 0, 0)
	now := at(2026, 11, 18, 12, 0)
	templates := []TemplateSlot{{
		ID: "tpl-thu", TrainerID: "tanya", TypeID: "stretch", Title: "Aero Stretching",
		Weekday: 4, StartTime: "10:00", Capacity: 6, DurationMinutes: 60, Active: true, TrainerActive: true,
	}}
	hold := Class{
		ID: "hold", Title: "Съёмка", Status: Scheduled, Kind: KindHold,
		StartsAt: at(2026, 11, 19, 10, 30), EndsAt: at(2026, 11, 19, 11, 0),
	}
	got := GenerateWeek(now, monday, templates, []ClassSnapshot{{Class: hold}}, nil, func() string { return "new" }, true)
	if got.Created != 0 || len(got.Upserts) != 0 {
		t.Fatalf("occupied %+v", got)
	}
	open := GenerateWeek(now, monday, templates, []ClassSnapshot{{Class: hold}}, nil, func() string { return "new" }, false)
	if open.Created != 1 {
		t.Fatalf("several rooms %+v", open)
	}
}

func code(err error) string {
	if err == nil {
		return ""
	}
	de, ok := err.(*Error)
	if !ok {
		return err.Error()
	}
	return de.Code
}
