package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/Tvister7/aero/internal/domain"
)

func (d *DB) TrainerCard(ctx context.Context, id string) (TrainerDetail, error) {
	item, err := trainerByID(ctx, d.sql, id)
	if err != nil {
		return TrainerDetail{}, missing(err, domain.Validation)
	}
	absences, err := listAbsences(ctx, d.sql, id)
	if err != nil {
		return TrainerDetail{}, err
	}
	return TrainerDetail{Trainer: item, Absences: absences}, nil
}

func (d *DB) AddAbsence(ctx context.Context, now time.Time, trainerID string, in AbsenceInput) (TrainerDetail, []Outbound, error) {
	start, derr := domain.ParseDay(strings.TrimSpace(in.StartsOn))
	if derr != nil {
		return TrainerDetail{}, nil, derr
	}
	endDay, derr := domain.ParseDay(strings.TrimSpace(in.EndsOn))
	if derr != nil {
		return TrainerDetail{}, nil, derr
	}
	if endDay.Before(start) {
		return TrainerDetail{}, nil, domain.Fail(domain.Validation)
	}
	var notes []Outbound
	err := d.write(ctx, func(tx *sql.Tx) error {
		if _, err := trainerByID(ctx, tx, trainerID); err != nil {
			return missing(err, domain.Validation)
		}
		if _, err := tx.ExecContext(ctx, `
			insert into trainer_absences (id, trainer_id, starts_on, ends_on, created_at)
			values (?, ?, ?, ?, ?)`,
			d.ids(), trainerID, domain.DateKey(start), domain.DateKey(endDay), utc(now)); err != nil {
			return err
		}
		until := endDay.AddDate(0, 0, 1)
		notices, err := cancelTrainerClasses(ctx, tx, d.ids, now, trainerID, &start, &until, false)
		if err != nil {
			return err
		}
		notes, err = outbound(ctx, tx, notices)
		return err
	})
	if err != nil {
		return TrainerDetail{}, nil, err
	}
	card, err := d.TrainerCard(ctx, trainerID)
	return card, notes, err
}

func (d *DB) DeleteAbsence(ctx context.Context, trainerID, absenceID string) (TrainerDetail, error) {
	err := d.write(ctx, func(tx *sql.Tx) error {
		res, err := tx.ExecContext(ctx, `delete from trainer_absences where id = ? and trainer_id = ?`, absenceID, trainerID)
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
		return TrainerDetail{}, err
	}
	return d.TrainerCard(ctx, trainerID)
}

func (d *DB) CancelUpcoming(ctx context.Context, now time.Time, trainerID string) ([]Outbound, error) {
	var notes []Outbound
	err := d.write(ctx, func(tx *sql.Tx) error {
		if _, err := trainerByID(ctx, tx, trainerID); err != nil {
			return missing(err, domain.Validation)
		}
		notices, err := cancelTrainerClasses(ctx, tx, d.ids, now, trainerID, nil, nil, true)
		if err != nil {
			return err
		}
		notes, err = outbound(ctx, tx, notices)
		return err
	})
	return notes, err
}

func (d *DB) DeleteTrainer(ctx context.Context, now time.Time, trainerID string) ([]Outbound, error) {
	var notes []Outbound
	err := d.write(ctx, func(tx *sql.Tx) error {
		if _, err := trainerByID(ctx, tx, trainerID); err != nil {
			return missing(err, domain.Validation)
		}
		notices, err := cancelTrainerClasses(ctx, tx, d.ids, now, trainerID, nil, nil, true)
		if err != nil {
			return err
		}
		notes, err = outbound(ctx, tx, notices)
		if err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `update class_templates set active = 0 where trainer_id = ?`, trainerID); err != nil {
			return err
		}
		var classes int
		if err := tx.QueryRowContext(ctx, `select count(*) from classes where trainer_id = ?`, trainerID).Scan(&classes); err != nil {
			return err
		}
		if classes == 0 {
			if _, err := tx.ExecContext(ctx, `
				delete from enrollments where template_id in (select id from class_templates where trainer_id = ?)`, trainerID); err != nil {
				return err
			}
			if _, err := tx.ExecContext(ctx, `delete from class_templates where trainer_id = ?`, trainerID); err != nil {
				return err
			}
			if _, err := tx.ExecContext(ctx, `delete from trainer_absences where trainer_id = ?`, trainerID); err != nil {
				return err
			}
			_, err = tx.ExecContext(ctx, `delete from trainers where id = ?`, trainerID)
			return err
		}
		_, err = tx.ExecContext(ctx, `
			update trainers set active = 0, telegram_user_id = null where id = ?`, trainerID)
		return err
	})
	return notes, err
}

func cancelTrainerClasses(ctx context.Context, tx *sql.Tx, ids func() string, now time.Time, trainerID string, from, until *time.Time, onlyFuture bool) ([]domain.Notice, error) {
	query := `select id from classes where trainer_id = ? and status = 'scheduled'`
	args := []any{trainerID}
	if from != nil {
		query += ` and starts_at >= ?`
		args = append(args, utc(*from))
	}
	if until != nil {
		query += ` and starts_at < ?`
		args = append(args, utc(*until))
	}
	if onlyFuture {
		query += ` and starts_at > ?`
		args = append(args, utc(now))
	}
	rows, err := tx.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var classIDs []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		classIDs = append(classIDs, id)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	var notices []domain.Notice
	for _, id := range classIDs {
		class, err := loadClass(ctx, tx, id)
		if err != nil {
			return nil, err
		}
		part, err := applyClassCancel(ctx, tx, ids, now, class)
		if err != nil {
			return nil, err
		}
		notices = append(notices, part...)
	}
	return notices, nil
}

func applyClassCancel(ctx context.Context, tx *sql.Tx, ids func() string, now time.Time, class domain.Class) ([]domain.Notice, error) {
	bookings, err := loadBooked(ctx, tx, class.ID)
	if err != nil {
		return nil, err
	}
	result, derr := domain.CancelClass(class, bookings, ids)
	if derr != nil {
		return nil, derr
	}
	if err := updateClass(ctx, tx, result.Class); err != nil {
		return nil, err
	}
	for _, booking := range result.Bookings {
		if err := updateBooking(ctx, tx, now, booking); err != nil {
			return nil, err
		}
	}
	for _, event := range result.Events {
		if err := insertEvent(ctx, tx, now, event); err != nil {
			return nil, err
		}
	}
	if _, err := tx.ExecContext(ctx, `
		update balance_notices set state = 'skipped' where class_id = ? and state = 'pending'`, class.ID); err != nil {
		return nil, err
	}
	return result.Notices, nil
}

func listAbsences(ctx context.Context, q querier, trainerID string) ([]Absence, error) {
	rows, err := q.QueryContext(ctx, `
		select id, starts_on, ends_on from trainer_absences
		where trainer_id = ? order by starts_on, id`, trainerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Absence{}
	for rows.Next() {
		var item Absence
		if err := rows.Scan(&item.ID, &item.StartsOn, &item.EndsOn); err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func (d *DB) ActiveTrainerID(ctx context.Context, telegramID int64) (string, error) {
	var id string
	err := d.sql.QueryRowContext(ctx, `
		select id from trainers where telegram_user_id = ? and active = 1`, strconv.FormatInt(telegramID, 10)).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil
	}
	return id, err
}

func (d *DB) PatchOwnTrainer(ctx context.Context, id, about, phone string) (Trainer, error) {
	about = strings.TrimSpace(about)
	phone = strings.TrimSpace(phone)
	if len([]rune(about)) > 500 {
		return Trainer{}, domain.Fail(domain.Validation)
	}
	if err := domain.ValidatePhone(phone); err != nil {
		return Trainer{}, err
	}
	var item Trainer
	err := d.write(ctx, func(tx *sql.Tx) error {
		current, err := trainerByID(ctx, tx, id)
		if err != nil {
			return missing(err, domain.Forbidden)
		}
		if !current.Active {
			return domain.Fail(domain.Forbidden)
		}
		current.About = about
		current.Phone = phone
		if err := saveTrainer(ctx, tx, current); err != nil {
			return err
		}
		item = current
		return nil
	})
	return item, err
}

func (d *DB) TrainerSchedule(ctx context.Context, from, to, trainerID string) ([]ClassView, error) {
	all, err := d.Schedule(ctx, from, to, true)
	if err != nil {
		return nil, err
	}
	out := []ClassView{}
	for _, item := range all {
		if item.Trainer.ID == trainerID {
			out = append(out, item)
			continue
		}
		if item.Kind == domain.KindHold && item.Status != nil && *item.Status == domain.Scheduled {
			out = append(out, item)
		}
	}
	return out, nil
}

func (d *DB) TrainerCancelClass(ctx context.Context, now time.Time, trainerID, classID string) (ClassChange, error) {
	var change ClassChange
	err := d.write(ctx, func(tx *sql.Tx) error {
		class, err := loadClass(ctx, tx, classID)
		if err != nil {
			return missing(err, domain.ClassNotFound)
		}
		if class.TrainerID != trainerID {
			return domain.Fail(domain.ClassNotFound)
		}
		if !class.StartsAt.After(now) {
			return domain.Fail(domain.ClassStarted)
		}
		notices, err := applyClassCancel(ctx, tx, d.ids, now, class)
		if err != nil {
			return err
		}
		view, err := loadView(ctx, tx, classID, true)
		if err != nil {
			return err
		}
		notes, err := outbound(ctx, tx, notices)
		if err != nil {
			return err
		}
		change = ClassChange{Class: view, Notices: notes}
		return nil
	})
	return change, err
}

func (d *DB) BookingClass(ctx context.Context, bookingID string) (string, error) {
	booking, err := loadBooking(ctx, d.sql, bookingID)
	if err != nil {
		return "", missing(err, domain.BookingNotFound)
	}
	return booking.ClassID, nil
}

func (d *DB) OwnClass(ctx context.Context, trainerID, classID string) error {
	class, err := loadClass(ctx, d.sql, classID)
	if err != nil {
		return missing(err, domain.ClassNotFound)
	}
	if class.TrainerID != trainerID || class.Status != domain.Scheduled {
		return domain.Fail(domain.ClassNotFound)
	}
	return nil
}

func (d *DB) TrainerStats(ctx context.Context, trainerID, from, to string) (TrainerStats, error) {
	start, end, err := dayBounds(from, to)
	if err != nil {
		return TrainerStats{}, err
	}
	stats := TrainerStats{From: from, To: to}
	if err := d.sql.QueryRowContext(ctx, `
		select count(distinct b.client_id)
		from attendances a
		join bookings b on b.id = a.booking_id
		join classes c on c.id = b.class_id
		where a.active = 1 and c.trainer_id = ? and c.starts_at >= ? and c.starts_at < ?`,
		trainerID, start, end).Scan(&stats.Clients); err != nil {
		return TrainerStats{}, err
	}
	if err := d.sql.QueryRowContext(ctx, `
		select count(*)
		from credit_entries e
		join bookings b on b.id = e.booking_id
		join classes c on c.id = b.class_id
		where e.reason = 'attend' and c.trainer_id = ? and c.starts_at >= ? and c.starts_at < ?`,
		trainerID, start, end).Scan(&stats.Attends); err != nil {
		return TrainerStats{}, err
	}
	return stats, nil
}

func (d *DB) PublicTrainers(ctx context.Context) ([]PublicTrainer, error) {
	rows, err := d.sql.QueryContext(ctx, `
		select id, name, about, phone from trainers where active = 1 order by name, id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []PublicTrainer{}
	for rows.Next() {
		var item PublicTrainer
		if err := rows.Scan(&item.ID, &item.Name, &item.About, &item.Phone); err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return d.attachTrainerClasses(ctx, out)
}

func (d *DB) PublicTrainer(ctx context.Context, id string) (PublicTrainer, error) {
	var item PublicTrainer
	err := d.sql.QueryRowContext(ctx, `
		select id, name, about, phone from trainers where id = ? and active = 1`, id).
		Scan(&item.ID, &item.Name, &item.About, &item.Phone)
	if errors.Is(err, sql.ErrNoRows) {
		return PublicTrainer{}, domain.FailMessage(domain.Validation, "Тренер не найден")
	}
	if err != nil {
		return PublicTrainer{}, err
	}
	list, err := d.attachTrainerClasses(ctx, []PublicTrainer{item})
	if err != nil {
		return PublicTrainer{}, err
	}
	if len(list) == 0 {
		return PublicTrainer{}, domain.FailMessage(domain.Validation, "Тренер не найден")
	}
	return list[0], nil
}

func (d *DB) attachTrainerClasses(ctx context.Context, trainers []PublicTrainer) ([]PublicTrainer, error) {
	if len(trainers) == 0 {
		return trainers, nil
	}
	rows, err := d.sql.QueryContext(ctx, `
		select trainer_id, title from class_templates where active = 1 order by title, id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	titles := map[string][]string{}
	seen := map[string]map[string]bool{}
	for rows.Next() {
		var trainerID, title string
		if err := rows.Scan(&trainerID, &title); err != nil {
			return nil, err
		}
		if seen[trainerID] == nil {
			seen[trainerID] = map[string]bool{}
		}
		if seen[trainerID][title] {
			continue
		}
		seen[trainerID][title] = true
		titles[trainerID] = append(titles[trainerID], title)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	for i := range trainers {
		trainers[i].Classes = titles[trainers[i].ID]
		if trainers[i].Classes == nil {
			trainers[i].Classes = []string{}
		}
	}
	return trainers, nil
}
