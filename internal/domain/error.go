package domain

const (
	Unauthorized       = "UNAUTHORIZED"
	Forbidden          = "FORBIDDEN"
	Validation         = "VALIDATION"
	ClassNotFound      = "CLASS_NOT_FOUND"
	NewClassNotFound   = "NEW_CLASS_NOT_FOUND"
	BookingNotFound    = "BOOKING_NOT_FOUND"
	ClassStarted       = "CLASS_STARTED"
	NoPlaces           = "NO_PLACES"
	AlreadyBooked      = "ALREADY_BOOKED"
	CutoffExceeded     = "CUTOFF_EXCEEDED"
	SameClass          = "SAME_CLASS"
	NoCredits          = "NO_CREDITS"
	AttendanceTooEarly = "ATTENDANCE_TOO_EARLY"
	StudioBusy         = "STUDIO_BUSY"
)

type Error struct {
	Code    string
	Message string
}

func (e *Error) Error() string {
	if e == nil {
		return ""
	}
	return e.Code
}

func Fail(code string) *Error {
	return &Error{Code: code}
}

func FailMessage(code, message string) *Error {
	return &Error{Code: code, Message: message}
}
