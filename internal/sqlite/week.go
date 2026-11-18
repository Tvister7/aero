package sqlite

import (
	"context"
	"database/sql"
	"strings"
	"time"

	"github.com/Tvister7/aero/internal/domain"
)

func (d *DB) GenerateWeek(ctx context.Context, now time.Time, weekStart string) (WeekOutcome, error) {
	monday := domain.NextMonday(now)
	if strings.TrimSpace(weekStart) != "" {
		day, err := domain.ParseDay(strings.TrimSpace(weekStart))
		if err != nil {
			return WeekOutcome{}, err
		}
		monday = domain.MondayOf(day)
	}
	var outcome WeekOutcome
	err := d.write(ctx, func(tx *sql.Tx) error {
		templates, err := loadSlots(ctx, tx)
		if err != nil {
			return err
		}
		existing, known, err := loadWeek(ctx, tx, monday)
		if err != nil {
			return err
		}
		result := domain.GenerateWeek(now, monday, templates, existing, d.ids)
		for _, class := range result.Upserts {
			if known[class.ID] {
				if err := updateClass(ctx, tx, class); err != nil {
					return err
				}
				continue
			}
			if err := insertClass(ctx, tx, now, class); err != nil {
				return err
			}
		}
		outcome = WeekOutcome{
			WeekStart: domain.DateKey(monday),
			WeekEnd:   domain.DateKey(monday.AddDate(0, 0, 6)),
			Created:   result.Created,
			Updated:   result.Updated,
		}
		return nil
	})
	return outcome, err
}

func loadSlots(ctx context.Context, q querier) ([]domain.TemplateSlot, error) {
	rows, err := q.QueryContext(ctx, `
		select tpl.id, tpl.trainer_id, tpl.training_type_id, tpl.title, tpl.weekday, tpl.start_time, tpl.capacity,
		       ty.duration_minutes, ty.price_min, ty.price_max, tpl.active, tr.active
		from class_templates tpl
		join trainers tr on tr.id = tpl.trainer_id
		join training_types ty on ty.id = tpl.training_type_id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.TemplateSlot
	for rows.Next() {
		var slot domain.TemplateSlot
		var templateActive, trainerActive int
		if err := rows.Scan(
			&slot.ID, &slot.TrainerID, &slot.TypeID, &slot.Title, &slot.Weekday, &slot.StartTime, &slot.Capacity,
			&slot.DurationMinutes, &slot.PriceMin, &slot.PriceMax, &templateActive, &trainerActive,
		); err != nil {
			return nil, err
		}
		slot.Active = templateActive == 1
		slot.TrainerActive = trainerActive == 1
		out = append(out, slot)
	}
	return out, rows.Err()
}

func loadWeek(ctx context.Context, q querier, monday time.Time) ([]domain.ClassSnapshot, map[string]bool, error) {
	rows, err := q.QueryContext(ctx, `
		select id, template_id, trainer_id, training_type_id, title, starts_at, ends_at, capacity, price_min, price_max, status
		from classes where starts_at >= ? and starts_at < ?`, utc(monday), utc(monday.AddDate(0, 0, 7)))
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()
	var snapshots []domain.ClassSnapshot
	known := map[string]bool{}
	for rows.Next() {
		var class domain.Class
		var template sql.NullString
		var start, end string
		if err := rows.Scan(&class.ID, &template, &class.TrainerID, &class.TypeID, &class.Title, &start, &end, &class.Capacity, &class.PriceMin, &class.PriceMax, &class.Status); err != nil {
			return nil, nil, err
		}
		if template.Valid {
			class.TemplateID = template.String
		}
		if class.StartsAt, err = parseTime(start); err != nil {
			return nil, nil, err
		}
		if class.EndsAt, err = parseTime(end); err != nil {
			return nil, nil, err
		}
		count, err := bookedCount(ctx, q, class.ID)
		if err != nil {
			return nil, nil, err
		}
		snapshots = append(snapshots, domain.ClassSnapshot{Class: class, Booked: count})
		known[class.ID] = true
	}
	return snapshots, known, rows.Err()
}
