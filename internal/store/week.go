package store

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
	err := d.write(ctx, func(tx querier) error {
		templates, err := loadSlots(ctx, tx)
		if err != nil {
			return err
		}
		absences, err := loadAbsences(ctx, tx)
		if err != nil {
			return err
		}
		existing, known, err := loadWeek(ctx, tx, monday)
		if err != nil {
			return err
		}
		result := domain.GenerateWeek(now, monday, templates, existing, absences, d.ids, d.Exclusive)
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
		if err := enrollWeek(ctx, tx, d.ids, now, monday); err != nil {
			return err
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
		if err := rows.Scan(
			&slot.ID, &slot.TrainerID, &slot.TypeID, &slot.Title, &slot.Weekday, &slot.StartTime, &slot.Capacity,
			&slot.DurationMinutes, &slot.PriceMin, &slot.PriceMax, &slot.Active, &slot.TrainerActive,
		); err != nil {
			return nil, err
		}
		slot.StartTime, err = shownClock(slot.StartTime)
		if err != nil {
			return nil, err
		}
		out = append(out, slot)
	}
	return out, rows.Err()
}

func loadWeek(ctx context.Context, q querier, monday time.Time) ([]domain.ClassSnapshot, map[string]bool, error) {
	rows, err := q.QueryContext(ctx, `
		select id, template_id, trainer_id, training_type_id, title, starts_at, ends_at, capacity, price_min, price_max, status, kind
		from classes where starts_at < ? and ends_at > ?`, utc(monday.AddDate(0, 0, 7)), utc(monday))
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()
	var pending []domain.Class
	for rows.Next() {
		var class domain.Class
		var template, trainer, typeID sql.NullString
		if err := rows.Scan(&class.ID, &template, &trainer, &typeID, &class.Title, &class.StartsAt, &class.EndsAt, &class.Capacity, &class.PriceMin, &class.PriceMax, &class.Status, &class.Kind); err != nil {
			return nil, nil, err
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
		pending = append(pending, class)
	}
	if err := rows.Err(); err != nil {
		return nil, nil, err
	}
	if err := rows.Close(); err != nil {
		return nil, nil, err
	}
	snapshots := make([]domain.ClassSnapshot, 0, len(pending))
	known := map[string]bool{}
	for _, class := range pending {
		count, err := bookedCount(ctx, q, class.ID)
		if err != nil {
			return nil, nil, err
		}
		snapshots = append(snapshots, domain.ClassSnapshot{Class: class, Booked: count})
		known[class.ID] = true
	}
	return snapshots, known, nil
}
