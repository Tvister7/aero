package store

import (
	"context"
	"strconv"
	"time"

	"github.com/Tvister7/aero/internal/domain"
)

type DueReminder struct {
	domain.ReminderCandidate
	TelegramID int64
}

func (d *DB) PendingReminders(ctx context.Context) ([]DueReminder, error) {
	rows, err := d.sql.QueryContext(ctx, `
		select b.id, b.client_id, cl.messenger_user_id, b.reminder_state, b.reminder_at, c.status, c.starts_at, c.title
		from bookings b
		join classes c on c.id = b.class_id
		join clients cl on cl.id = b.client_id
		where b.status = 'booked' and b.reminder_state = 'pending'`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []DueReminder
	for rows.Next() {
		var item DueReminder
		var telegram string
		if err := rows.Scan(&item.BookingID, &item.ClientID, &telegram, &item.State, &item.ReminderAt, &item.ClassStatus, &item.StartsAt, &item.Title); err != nil {
			return nil, err
		}
		item.TelegramID, err = strconv.ParseInt(telegram, 10, 64)
		if err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func (d *DB) SetReminderState(ctx context.Context, now time.Time, bookingID, state string) error {
	return d.write(ctx, func(tx querier) error {
		_, err := tx.ExecContext(ctx, `
			update bookings set reminder_state = ?, updated_at = ? where id = ? and reminder_state = 'pending'`,
			state, utc(now), bookingID)
		return err
	})
}

func (d *DB) ReminderState(ctx context.Context, bookingID string) (string, error) {
	var state string
	err := d.sql.QueryRowContext(ctx, `select reminder_state from bookings where id = ?`, bookingID).Scan(&state)
	return state, err
}
