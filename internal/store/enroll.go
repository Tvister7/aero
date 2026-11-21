package store

import (
	"context"
	"strconv"
	"strings"
	"time"

	"github.com/Tvister7/aero/internal/domain"
)

func (d *DB) EnrollmentOptions(ctx context.Context, now time.Time, user User) ([]EnrollmentOption, error) {
	var clientID string
	err := d.write(ctx, func(tx querier) error {
		client, err := ensureClient(ctx, tx, d.ids, now, user)
		if err != nil {
			return err
		}
		clientID = client.ID
		return nil
	})
	if err != nil {
		return nil, err
	}
	return listEnrollmentOptions(ctx, d.sql, clientID)
}

func (d *DB) ReplaceEnrollments(ctx context.Context, now time.Time, user User, templateIDs []string) ([]EnrollmentOption, error) {
	var clientID string
	err := d.write(ctx, func(tx querier) error {
		client, err := ensureClient(ctx, tx, d.ids, now, user)
		if err != nil {
			return err
		}
		clientID = client.ID
		clean, err := activeTemplateIDs(ctx, tx, templateIDs)
		if err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `update enrollments set active = false where client_id = ?`, client.ID); err != nil {
			return err
		}
		for _, id := range clean {
			if _, err := tx.ExecContext(ctx, `
				insert into enrollments (id, client_id, template_id, active, created_at)
				values (?, ?, ?, true, ?)
				on conflict (client_id, template_id) do update set active = true`,
				d.ids(), client.ID, id, utc(now)); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return listEnrollmentOptions(ctx, d.sql, clientID)
}

func activeTemplateIDs(ctx context.Context, q querier, templateIDs []string) ([]string, error) {
	seen := map[string]bool{}
	var clean []string
	for _, id := range templateIDs {
		id = strings.TrimSpace(id)
		if id == "" || seen[id] {
			continue
		}
		seen[id] = true
		var found int
		if err := q.QueryRowContext(ctx, `
			select count(*) from class_templates tpl
			join trainers tr on tr.id = tpl.trainer_id
			where tpl.id = ? and tpl.active = true and tr.active = true`, id).Scan(&found); err != nil {
			return nil, err
		}
		if found == 0 {
			return nil, domain.Fail(domain.Validation)
		}
		clean = append(clean, id)
	}
	return clean, nil
}

func listEnrollmentOptions(ctx context.Context, q querier, clientID string) ([]EnrollmentOption, error) {
	rows, err := q.QueryContext(ctx, `
		select tpl.id, tpl.trainer_id, tr.name, tpl.title, tpl.weekday, tpl.start_time,
		       exists(select 1 from enrollments e where e.template_id = tpl.id and e.client_id = ? and e.active = true)
		from class_templates tpl
		join trainers tr on tr.id = tpl.trainer_id
		where tpl.active = true and tr.active = true
		order by tr.name, tpl.weekday, tpl.start_time, tpl.id`, clientID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []EnrollmentOption{}
	for rows.Next() {
		var item EnrollmentOption
		if err := rows.Scan(&item.TemplateID, &item.TrainerID, &item.TrainerName, &item.Title, &item.Weekday, &item.StartTime, &item.Enrolled); err != nil {
			return nil, err
		}
		item.StartTime, err = shownClock(item.StartTime)
		if err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func loadAbsences(ctx context.Context, q querier) ([]domain.Absence, error) {
	rows, err := q.QueryContext(ctx, `select trainer_id, starts_on, ends_on from trainer_absences`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.Absence
	for rows.Next() {
		var item domain.Absence
		var start, end time.Time
		if err := rows.Scan(&item.TrainerID, &start, &end); err != nil {
			return nil, err
		}
		item.StartsOn = dayText(start)
		item.EndsOn = dayText(end)
		out = append(out, item)
	}
	return out, rows.Err()
}

func enrollWeek(ctx context.Context, tx querier, ids func() string, now, monday time.Time) error {
	rows, err := tx.QueryContext(ctx, `
		select e.client_id, c.id
		from enrollments e
		join classes c on c.template_id = e.template_id
		where e.active = true and c.status = 'scheduled'
		  and c.starts_at >= ? and c.starts_at < ? and c.starts_at > ?
		order by e.client_id, c.starts_at, c.id`,
		utc(monday), utc(monday.AddDate(0, 0, 7)), utc(now))
	if err != nil {
		return err
	}
	defer rows.Close()
	var order []string
	grouped := map[string][]string{}
	for rows.Next() {
		var clientID, classID string
		if err := rows.Scan(&clientID, &classID); err != nil {
			return err
		}
		if _, ok := grouped[clientID]; !ok {
			order = append(order, clientID)
		}
		grouped[clientID] = append(grouped[clientID], classID)
	}
	if err := rows.Err(); err != nil {
		return err
	}
	if err := rows.Close(); err != nil {
		return err
	}
	for _, clientID := range order {
		if err := enrollClient(ctx, tx, ids, now, clientID, grouped[clientID]); err != nil {
			return err
		}
	}
	return nil
}

func enrollClient(ctx context.Context, tx querier, ids func() string, now time.Time, clientID string, classIDs []string) error {
	client, err := clientByID(ctx, tx, clientID)
	if err != nil {
		return err
	}
	balance, err := balanceOf(ctx, tx, clientID)
	if err != nil {
		return err
	}
	future, err := futureUnattended(ctx, tx, clientID, now)
	if err != nil {
		return err
	}
	slots := make([]domain.SlotChoice, 0, len(classIDs))
	for _, classID := range classIDs {
		class, err := loadClass(ctx, tx, classID)
		if err != nil {
			return err
		}
		count, err := bookedCount(ctx, tx, classID)
		if err != nil {
			return err
		}
		already, err := hasBooking(ctx, tx, classID, clientID)
		if err != nil {
			return err
		}
		slots = append(slots, domain.SlotChoice{Class: class, BookedCount: count, Already: already})
	}
	for _, decision := range domain.DecideEnrollments(client.Policy, balance, domain.Allowance(client.Policy, balance, future), slots) {
		if decision.Book {
			count, err := bookedCount(ctx, tx, decision.Class.ID)
			if err != nil {
				return err
			}
			already, err := hasBooking(ctx, tx, decision.Class.ID, clientID)
			if err != nil {
				return err
			}
			result, derr := domain.Book(domain.BookRequest{
				Now: now, Class: decision.Class, BookedCount: count, AlreadyBooked: already,
				Client: client, Balance: balance, BookingID: ids(), EventID: ids(),
			})
			if derr != nil {
				if derr.Code == domain.NoPlaces || derr.Code == domain.AlreadyBooked {
					continue
				}
				if derr.Code == domain.NoCredits {
					if err := queueNotice(ctx, tx, ids, now, clientID, decision.Class, domain.NoticeSkippedEmpty); err != nil {
						return err
					}
					continue
				}
				return derr
			}
			if err := insertBooking(ctx, tx, now, result.Booking); err != nil {
				return err
			}
			if err := insertEvent(ctx, tx, now, result.Event); err != nil {
				return err
			}
			if _, err := tx.ExecContext(ctx, `
				update balance_notices set state = 'skipped'
				where client_id = ? and class_id = ? and kind = ? and state = 'pending'`,
				clientID, decision.Class.ID, domain.NoticeSkippedEmpty); err != nil {
				return err
			}
		}
		if decision.Notice != "" {
			if err := queueNotice(ctx, tx, ids, now, clientID, decision.Class, decision.Notice); err != nil {
				return err
			}
		}
	}
	return nil
}

func clientByID(ctx context.Context, q querier, id string) (domain.Client, error) {
	return scanClient(q.QueryRowContext(ctx, `
		select id, messenger_user_id, messenger_username, name, phone, policy
		from clients where id = ?`, id))
}

func futureUnattended(ctx context.Context, q querier, clientID string, now time.Time) (int, error) {
	var count int
	err := q.QueryRowContext(ctx, `
		select count(*) from bookings b
		join classes c on c.id = b.class_id
		where b.client_id = ? and b.status = 'booked' and c.status = 'scheduled' and c.starts_at > ?
		  and not exists (select 1 from attendances a where a.booking_id = b.id and a.active = true)`,
		clientID, utc(now)).Scan(&count)
	return count, err
}

func queueNotice(ctx context.Context, q querier, ids func() string, now time.Time, clientID string, class domain.Class, kind string) error {
	_, err := q.ExecContext(ctx, `
		insert into balance_notices (id, client_id, class_id, kind, state, notice_at, created_at)
		values (?, ?, ?, ?, 'pending', ?, ?)
		on conflict (client_id, class_id, kind) do nothing`,
		ids(), clientID, class.ID, kind, utc(class.StartsAt.Add(-24*time.Hour)), utc(now))
	return err
}

type DueBalanceNotice struct {
	domain.BalanceCandidate
	TelegramID int64
}

func (d *DB) PendingBalanceNotices(ctx context.Context, now time.Time) ([]DueBalanceNotice, error) {
	rows, err := d.sql.QueryContext(ctx, `
		select n.id, n.kind, n.state, n.notice_at, c.status, c.starts_at, c.title, cl.policy, cl.messenger_user_id,
		       (select coalesce(sum(delta), 0) from credit_entries e where e.client_id = n.client_id),
		       exists(select 1 from bookings b where b.class_id = n.class_id and b.client_id = n.client_id and b.status = 'booked'),
		       (select count(*) from bookings b
		          join classes c2 on c2.id = b.class_id
		          where b.client_id = n.client_id and b.status = 'booked' and c2.status = 'scheduled' and c2.starts_at > ?
		            and not exists (select 1 from attendances a where a.booking_id = b.id and a.active = true))
		from balance_notices n
		join classes c on c.id = n.class_id
		join clients cl on cl.id = n.client_id
		where n.state = 'pending'`, utc(now))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []DueBalanceNotice
	for rows.Next() {
		var item DueBalanceNotice
		var telegram string
		if err := rows.Scan(
			&item.ID, &item.Kind, &item.State, &item.NoticeAt, &item.ClassStatus, &item.StartsAt, &item.Title, &item.Policy, &telegram,
			&item.Balance, &item.Booked, &item.Future,
		); err != nil {
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

func (d *DB) SetBalanceNoticeState(ctx context.Context, id, state string) error {
	return d.write(ctx, func(tx querier) error {
		_, err := tx.ExecContext(ctx, `
			update balance_notices set state = ? where id = ? and state = 'pending'`, state, id)
		return err
	})
}
