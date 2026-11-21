package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/Tvister7/aero/internal/domain"
)

func (d *DB) Schedule(ctx context.Context, from, to string, admin bool) ([]ClassView, error) {
	start, end, err := dayBounds(from, to)
	if err != nil {
		return nil, err
	}
	query := classSelect + ` where c.starts_at >= ? and c.starts_at < ?`
	if !admin {
		query += ` and c.status = 'scheduled' and c.kind in ('group', 'hold')`
	}
	query += ` order by c.starts_at, c.id`
	rows, err := d.sql.QueryContext(ctx, query, start, end)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []ClassView{}
	for rows.Next() {
		view, err := scanClass(rows, admin)
		if err != nil {
			return nil, err
		}
		out = append(out, view)
	}
	return out, rows.Err()
}

func (d *DB) Class(ctx context.Context, id string, admin bool) (ClassView, error) {
	view, err := loadView(ctx, d.sql, id, admin)
	if err != nil {
		return ClassView{}, missing(err, domain.ClassNotFound)
	}
	if !admin && view.Kind == domain.KindIndividual {
		return ClassView{}, domain.Fail(domain.ClassNotFound)
	}
	return view, nil
}

func (d *DB) Book(ctx context.Context, now time.Time, user User, classID, name, phone string) (BookOutcome, error) {
	var out BookOutcome
	err := d.write(ctx, func(tx *sql.Tx) error {
		client, err := ensureClient(ctx, tx, d.ids, now, user)
		if err != nil {
			return err
		}
		class, err := loadClass(ctx, tx, classID)
		if err != nil {
			return missing(err, domain.ClassNotFound)
		}
		count, err := bookedCount(ctx, tx, classID)
		if err != nil {
			return err
		}
		already, err := hasBooking(ctx, tx, classID, client.ID)
		if err != nil {
			return err
		}
		balance, err := balanceOf(ctx, tx, client.ID)
		if err != nil {
			return err
		}
		result, derr := domain.Book(domain.BookRequest{
			Now: now, Class: class, BookedCount: count, AlreadyBooked: already,
			Client: client, Balance: balance, Name: name, Phone: phone, Username: user.Username,
			BookingID: d.ids(), EventID: d.ids(),
		})
		if derr != nil {
			return derr
		}
		if err := updateClient(ctx, tx, now, result.Client); err != nil {
			return err
		}
		var prior int
		if err := tx.QueryRowContext(ctx, `select count(*) from bookings where client_id = ?`, client.ID).Scan(&prior); err != nil {
			return err
		}
		if err := insertBooking(ctx, tx, now, result.Booking); err != nil {
			return err
		}
		if err := insertEvent(ctx, tx, now, result.Event); err != nil {
			return err
		}
		view, err := loadView(ctx, tx, classID, false)
		if err != nil {
			return err
		}
		balance, err = balanceOf(ctx, tx, client.ID)
		if err != nil {
			return err
		}
		out = BookOutcome{
			BookingID: result.Booking.ID, Balance: balance, Class: view, First: prior == 0,
			ClientName: result.Client.Name, ClientPhone: result.Client.Phone, Username: result.Client.Username,
		}
		return nil
	})
	return out, err
}

func (d *DB) MyBookings(ctx context.Context, now time.Time, user User) ([]MyBooking, error) {
	client, err := findClient(ctx, d.sql, user.TelegramID)
	if errors.Is(err, sql.ErrNoRows) {
		return []MyBooking{}, nil
	}
	if err != nil {
		return nil, err
	}
	rows, err := d.sql.QueryContext(ctx, `
		select b.id from bookings b
		join classes c on c.id = b.class_id
		where b.client_id = ? and b.status = 'booked' and c.status = 'scheduled' and c.starts_at > ?
		order by c.starts_at, b.id`, client.ID, utc(now))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []MyBooking{}
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	for _, id := range ids {
		booking, err := loadBooking(ctx, d.sql, id)
		if err != nil {
			return nil, err
		}
		view, err := loadView(ctx, d.sql, booking.ClassID, false)
		if err != nil {
			return nil, err
		}
		out = append(out, MyBooking{BookingID: id, Class: view})
	}
	return out, nil
}

func (d *DB) Visits(ctx context.Context, user User) ([]Visit, error) {
	client, err := findClient(ctx, d.sql, user.TelegramID)
	if errors.Is(err, sql.ErrNoRows) {
		return []Visit{}, nil
	}
	if err != nil {
		return nil, err
	}
	rows, err := d.sql.QueryContext(ctx, `
		select c.title, t.name, c.starts_at, a.payment
		from attendances a
		join bookings b on b.id = a.booking_id
		join classes c on c.id = b.class_id
		join trainers t on t.id = c.trainer_id
		where a.active = 1 and b.client_id = ?
		order by c.starts_at desc, a.id desc`, client.ID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Visit{}
	for rows.Next() {
		var item Visit
		var start string
		if err := rows.Scan(&item.Title, &item.TrainerName, &start, &item.Payment); err != nil {
			return nil, err
		}
		when, err := parseTime(start)
		if err != nil {
			return nil, err
		}
		item.StartsAt = domain.FormatISO(when)
		out = append(out, item)
	}
	return out, rows.Err()
}

func (d *DB) CancelBooking(ctx context.Context, now time.Time, user User, bookingID string) (string, error) {
	err := d.write(ctx, func(tx *sql.Tx) error {
		client, err := findClient(ctx, tx, user.TelegramID)
		if err != nil {
			return missing(err, domain.BookingNotFound)
		}
		booking, err := loadBooking(ctx, tx, bookingID)
		if err != nil {
			return missing(err, domain.BookingNotFound)
		}
		if booking.ClientID != client.ID {
			return domain.Fail(domain.BookingNotFound)
		}
		class, err := loadClass(ctx, tx, booking.ClassID)
		if err != nil {
			return missing(err, domain.BookingNotFound)
		}
		next, event, derr := domain.Cancel(now, booking, class, d.ids())
		if derr != nil {
			return derr
		}
		if err := updateBooking(ctx, tx, now, next); err != nil {
			return err
		}
		if class.Kind == domain.KindIndividual {
			class.Status = domain.Cancelled
			if err := updateClass(ctx, tx, class); err != nil {
				return err
			}
		}
		return insertEvent(ctx, tx, now, event)
	})
	return bookingID, err
}

func (d *DB) Reschedule(ctx context.Context, now time.Time, user User, bookingID, classID string) (ClassView, error) {
	var view ClassView
	err := d.write(ctx, func(tx *sql.Tx) error {
		client, err := findClient(ctx, tx, user.TelegramID)
		if err != nil {
			return missing(err, domain.BookingNotFound)
		}
		booking, err := loadBooking(ctx, tx, bookingID)
		if err != nil {
			return missing(err, domain.BookingNotFound)
		}
		if booking.ClientID != client.ID {
			return domain.Fail(domain.BookingNotFound)
		}
		current, err := loadClass(ctx, tx, booking.ClassID)
		if err != nil {
			return missing(err, domain.BookingNotFound)
		}
		next, err := loadClass(ctx, tx, classID)
		if errors.Is(err, sql.ErrNoRows) {
			next = domain.Class{ID: classID}
		} else if err != nil {
			return err
		}
		count, err := bookedCount(ctx, tx, classID)
		if err != nil {
			return err
		}
		already, err := hasBooking(ctx, tx, classID, client.ID)
		if err != nil {
			return err
		}
		updated, event, derr := domain.Reschedule(domain.RescheduleRequest{
			Now: now, Booking: booking, Current: current, Next: next,
			NextBooked: count, AlreadyOnNext: already, EventID: d.ids(),
		})
		if derr != nil {
			return derr
		}
		if err := updateBooking(ctx, tx, now, updated); err != nil {
			return err
		}
		if err := insertEvent(ctx, tx, now, event); err != nil {
			return err
		}
		view, err = loadView(ctx, tx, classID, false)
		return err
	})
	return view, err
}

func (d *DB) CreateClass(ctx context.Context, now time.Time, in ClassInput) (ClassView, error) {
	var view ClassView
	err := d.write(ctx, func(tx *sql.Tx) error {
		if in.TrainerID == "" || in.TypeID == "" || in.StartsAt == "" {
			return domain.Fail(domain.Validation)
		}
		if _, err := trainerName(ctx, tx, in.TrainerID); err != nil {
			return missing(err, domain.Validation)
		}
		kind, err := loadType(ctx, tx, in.TypeID)
		if err != nil {
			return missing(err, domain.Validation)
		}
		start, derr := domain.ParseInstant(in.StartsAt)
		if derr != nil {
			return derr
		}
		capacity := kind.Capacity
		if in.Capacity != nil {
			capacity = *in.Capacity
		}
		if capacity < 1 {
			return domain.Fail(domain.Validation)
		}
		class := domain.Class{
			ID: d.ids(), TrainerID: in.TrainerID, TypeID: kind.ID, Title: kind.Title,
			StartsAt: start, EndsAt: start.Add(time.Duration(kind.DurationMinutes) * time.Minute),
			Capacity: capacity, PriceMin: kind.PriceMin, PriceMax: kind.PriceMax, Status: domain.Scheduled,
			Kind: domain.KindGroup,
		}
		if err := d.studioBusy(ctx, tx, class.StartsAt, class.EndsAt, ""); err != nil {
			return err
		}
		if err := insertClass(ctx, tx, now, class); err != nil {
			return err
		}
		view, err = loadView(ctx, tx, class.ID, true)
		return err
	})
	return view, err
}

func (d *DB) PatchClass(ctx context.Context, now time.Time, id string, patch ClassPatch) (ClassChange, error) {
	var change ClassChange
	err := d.write(ctx, func(tx *sql.Tx) error {
		class, err := loadClass(ctx, tx, id)
		if err != nil {
			return missing(err, domain.ClassNotFound)
		}
		bookings, err := loadBooked(ctx, tx, id)
		if err != nil {
			return err
		}
		if class.Kind == domain.KindHold && (patch.Capacity != nil || patch.TrainerID != nil) {
			return domain.Fail(domain.Validation)
		}
		if class.Kind == domain.KindIndividual && patch.Capacity != nil && *patch.Capacity != 1 {
			return domain.Fail(domain.Validation)
		}
		if patch.Status != nil {
			if *patch.Status != domain.Cancelled {
				return domain.Fail(domain.Validation)
			}
			result, derr := domain.CancelClass(class, bookings, d.ids)
			if derr != nil {
				return derr
			}
			if err := updateClass(ctx, tx, result.Class); err != nil {
				return err
			}
			for _, booking := range result.Bookings {
				if err := updateBooking(ctx, tx, now, booking); err != nil {
					return err
				}
			}
			for _, event := range result.Events {
				if err := insertEvent(ctx, tx, now, event); err != nil {
					return err
				}
			}
			view, err := loadView(ctx, tx, id, true)
			if err != nil {
				return err
			}
			notes, err := outbound(ctx, tx, result.Notices)
			if err != nil {
				return err
			}
			change = ClassChange{Class: view, Notices: notes}
			return nil
		}
		if patch.TrainerID != nil {
			if _, err := trainerName(ctx, tx, *patch.TrainerID); err != nil {
				return missing(err, domain.Validation)
			}
		}
		var starts *time.Time
		if patch.StartsAt != nil {
			parsed, derr := domain.ParseInstant(*patch.StartsAt)
			if derr != nil {
				return derr
			}
			starts = &parsed
		}
		result, derr := domain.MoveClass(domain.MoveRequest{
			Now: now, Class: class, Booked: len(bookings), Bookings: bookings,
			StartsAt: starts, TrainerID: patch.TrainerID, Capacity: patch.Capacity, Title: patch.Title,
		})
		if derr != nil {
			return derr
		}
		var other string
		err = tx.QueryRowContext(ctx, `select id from classes where trainer_id = ? and starts_at = ? and id <> ?`,
			result.Class.TrainerID, utc(result.Class.StartsAt), result.Class.ID).Scan(&other)
		if err == nil {
			return domain.Fail(domain.Validation)
		}
		if !errors.Is(err, sql.ErrNoRows) {
			return err
		}
		if err := d.studioBusy(ctx, tx, result.Class.StartsAt, result.Class.EndsAt, result.Class.ID); err != nil {
			return err
		}
		if err := updateClass(ctx, tx, result.Class); err != nil {
			return err
		}
		for _, booking := range result.Bookings {
			if err := updateBooking(ctx, tx, now, booking); err != nil {
				return err
			}
		}
		view, err := loadView(ctx, tx, id, true)
		if err != nil {
			return err
		}
		notes, err := outbound(ctx, tx, result.Notices)
		if err != nil {
			return err
		}
		change = ClassChange{Class: view, Notices: notes}
		return nil
	})
	return change, err
}

func (d *DB) Roster(ctx context.Context, id string) (Roster, error) {
	var roster Roster
	view, err := loadView(ctx, d.sql, id, true)
	if err != nil {
		return Roster{}, missing(err, domain.ClassNotFound)
	}
	if view.Status != nil && *view.Status != domain.Scheduled {
		return Roster{}, domain.Fail(domain.ClassNotFound)
	}
	rows, err := d.sql.QueryContext(ctx, `
		select b.id, c.id, c.name, c.phone, c.messenger_username, a.payment, a.marked_at
		from bookings b
		join clients c on c.id = b.client_id
		left join attendances a on a.booking_id = b.id and a.active = 1
		where b.class_id = ? and b.status = 'booked'
		order by c.name, b.id`, id)
	if err != nil {
		return Roster{}, err
	}
	defer rows.Close()
	roster.Class = view
	roster.Bookings = []RosterBooking{}
	for rows.Next() {
		var item RosterBooking
		var payment, marked sql.NullString
		if err := rows.Scan(&item.BookingID, &item.ClientID, &item.Name, &item.Phone, &item.Username, &payment, &marked); err != nil {
			return Roster{}, err
		}
		if payment.Valid {
			when, err := parseTime(marked.String)
			if err != nil {
				return Roster{}, err
			}
			item.Attendance = &AttendanceView{Payment: payment.String, MarkedAt: domain.FormatISO(when)}
		}
		roster.Bookings = append(roster.Bookings, item)
	}
	return roster, rows.Err()
}

func (d *DB) Attend(ctx context.Context, now time.Time, bookingID string, arrived bool, role, actor string) (AttendOutcome, error) {
	var out AttendOutcome
	err := d.write(ctx, func(tx *sql.Tx) error {
		booking, err := loadBooking(ctx, tx, bookingID)
		if err != nil {
			return missing(err, domain.BookingNotFound)
		}
		class, err := loadClass(ctx, tx, booking.ClassID)
		if err != nil {
			return missing(err, domain.BookingNotFound)
		}
		balance, err := balanceOf(ctx, tx, booking.ClientID)
		if err != nil {
			return err
		}
		active, err := loadActiveAttendance(ctx, tx, booking.ID)
		if err != nil {
			return err
		}
		result, derr := domain.Attend(domain.AttendRequest{
			Now: now, Class: class, Booking: booking, Balance: balance, Active: active, Arrived: arrived,
			ActorTelegramID: actor, ActorRole: role, AttendanceID: d.ids(), LedgerID: d.ids(),
		})
		if derr != nil {
			return derr
		}
		if result.Ledger != nil {
			if err := insertLedger(ctx, tx, *result.Ledger); err != nil {
				return err
			}
		}
		if result.Updated != nil {
			if _, err := tx.ExecContext(ctx, `update attendances set active = 0 where id = ?`, result.Updated.ID); err != nil {
				return err
			}
		}
		if result.Inserted != nil {
			row := result.Inserted
			if _, err := tx.ExecContext(ctx, `
				insert into attendances (id, booking_id, active, payment, actor_role, actor_telegram_id, marked_at, ledger_entry_id)
				values (?, ?, 1, ?, ?, ?, ?, ?)`,
				row.ID, row.BookingID, row.Payment, row.ActorRole, row.ActorTelegramID, utc(row.MarkedAt), row.LedgerID,
			); err != nil {
				return err
			}
		}
		balance, err = balanceOf(ctx, tx, booking.ClientID)
		if err != nil {
			return err
		}
		out = AttendOutcome{BookingID: booking.ID, Balance: balance}
		if result.Current != nil {
			out.Attendance = &AttendanceView{Payment: result.Current.Payment, MarkedAt: domain.FormatISO(result.Current.MarkedAt)}
		}
		return nil
	})
	return out, err
}

func loadActiveAttendance(ctx context.Context, q querier, bookingID string) (*domain.Attendance, error) {
	var row domain.Attendance
	var active int
	var marked string
	err := q.QueryRowContext(ctx, `
		select id, booking_id, active, payment, actor_role, actor_telegram_id, marked_at, ledger_entry_id
		from attendances where booking_id = ? and active = 1`, bookingID).Scan(
		&row.ID, &row.BookingID, &active, &row.Payment, &row.ActorRole, &row.ActorTelegramID, &marked, &row.LedgerID,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	row.Active = active == 1
	row.MarkedAt, err = parseTime(marked)
	return &row, err
}

func dayBounds(from, to string) (string, string, error) {
	start, err := domain.ParseDay(from)
	if err != nil {
		return "", "", err
	}
	end, err := domain.ParseDay(to)
	if err != nil {
		return "", "", err
	}
	if end.Before(start) {
		return "", "", domain.Fail(domain.Validation)
	}
	return utc(start), utc(end.AddDate(0, 0, 1)), nil
}
