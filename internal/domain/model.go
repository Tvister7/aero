package domain

import "time"

const (
	Scheduled = "scheduled"
	Cancelled = "cancelled"

	Booked           = "booked"
	BookingCancelled = "cancelled"

	Postpay    = "postpay"
	Restricted = "restricted"

	ReminderPending   = "pending"
	ReminderSent      = "sent"
	ReminderCancelled = "cancelled"
	ReminderSkipped   = "skipped"

	EventCreated     = "created"
	EventCancelled   = "cancelled"
	EventRescheduled = "rescheduled"

	ReasonGrant    = "grant"
	ReasonAttend   = "attend"
	ReasonReversal = "reversal"

	Paid   = "paid"
	Unpaid = "unpaid"

	ActorAdmin   = "admin"
	ActorTrainer = "trainer"

	KindGroup      = "group"
	KindIndividual = "individual"
	KindHold       = "hold"

	NoticeBookedEmpty  = "booked_empty"
	NoticeSkippedEmpty = "skipped_empty"
)

type Class struct {
	ID         string
	TemplateID string
	TrainerID  string
	TypeID     string
	Title      string
	StartsAt   time.Time
	EndsAt     time.Time
	Capacity   int
	PriceMin   string
	PriceMax   string
	Status     string
	Kind       string
}

type Client struct {
	ID         string
	TelegramID int64
	Username   string
	Name       string
	Phone      string
	Policy     string
}

type Booking struct {
	ID              string
	ClassID         string
	ClientID        string
	Status          string
	ReminderEnabled bool
	ReminderAt      time.Time
	ReminderState   string
}

type Event struct {
	ID         string
	BookingID  string
	Type       string
	OldClassID string
	NewClassID string
	CreatedAt  time.Time
}

type Ledger struct {
	ID              string
	ClientID        string
	BookingID       string
	Delta           int
	Reason          string
	Note            string
	ActorTelegramID string
	CreatedAt       time.Time
}

type Attendance struct {
	ID              string
	BookingID       string
	Active          bool
	Payment         string
	ActorRole       string
	ActorTelegramID string
	MarkedAt        time.Time
	LedgerID        string
}

type Notice struct {
	ClientID string
	Text     string
}
