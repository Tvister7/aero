package domain

import (
	"strings"
	"time"
)

type MoveRequest struct {
	Now       time.Time
	Class     Class
	Booked    int
	Bookings  []Booking
	StartsAt  *time.Time
	TrainerID *string
	Capacity  *int
	Title     *string
}

type MoveResult struct {
	Class    Class
	Bookings []Booking
	Notices  []Notice
}

func MoveClass(in MoveRequest) (MoveResult, *Error) {
	if in.Class.Status != Scheduled {
		return MoveResult{}, Fail(ClassNotFound)
	}
	if in.Capacity != nil && (*in.Capacity < 1 || *in.Capacity < in.Booked) {
		return MoveResult{}, Fail(Validation)
	}
	if in.Title != nil && strings.TrimSpace(*in.Title) == "" {
		return MoveResult{}, Fail(Validation)
	}
	if in.TrainerID != nil && strings.TrimSpace(*in.TrainerID) == "" {
		return MoveResult{}, Fail(Validation)
	}
	class := in.Class
	startChanged := false
	if in.StartsAt != nil && !in.StartsAt.Equal(class.StartsAt) {
		duration := class.EndsAt.Sub(class.StartsAt)
		class.StartsAt = *in.StartsAt
		class.EndsAt = in.StartsAt.Add(duration)
		startChanged = true
	}
	if in.TrainerID != nil {
		class.TrainerID = *in.TrainerID
	}
	if in.Capacity != nil {
		class.Capacity = *in.Capacity
	}
	if in.Title != nil {
		class.Title = strings.TrimSpace(*in.Title)
	}
	bookings := append([]Booking(nil), in.Bookings...)
	if startChanged {
		for i := range bookings {
			if bookings[i].Status != Booked {
				continue
			}
			bookings[i].ReminderAt = class.StartsAt.Add(-24 * time.Hour)
			if bookings[i].ReminderState == ReminderSent && class.StartsAt.Sub(in.Now) > 24*time.Hour {
				bookings[i].ReminderState = ReminderPending
			}
		}
	}
	var notices []Notice
	if startChanged || in.TrainerID != nil || in.Title != nil {
		text := MoveText(class.Title, class.StartsAt)
		for _, booking := range bookings {
			if booking.Status == Booked {
				notices = append(notices, Notice{ClientID: booking.ClientID, Text: text})
			}
		}
	}
	return MoveResult{Class: class, Bookings: bookings, Notices: notices}, nil
}

type CancelClassResult struct {
	Class    Class
	Bookings []Booking
	Events   []Event
	Notices  []Notice
}

func CancelClass(class Class, bookings []Booking, eventID func() string) (CancelClassResult, *Error) {
	if class.Status != Scheduled {
		return CancelClassResult{}, Fail(ClassNotFound)
	}
	class.Status = Cancelled
	text := CancelText(class.Title, class.StartsAt)
	var events []Event
	var notices []Notice
	next := append([]Booking(nil), bookings...)
	for i := range next {
		if next[i].Status != Booked {
			continue
		}
		next[i].Status = BookingCancelled
		next[i].ReminderState = ReminderCancelled
		events = append(events, Event{
			ID:         eventID(),
			BookingID:  next[i].ID,
			Type:       EventCancelled,
			OldClassID: class.ID,
		})
		notices = append(notices, Notice{ClientID: next[i].ClientID, Text: text})
	}
	return CancelClassResult{Class: class, Bookings: next, Events: events, Notices: notices}, nil
}
