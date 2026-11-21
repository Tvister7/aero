package store

import (
	"context"
	"database/sql"
	"strconv"
	"time"

	"github.com/Tvister7/aero/internal/domain"
)

type querier interface {
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
}

const classSelect = `
select c.id, c.title, coalesce(t.id::text, ''), coalesce(t.name, ''), c.starts_at, c.ends_at, c.capacity,
       (select count(*) from bookings b where b.class_id = c.id and b.status = 'booked'),
       c.price_min, c.price_max, c.status, coalesce(ty.color, ''), coalesce(c.training_type_id::text, ''), c.template_id, c.kind
from classes c
left join trainers t on t.id = c.trainer_id
left join training_types ty on ty.id = c.training_type_id`

func loadClass(ctx context.Context, q querier, id string) (domain.Class, error) {
	var class domain.Class
	var template, trainer, typeID sql.NullString
	err := q.QueryRowContext(ctx, `
		select id, template_id, trainer_id, training_type_id, title, starts_at, ends_at, capacity, price_min, price_max, status, kind
		from classes where id = ?`, id).Scan(
		&class.ID, &template, &trainer, &typeID, &class.Title, &class.StartsAt, &class.EndsAt,
		&class.Capacity, &class.PriceMin, &class.PriceMax, &class.Status, &class.Kind,
	)
	if err != nil {
		return domain.Class{}, err
	}
	if template.Valid {
		class.TemplateID = template.String
	}
	if trainer.Valid {
		class.TrainerID = trainer.String
	}
	if typeID.Valid {
		class.TypeID = typeID.String
	}
	return class, nil
}

func loadView(ctx context.Context, q querier, id string, admin bool) (ClassView, error) {
	query := classSelect + ` where c.id = ?`
	if !admin {
		query += ` and c.status = 'scheduled'`
	}
	return scanClass(q.QueryRowContext(ctx, query, id), admin)
}

type scanner interface{ Scan(dest ...any) error }

func scanClass(row scanner, admin bool) (ClassView, error) {
	var view ClassView
	var starts, ends time.Time
	var status string
	var template sql.NullString
	err := row.Scan(
		&view.ID, &view.Title, &view.Trainer.ID, &view.Trainer.Name, &starts, &ends,
		&view.Capacity, &view.Booked, &view.PriceMin, &view.PriceMax, &status, &view.Color,
		&view.TrainingTypeID, &template, &view.Kind,
	)
	if err != nil {
		return ClassView{}, err
	}
	if template.Valid && template.String != "" {
		id := template.String
		view.TemplateID = &id
	}
	view.StartsAt = domain.FormatISO(starts)
	view.EndsAt = domain.FormatISO(ends)
	view.FreePlaces = view.Capacity - view.Booked
	if view.FreePlaces < 0 {
		view.FreePlaces = 0
	}
	if admin {
		view.Status = &status
	}
	if !admin && view.Kind == domain.KindHold {
		view.Title = "Студия занята"
		view.PriceMin = ""
		view.PriceMax = ""
	}
	if view.Color == "" && view.Kind == domain.KindHold {
		view.Color = "#9aa3b2"
	}
	if view.Color == "" && view.Kind == domain.KindIndividual {
		view.Color = "#7a5ea7"
	}
	return view, nil
}

func insertClass(ctx context.Context, q querier, now time.Time, class domain.Class) error {
	_, err := q.ExecContext(ctx, `
		insert into classes (id, template_id, trainer_id, training_type_id, title, starts_at, ends_at, capacity, price_min, price_max, status, kind, created_at)
		values (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		class.ID, nullable(class.TemplateID), nullable(class.TrainerID), nullable(class.TypeID), class.Title,
		utc(class.StartsAt), utc(class.EndsAt), class.Capacity, class.PriceMin, class.PriceMax, class.Status, classKind(class.Kind), utc(now),
	)
	return err
}

func updateClass(ctx context.Context, q querier, class domain.Class) error {
	_, err := q.ExecContext(ctx, `
		update classes
		set template_id = ?, trainer_id = ?, training_type_id = ?, title = ?, starts_at = ?, ends_at = ?, capacity = ?, price_min = ?, price_max = ?, status = ?, kind = ?
		where id = ?`,
		nullable(class.TemplateID), nullable(class.TrainerID), nullable(class.TypeID), class.Title,
		utc(class.StartsAt), utc(class.EndsAt), class.Capacity, class.PriceMin, class.PriceMax, class.Status, classKind(class.Kind), class.ID,
	)
	return err
}

func findClient(ctx context.Context, q querier, telegramID int64) (domain.Client, error) {
	row := q.QueryRowContext(ctx, `
		select id, messenger_user_id, messenger_username, name, phone, policy
		from clients where messenger_platform = 'telegram' and messenger_user_id = ?`, strconv.FormatInt(telegramID, 10))
	return scanClient(row)
}

func scanClient(row scanner) (domain.Client, error) {
	var client domain.Client
	var telegram string
	err := row.Scan(&client.ID, &telegram, &client.Username, &client.Name, &client.Phone, &client.Policy)
	if err != nil {
		return domain.Client{}, err
	}
	client.TelegramID, err = strconv.ParseInt(telegram, 10, 64)
	return client, err
}

func ensureClient(ctx context.Context, q querier, ids func() string, now time.Time, user User) (domain.Client, error) {
	client, err := findClient(ctx, q, user.TelegramID)
	if err == nil {
		return client, nil
	}
	if err != sql.ErrNoRows {
		return domain.Client{}, err
	}
	client = domain.Client{
		ID: ids(), TelegramID: user.TelegramID, Username: user.Username, Policy: domain.Postpay,
	}
	_, err = q.ExecContext(ctx, `
		insert into clients (id, messenger_platform, messenger_user_id, messenger_username, name, phone, policy, created_at, updated_at)
		values (?, 'telegram', ?, ?, '', '', ?, ?, ?)`,
		client.ID, strconv.FormatInt(user.TelegramID, 10), user.Username, client.Policy, utc(now), utc(now),
	)
	return client, err
}

func updateClient(ctx context.Context, q querier, now time.Time, client domain.Client) error {
	_, err := q.ExecContext(ctx, `
		update clients set messenger_username = ?, name = ?, phone = ?, policy = ?, updated_at = ? where id = ?`,
		client.Username, client.Name, client.Phone, client.Policy, utc(now), client.ID,
	)
	return err
}

func balanceOf(ctx context.Context, q querier, clientID string) (int, error) {
	var balance int
	err := q.QueryRowContext(ctx, `select coalesce(sum(delta), 0) from credit_entries where client_id = ?`, clientID).Scan(&balance)
	return balance, err
}

func bookedCount(ctx context.Context, q querier, classID string) (int, error) {
	var count int
	err := q.QueryRowContext(ctx, `select count(*) from bookings where class_id = ? and status = 'booked'`, classID).Scan(&count)
	return count, err
}

func hasBooking(ctx context.Context, q querier, classID, clientID string) (bool, error) {
	var count int
	err := q.QueryRowContext(ctx, `
		select count(*) from bookings where class_id = ? and client_id = ? and status = 'booked'`, classID, clientID).Scan(&count)
	return count > 0, err
}

func loadBooking(ctx context.Context, q querier, id string) (domain.Booking, error) {
	var booking domain.Booking
	err := q.QueryRowContext(ctx, `
		select id, class_id, client_id, status, reminder_enabled, reminder_at, reminder_state
		from bookings where id = ?`, id).Scan(
		&booking.ID, &booking.ClassID, &booking.ClientID, &booking.Status, &booking.ReminderEnabled, &booking.ReminderAt, &booking.ReminderState,
	)
	return booking, err
}

func loadBooked(ctx context.Context, q querier, classID string) ([]domain.Booking, error) {
	rows, err := q.QueryContext(ctx, `
		select id, class_id, client_id, status, reminder_enabled, reminder_at, reminder_state
		from bookings where class_id = ? and status = 'booked' order by created_at, id`, classID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.Booking
	for rows.Next() {
		var booking domain.Booking
		if err := rows.Scan(&booking.ID, &booking.ClassID, &booking.ClientID, &booking.Status, &booking.ReminderEnabled, &booking.ReminderAt, &booking.ReminderState); err != nil {
			return nil, err
		}
		out = append(out, booking)
	}
	return out, rows.Err()
}

func insertBooking(ctx context.Context, q querier, now time.Time, booking domain.Booking) error {
	_, err := q.ExecContext(ctx, `
		insert into bookings (id, class_id, client_id, status, reminder_enabled, reminder_at, reminder_state, created_at, updated_at)
		values (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		booking.ID, booking.ClassID, booking.ClientID, booking.Status, booking.ReminderEnabled,
		utc(booking.ReminderAt), booking.ReminderState, utc(now), utc(now),
	)
	return err
}

func updateBooking(ctx context.Context, q querier, now time.Time, booking domain.Booking) error {
	_, err := q.ExecContext(ctx, `
		update bookings
		set class_id = ?, status = ?, reminder_enabled = ?, reminder_at = ?, reminder_state = ?, updated_at = ?
		where id = ?`,
		booking.ClassID, booking.Status, booking.ReminderEnabled, utc(booking.ReminderAt), booking.ReminderState, utc(now), booking.ID,
	)
	return err
}

func insertEvent(ctx context.Context, q querier, now time.Time, event domain.Event) error {
	when := event.CreatedAt
	if when.IsZero() {
		when = now
	}
	_, err := q.ExecContext(ctx, `
		insert into booking_events (id, booking_id, event_type, old_class_id, new_class_id, created_at)
		values (?, ?, ?, ?, ?, ?)`,
		event.ID, event.BookingID, event.Type, nullable(event.OldClassID), nullable(event.NewClassID), utc(when),
	)
	return err
}

func insertLedger(ctx context.Context, q querier, entry domain.Ledger) error {
	_, err := q.ExecContext(ctx, `
		insert into credit_entries (id, client_id, delta, reason, booking_id, note, actor_telegram_id, created_at)
		values (?, ?, ?, ?, ?, ?, ?, ?)`,
		entry.ID, entry.ClientID, entry.Delta, entry.Reason, nullable(entry.BookingID), entry.Note, entry.ActorTelegramID, utc(entry.CreatedAt),
	)
	return err
}

func clientTelegram(ctx context.Context, q querier, clientID string) (int64, error) {
	var raw string
	err := q.QueryRowContext(ctx, `select messenger_user_id from clients where id = ?`, clientID).Scan(&raw)
	if err != nil {
		return 0, err
	}
	return strconv.ParseInt(raw, 10, 64)
}

func outbound(ctx context.Context, q querier, notices []domain.Notice) ([]Outbound, error) {
	out := []Outbound{}
	for _, notice := range notices {
		id, err := clientTelegram(ctx, q, notice.ClientID)
		if err != nil {
			return nil, err
		}
		out = append(out, Outbound{TelegramID: id, Text: notice.Text})
	}
	return out, nil
}

func classKind(kind string) string {
	if kind == "" {
		return domain.KindGroup
	}
	return kind
}

func missing(err error, code string) error {
	if err == sql.ErrNoRows {
		return domain.Fail(code)
	}
	return err
}
