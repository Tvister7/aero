package store

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"github.com/Tvister7/aero/internal/domain"
)

func (d *DB) studioBusy(ctx context.Context, q querier, start, end time.Time, ignoreID string) error {
	if !d.Exclusive {
		return nil
	}
	query := `
		select id from classes
		where status = 'scheduled' and starts_at < ? and ends_at > ?`
	args := []any{utc(end), utc(start)}
	if ignoreID != "" {
		query += ` and id <> ?`
		args = append(args, ignoreID)
	}
	query += ` limit 1`
	var id string
	err := q.QueryRowContext(ctx, query, args...).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	return domain.Fail(domain.StudioBusy)
}

func durationMinutes(value *int) (int, error) {
	minutes := 60
	if value != nil {
		minutes = *value
	}
	if minutes < 1 || minutes > 600 {
		return 0, domain.Fail(domain.Validation)
	}
	return minutes, nil
}

func (d *DB) CreateHold(ctx context.Context, now time.Time, title, startsAt string, duration *int) (ClassView, error) {
	var view ClassView
	err := d.write(ctx, func(tx querier) error {
		title = strings.TrimSpace(title)
		minutes, err := durationMinutes(duration)
		if err != nil || title == "" {
			return domain.Fail(domain.Validation)
		}
		start, derr := domain.ParseInstant(startsAt)
		if derr != nil {
			return derr
		}
		class := domain.Class{
			ID: d.ids(), Title: title, StartsAt: start, EndsAt: start.Add(time.Duration(minutes) * time.Minute),
			Status: domain.Scheduled, Kind: domain.KindHold,
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

func (d *DB) CreateIndividual(ctx context.Context, now time.Time, trainerID, clientID, startsAt string, duration *int) (ClassView, error) {
	var view ClassView
	err := d.write(ctx, func(tx querier) error {
		minutes, err := durationMinutes(duration)
		if err != nil || trainerID == "" || clientID == "" {
			return domain.Fail(domain.Validation)
		}
		if _, err := trainerName(ctx, tx, trainerID); err != nil {
			return missing(err, domain.Validation)
		}
		var name string
		if err := tx.QueryRowContext(ctx, `select name from clients where id = ?`, clientID).Scan(&name); err != nil {
			return missing(err, domain.Validation)
		}
		start, derr := domain.ParseInstant(startsAt)
		if derr != nil {
			return derr
		}
		class := domain.Class{
			ID: d.ids(), TrainerID: trainerID, Title: "Индивидуальное",
			StartsAt: start, EndsAt: start.Add(time.Duration(minutes) * time.Minute),
			Capacity: 1, Status: domain.Scheduled, Kind: domain.KindIndividual,
		}
		if err := d.studioBusy(ctx, tx, class.StartsAt, class.EndsAt, ""); err != nil {
			return err
		}
		if err := insertClass(ctx, tx, now, class); err != nil {
			return err
		}
		booking := domain.Booking{
			ID: d.ids(), ClassID: class.ID, ClientID: clientID, Status: domain.Booked,
			ReminderEnabled: true, ReminderAt: start.Add(-24 * time.Hour), ReminderState: domain.ReminderPending,
		}
		if err := insertBooking(ctx, tx, now, booking); err != nil {
			return err
		}
		event := domain.Event{
			ID: d.ids(), BookingID: booking.ID, Type: domain.EventCreated, NewClassID: class.ID, CreatedAt: now,
		}
		if err := insertEvent(ctx, tx, now, event); err != nil {
			return err
		}
		view, err = loadView(ctx, tx, class.ID, true)
		return err
	})
	return view, err
}

func (d *DB) SearchClientNames(ctx context.Context, query string) ([]ClientName, error) {
	needle := strings.ToLower(strings.TrimSpace(query))
	if needle == "" {
		return []ClientName{}, nil
	}
	rows, err := d.sql.QueryContext(ctx, `select id, name from clients order by name, id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []ClientName{}
	for rows.Next() {
		var item ClientName
		if err := rows.Scan(&item.ID, &item.Name); err != nil {
			return nil, err
		}
		if !strings.Contains(strings.ToLower(item.Name), needle) {
			continue
		}
		out = append(out, item)
		if len(out) == 20 {
			break
		}
	}
	return out, rows.Err()
}

func (d *DB) MoveTrainerClass(ctx context.Context, now time.Time, trainerID, classID, startsAt string) (ClassChange, error) {
	class, err := loadClass(ctx, d.sql, classID)
	if err != nil {
		return ClassChange{}, missing(err, domain.ClassNotFound)
	}
	if class.TrainerID != trainerID || class.Kind != domain.KindIndividual {
		return ClassChange{}, domain.Fail(domain.ClassNotFound)
	}
	return d.PatchClass(ctx, now, classID, ClassPatch{StartsAt: &startsAt})
}
