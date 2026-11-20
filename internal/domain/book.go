package domain

import (
	"strings"
	"time"
)

type BookRequest struct {
	Now           time.Time
	Class         Class
	BookedCount   int
	AlreadyBooked bool
	Client        Client
	Balance       int
	Name          string
	Phone         string
	Username      string
	BookingID     string
	EventID       string
}

type BookResult struct {
	Client  Client
	Booking Booking
	Event   Event
	Ledger  *Ledger
}

func Book(in BookRequest) (BookResult, *Error) {
	if in.Class.Status != Scheduled || in.Class.Kind == KindIndividual || in.Class.Kind == KindHold {
		return BookResult{}, Fail(ClassNotFound)
	}
	if !in.Class.StartsAt.After(in.Now) {
		return BookResult{}, Fail(ClassStarted)
	}
	if in.BookedCount >= in.Class.Capacity {
		return BookResult{}, Fail(NoPlaces)
	}
	if in.AlreadyBooked {
		return BookResult{}, Fail(AlreadyBooked)
	}
	if in.Client.Policy == Restricted && in.Balance <= 0 {
		return BookResult{}, Fail(NoCredits)
	}
	client := merged(in.Client, in.Name, in.Phone, in.Username)
	if client.Policy == "" {
		client.Policy = Postpay
	}
	return BookResult{
		Client: client,
		Booking: Booking{
			ID:              in.BookingID,
			ClassID:         in.Class.ID,
			ClientID:        client.ID,
			Status:          Booked,
			ReminderEnabled: true,
			ReminderAt:      in.Class.StartsAt.Add(-24 * time.Hour),
			ReminderState:   ReminderPending,
		},
		Event: Event{
			ID:         in.EventID,
			BookingID:  in.BookingID,
			Type:       EventCreated,
			NewClassID: in.Class.ID,
			CreatedAt:  in.Now,
		},
	}, nil
}

func merged(client Client, name, phone, username string) Client {
	if next := strings.TrimSpace(name); next != "" {
		client.Name = next
	}
	if next := strings.TrimSpace(phone); next != "" {
		client.Phone = next
	}
	if next := strings.TrimSpace(username); next != "" {
		client.Username = next
	}
	return client
}

func MergeProfile(client Client, name, phone string) Client {
	return merged(client, name, phone, "")
}

func Cancel(now time.Time, booking Booking, class Class, eventID string) (Booking, Event, *Error) {
	if booking.Status != Booked {
		return Booking{}, Event{}, Fail(BookingNotFound)
	}
	if class.StartsAt.Sub(now) <= 6*time.Hour {
		return Booking{}, Event{}, Fail(CutoffExceeded)
	}
	booking.Status = BookingCancelled
	booking.ReminderState = ReminderCancelled
	return booking, Event{
		ID:         eventID,
		BookingID:  booking.ID,
		Type:       EventCancelled,
		OldClassID: class.ID,
		CreatedAt:  now,
	}, nil
}

type RescheduleRequest struct {
	Now           time.Time
	Booking       Booking
	Current       Class
	Next          Class
	NextBooked    int
	AlreadyOnNext bool
	EventID       string
}

func Reschedule(in RescheduleRequest) (Booking, Event, *Error) {
	if in.Booking.Status != Booked {
		return Booking{}, Event{}, Fail(BookingNotFound)
	}
	if in.Current.StartsAt.Sub(in.Now) <= 6*time.Hour {
		return Booking{}, Event{}, Fail(CutoffExceeded)
	}
	if in.Next.ID == in.Booking.ClassID {
		return Booking{}, Event{}, Fail(SameClass)
	}
	if in.Current.TemplateID != "" || in.Current.Kind == KindIndividual || in.Current.Kind == KindHold {
		return Booking{}, Event{}, Fail(Validation)
	}
	if in.Next.Status != Scheduled || !in.Next.StartsAt.After(in.Now) {
		return Booking{}, Event{}, Fail(NewClassNotFound)
	}
	if in.Next.TypeID == "" || in.Next.TypeID != in.Current.TypeID {
		return Booking{}, Event{}, Fail(NewClassNotFound)
	}
	if in.NextBooked >= in.Next.Capacity {
		return Booking{}, Event{}, Fail(NoPlaces)
	}
	if in.AlreadyOnNext {
		return Booking{}, Event{}, Fail(AlreadyBooked)
	}
	old := in.Booking.ClassID
	in.Booking.ClassID = in.Next.ID
	in.Booking.ReminderState = ReminderPending
	in.Booking.ReminderAt = in.Next.StartsAt.Add(-24 * time.Hour)
	return in.Booking, Event{
		ID:         in.EventID,
		BookingID:  in.Booking.ID,
		Type:       EventRescheduled,
		OldClassID: old,
		NewClassID: in.Next.ID,
		CreatedAt:  in.Now,
	}, nil
}
