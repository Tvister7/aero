package sqlite

import (
	"context"
	"database/sql"
	"strconv"
	"strings"
	"time"

	"github.com/Tvister7/aero/internal/domain"
)

func (d *DB) Me(ctx context.Context, now time.Time, user User) (Me, error) {
	var me Me
	err := d.write(ctx, func(tx *sql.Tx) error {
		client, err := ensureClient(ctx, tx, d.ids, now, user)
		if err != nil {
			return err
		}
		balance, err := balanceOf(ctx, tx, client.ID)
		if err != nil {
			return err
		}
		visited, err := hasVisit(ctx, tx, client.ID)
		if err != nil {
			return err
		}
		trainer, err := isActiveTrainer(ctx, tx, user.TelegramID)
		if err != nil {
			return err
		}
		me = meFrom(client, balance, visited)
		me.IsTrainer = trainer
		return nil
	})
	return me, err
}

func isActiveTrainer(ctx context.Context, q querier, telegramID int64) (bool, error) {
	var found int
	err := q.QueryRowContext(ctx, `
		select exists(select 1 from trainers where telegram_user_id = ? and active = 1)`,
		strconv.FormatInt(telegramID, 10)).Scan(&found)
	return found == 1, err
}

func (d *DB) PatchMe(ctx context.Context, now time.Time, user User, name, phone string) (Me, error) {
	var me Me
	err := d.write(ctx, func(tx *sql.Tx) error {
		client, err := ensureClient(ctx, tx, d.ids, now, user)
		if err != nil {
			return err
		}
		client = domain.MergeProfile(client, name, phone)
		if err := updateClient(ctx, tx, now, client); err != nil {
			return err
		}
		balance, err := balanceOf(ctx, tx, client.ID)
		if err != nil {
			return err
		}
		visited, err := hasVisit(ctx, tx, client.ID)
		if err != nil {
			return err
		}
		trainer, err := isActiveTrainer(ctx, tx, user.TelegramID)
		if err != nil {
			return err
		}
		me = meFrom(client, balance, visited)
		me.IsTrainer = trainer
		return nil
	})
	return me, err
}

func hasVisit(ctx context.Context, tx *sql.Tx, clientID string) (bool, error) {
	var visited int
	err := tx.QueryRowContext(ctx, `
		select exists(
			select 1 from attendances a
			join bookings b on b.id = a.booking_id
			where b.client_id = ?
		)`, clientID).Scan(&visited)
	return visited == 1, err
}

func meFrom(client domain.Client, balance int, visited bool) Me {
	return Me{
		ID: client.ID, Name: client.Name, Phone: client.Phone, Username: client.Username,
		Balance: balance, Policy: client.Policy, Visited: visited,
	}
}

func (d *DB) Ping(ctx context.Context, now time.Time, user User) error {
	return d.write(ctx, func(tx *sql.Tx) error {
		if _, err := ensureClient(ctx, tx, d.ids, now, user); err != nil {
			return err
		}
		_, err := tx.ExecContext(ctx, `insert into app_opens (id, messenger_user_id, opened_at) values (?, ?, ?)`,
			d.ids(), strconv.FormatInt(user.TelegramID, 10), utc(now))
		return err
	})
}

func (d *DB) ListClients(ctx context.Context, query string) ([]ClientSummary, error) {
	rows, err := d.sql.QueryContext(ctx, `
		select c.id, c.messenger_user_id, c.messenger_username, c.name, c.phone, c.policy, coalesce(sum(e.delta), 0)
		from clients c
		left join credit_entries e on e.client_id = c.id
		group by c.id
		order by c.name, c.id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	needle := strings.ToLower(strings.TrimSpace(query))
	out := []ClientSummary{}
	for rows.Next() {
		var item ClientSummary
		if err := rows.Scan(&item.ID, &item.TelegramID, &item.Username, &item.Name, &item.Phone, &item.Policy, &item.Balance); err != nil {
			return nil, err
		}
		if needle != "" && !matchesClient(item, needle) {
			continue
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func matchesClient(item ClientSummary, needle string) bool {
	haystack := strings.ToLower(strings.Join([]string{item.Name, item.Phone, item.Username, item.TelegramID}, " "))
	return strings.Contains(haystack, needle)
}

func (d *DB) Client(ctx context.Context, id string) (ClientCard, error) {
	rows, err := d.ListClients(ctx, "")
	if err != nil {
		return ClientCard{}, err
	}
	var summary ClientSummary
	found := false
	for _, item := range rows {
		if item.ID == id {
			summary = item
			found = true
			break
		}
	}
	if !found {
		return ClientCard{}, domain.Fail(domain.Validation)
	}
	ledger, err := d.ledger(ctx, id)
	if err != nil {
		return ClientCard{}, err
	}
	return ClientCard{ClientSummary: summary, Ledger: ledger}, nil
}

func (d *DB) ledger(ctx context.Context, clientID string) ([]LedgerView, error) {
	rows, err := d.sql.QueryContext(ctx, `
		select id, delta, reason, note, booking_id, created_at
		from credit_entries where client_id = ? order by created_at desc, rowid desc`, clientID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []LedgerView{}
	for rows.Next() {
		var item LedgerView
		var booking sql.NullString
		var created string
		if err := rows.Scan(&item.ID, &item.Delta, &item.Reason, &item.Note, &booking, &created); err != nil {
			return nil, err
		}
		when, err := parseTime(created)
		if err != nil {
			return nil, err
		}
		item.CreatedAt = domain.FormatISO(when)
		if booking.Valid {
			item.BookingID = &booking.String
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func (d *DB) PatchClient(ctx context.Context, now time.Time, id, policy string) (ClientCard, error) {
	if policy != domain.Postpay && policy != domain.Restricted {
		return ClientCard{}, domain.Fail(domain.Validation)
	}
	err := d.write(ctx, func(tx *sql.Tx) error {
		res, err := tx.ExecContext(ctx, `update clients set policy = ?, updated_at = ? where id = ?`, policy, utc(now), id)
		if err != nil {
			return err
		}
		n, err := res.RowsAffected()
		if err != nil {
			return err
		}
		if n == 0 {
			return domain.Fail(domain.Validation)
		}
		return nil
	})
	if err != nil {
		return ClientCard{}, err
	}
	return d.Client(ctx, id)
}

func (d *DB) Grant(ctx context.Context, now time.Time, id string, delta int, note, actor string) (ClientCard, error) {
	if err := domain.ValidateGrant(delta); err != nil {
		return ClientCard{}, err
	}
	err := d.write(ctx, func(tx *sql.Tx) error {
		var exists int
		if err := tx.QueryRowContext(ctx, `select count(*) from clients where id = ?`, id).Scan(&exists); err != nil {
			return err
		}
		if exists == 0 {
			return domain.Fail(domain.Validation)
		}
		return insertLedger(ctx, tx, domain.Ledger{
			ID: d.ids(), ClientID: id, Delta: delta, Reason: domain.ReasonGrant,
			Note: strings.TrimSpace(note), ActorTelegramID: actor, CreatedAt: now,
		})
	})
	if err != nil {
		return ClientCard{}, err
	}
	return d.Client(ctx, id)
}

func (d *DB) Stats(ctx context.Context, from, to string) (Stats, error) {
	start, end, err := dayBounds(from, to)
	if err != nil {
		return Stats{}, err
	}
	stats := Stats{From: from, To: to}
	users := map[string]struct{}{}
	rows, err := d.sql.QueryContext(ctx, `select messenger_user_id from app_opens where opened_at >= ? and opened_at < ?`, start, end)
	if err != nil {
		return Stats{}, err
	}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return Stats{}, err
		}
		users[id] = struct{}{}
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return Stats{}, err
	}
	rows.Close()
	stats.UniqueUsers = len(users)
	eventRows, err := d.sql.QueryContext(ctx, `select event_type from booking_events where created_at >= ? and created_at < ?`, start, end)
	if err != nil {
		return Stats{}, err
	}
	for eventRows.Next() {
		var kind string
		if err := eventRows.Scan(&kind); err != nil {
			eventRows.Close()
			return Stats{}, err
		}
		switch kind {
		case domain.EventCreated:
			stats.BookingsCreated++
		case domain.EventCancelled:
			stats.BookingsCancelled++
		}
	}
	if err := eventRows.Err(); err != nil {
		eventRows.Close()
		return Stats{}, err
	}
	eventRows.Close()
	arrivalRows, err := d.sql.QueryContext(ctx, `select payment from attendances where active = 1 and marked_at >= ? and marked_at < ?`, start, end)
	if err != nil {
		return Stats{}, err
	}
	for arrivalRows.Next() {
		var payment string
		if err := arrivalRows.Scan(&payment); err != nil {
			arrivalRows.Close()
			return Stats{}, err
		}
		stats.Arrivals++
		if payment == domain.Unpaid {
			stats.UnpaidVisits++
		}
	}
	if err := arrivalRows.Err(); err != nil {
		arrivalRows.Close()
		return Stats{}, err
	}
	arrivalRows.Close()
	endAt, err := parseTime(end)
	if err != nil {
		return Stats{}, err
	}
	noShowRows, err := d.sql.QueryContext(ctx, `
		select c.status, c.ends_at, b.status, exists(select 1 from attendances a where a.booking_id = b.id and a.active = 1)
		from classes c
		join bookings b on b.class_id = c.id
		where c.ends_at >= ? and c.ends_at < ?`, start, end)
	if err != nil {
		return Stats{}, err
	}
	defer noShowRows.Close()
	for noShowRows.Next() {
		var classStatus, ends, bookingStatus string
		var active int
		if err := noShowRows.Scan(&classStatus, &ends, &bookingStatus, &active); err != nil {
			return Stats{}, err
		}
		endsAt, err := parseTime(ends)
		if err != nil {
			return Stats{}, err
		}
		if domain.IsNoShow(endAt, domain.Class{Status: classStatus, EndsAt: endsAt}, domain.Booking{Status: bookingStatus}, active == 1) {
			stats.NoShows++
		}
	}
	return stats, noShowRows.Err()
}
