package domain

import "time"

type ReminderCandidate struct {
	BookingID   string
	State       string
	ReminderAt  time.Time
	ClassStatus string
	StartsAt    time.Time
	Title       string
	ClientID    string
}

type ReminderDecision struct {
	Send []ReminderCandidate
	Skip []string
}

func Reminders(now time.Time, items []ReminderCandidate) ReminderDecision {
	var decision ReminderDecision
	for _, item := range items {
		if item.State != ReminderPending {
			continue
		}
		if item.ClassStatus != Scheduled || !item.StartsAt.After(now) {
			decision.Skip = append(decision.Skip, item.BookingID)
			continue
		}
		if !item.ReminderAt.After(now) {
			decision.Send = append(decision.Send, item)
		}
	}
	return decision
}
