package domain

import "time"

type AttendRequest struct {
	Now             time.Time
	Class           Class
	Booking         Booking
	Balance         int
	Active          *Attendance
	Arrived         bool
	ActorTelegramID string
	AttendanceID    string
	LedgerID        string
}

type AttendResult struct {
	Inserted *Attendance
	Updated  *Attendance
	Current  *Attendance
	Ledger   *Ledger
}

func Attend(in AttendRequest) (AttendResult, *Error) {
	if in.Booking.Status != Booked || in.Class.Status != Scheduled || in.Class.ID != in.Booking.ClassID {
		return AttendResult{}, Fail(BookingNotFound)
	}
	if in.Arrived && in.Now.Before(in.Class.StartsAt.Add(-30*time.Minute)) {
		return AttendResult{}, Fail(AttendanceTooEarly)
	}
	if in.Arrived && in.Active != nil {
		return AttendResult{Current: in.Active}, nil
	}
	if !in.Arrived && in.Active == nil {
		return AttendResult{}, nil
	}
	if in.Arrived {
		payment := Unpaid
		if in.Balance > 0 {
			payment = Paid
		}
		row := &Attendance{
			ID:              in.AttendanceID,
			BookingID:       in.Booking.ID,
			Active:          true,
			Payment:         payment,
			ActorRole:       ActorAdmin,
			ActorTelegramID: in.ActorTelegramID,
			MarkedAt:        in.Now,
			LedgerID:        in.LedgerID,
		}
		return AttendResult{Inserted: row, Current: row, Ledger: attendLedger(in, -1, ReasonAttend)}, nil
	}
	updated := *in.Active
	updated.Active = false
	return AttendResult{Updated: &updated, Ledger: attendLedger(in, 1, ReasonReversal)}, nil
}

func attendLedger(in AttendRequest, delta int, reason string) *Ledger {
	return &Ledger{
		ID:              in.LedgerID,
		ClientID:        in.Booking.ClientID,
		BookingID:       in.Booking.ID,
		Delta:           delta,
		Reason:          reason,
		ActorTelegramID: in.ActorTelegramID,
		CreatedAt:       in.Now,
	}
}

func ValidateGrant(delta int) *Error {
	if delta < 1 {
		return Fail(Validation)
	}
	return nil
}

func IsNoShow(asOf time.Time, class Class, booking Booking, activeAttendance bool) bool {
	if activeAttendance || booking.Status != Booked || class.Status != Scheduled {
		return false
	}
	return !class.EndsAt.After(asOf)
}
