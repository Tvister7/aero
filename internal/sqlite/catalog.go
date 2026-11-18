package sqlite

import (
	"context"
	"database/sql"
	"strconv"
	"strings"
	"time"

	"github.com/Tvister7/aero/internal/domain"
)

func (d *DB) ListTrainers(ctx context.Context) ([]Trainer, error) {
	rows, err := d.sql.QueryContext(ctx, `select id, name, active from trainers order by name, id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Trainer{}
	for rows.Next() {
		var item Trainer
		var active int
		if err := rows.Scan(&item.ID, &item.Name, &active); err != nil {
			return nil, err
		}
		item.Active = active == 1
		out = append(out, item)
	}
	return out, rows.Err()
}

func (d *DB) CreateTrainer(ctx context.Context, now time.Time, name string) (Trainer, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return Trainer{}, domain.Fail(domain.Validation)
	}
	item := Trainer{ID: d.ids(), Name: name, Active: true}
	err := d.write(ctx, func(tx *sql.Tx) error {
		_, err := tx.ExecContext(ctx, `insert into trainers (id, name, active, created_at) values (?, ?, 1, ?)`, item.ID, item.Name, utc(now))
		return err
	})
	return item, err
}

func (d *DB) PatchTrainer(ctx context.Context, id string, name *string, active *bool) (Trainer, error) {
	var item Trainer
	err := d.write(ctx, func(tx *sql.Tx) error {
		current, err := d.trainer(ctx, tx, id)
		if err != nil {
			return missing(err, domain.Validation)
		}
		if name != nil {
			current.Name = strings.TrimSpace(*name)
			if current.Name == "" {
				return domain.Fail(domain.Validation)
			}
		}
		if active != nil {
			current.Active = *active
		}
		_, err = tx.ExecContext(ctx, `update trainers set name = ?, active = ? where id = ?`, current.Name, bit(current.Active), id)
		item = current
		return err
	})
	return item, err
}

func (d *DB) trainer(ctx context.Context, q querier, id string) (Trainer, error) {
	var item Trainer
	var active int
	err := q.QueryRowContext(ctx, `select id, name, active from trainers where id = ?`, id).Scan(&item.ID, &item.Name, &active)
	item.Active = active == 1
	return item, err
}

func trainerName(ctx context.Context, q querier, id string) (string, error) {
	var name string
	err := q.QueryRowContext(ctx, `select name from trainers where id = ?`, id).Scan(&name)
	return name, err
}

func (d *DB) ListTypes(ctx context.Context) ([]TrainingType, error) {
	rows, err := d.sql.QueryContext(ctx, `
		select id, title, duration_minutes, capacity, price_min, price_max, active
		from training_types order by title, id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []TrainingType{}
	for rows.Next() {
		item, err := scanType(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func (d *DB) CreateType(ctx context.Context, now time.Time, in TypeInput) (TrainingType, error) {
	if in.Title == nil || in.DurationMinutes == nil || in.Capacity == nil || in.PriceMin == nil || in.PriceMax == nil {
		return TrainingType{}, domain.Fail(domain.Validation)
	}
	item, err := normalizeType(TrainingType{
		ID: d.ids(), Title: *in.Title, DurationMinutes: *in.DurationMinutes, Capacity: *in.Capacity,
		PriceMin: *in.PriceMin, PriceMax: *in.PriceMax, Active: in.Active == nil || *in.Active,
	})
	if err != nil {
		return TrainingType{}, err
	}
	err = d.write(ctx, func(tx *sql.Tx) error {
		_, err := tx.ExecContext(ctx, `
			insert into training_types (id, title, duration_minutes, capacity, price_min, price_max, active, created_at)
			values (?, ?, ?, ?, ?, ?, ?, ?)`,
			item.ID, item.Title, item.DurationMinutes, item.Capacity, item.PriceMin, item.PriceMax, bit(item.Active), utc(now))
		return err
	})
	return item, err
}

func (d *DB) PatchType(ctx context.Context, id string, in TypeInput) (TrainingType, error) {
	var item TrainingType
	err := d.write(ctx, func(tx *sql.Tx) error {
		current, err := loadType(ctx, tx, id)
		if err != nil {
			return missing(err, domain.Validation)
		}
		if in.Title != nil {
			current.Title = *in.Title
		}
		if in.DurationMinutes != nil {
			current.DurationMinutes = *in.DurationMinutes
		}
		if in.Capacity != nil {
			current.Capacity = *in.Capacity
		}
		if in.PriceMin != nil {
			current.PriceMin = *in.PriceMin
		}
		if in.PriceMax != nil {
			current.PriceMax = *in.PriceMax
		}
		if in.Active != nil {
			current.Active = *in.Active
		}
		current, err = normalizeType(current)
		if err != nil {
			return err
		}
		_, err = tx.ExecContext(ctx, `
			update training_types
			set title = ?, duration_minutes = ?, capacity = ?, price_min = ?, price_max = ?, active = ?
			where id = ?`,
			current.Title, current.DurationMinutes, current.Capacity, current.PriceMin, current.PriceMax, bit(current.Active), id)
		item = current
		return err
	})
	return item, err
}

func loadType(ctx context.Context, q querier, id string) (TrainingType, error) {
	return scanType(q.QueryRowContext(ctx, `
		select id, title, duration_minutes, capacity, price_min, price_max, active
		from training_types where id = ?`, id))
}

func scanType(row scanner) (TrainingType, error) {
	var item TrainingType
	var active int
	err := row.Scan(&item.ID, &item.Title, &item.DurationMinutes, &item.Capacity, &item.PriceMin, &item.PriceMax, &active)
	item.Active = active == 1
	return item, err
}

func normalizeType(item TrainingType) (TrainingType, error) {
	item.Title = strings.TrimSpace(item.Title)
	if item.Title == "" || item.DurationMinutes < 1 || item.DurationMinutes > 600 || item.Capacity < 1 {
		return TrainingType{}, domain.Fail(domain.Validation)
	}
	min, max, err := domain.NormalizePrice(item.PriceMin, item.PriceMax)
	if err != nil {
		return TrainingType{}, err
	}
	item.PriceMin, item.PriceMax = min, max
	return item, nil
}

func (d *DB) ListTemplates(ctx context.Context) ([]Template, error) {
	rows, err := d.sql.QueryContext(ctx, `
		select id, trainer_id, training_type_id, title, weekday, start_time, capacity, active
		from class_templates order by weekday, start_time, title`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Template{}
	for rows.Next() {
		item, err := scanTemplate(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func (d *DB) CreateTemplate(ctx context.Context, now time.Time, in TemplateInput) (Template, error) {
	if in.TrainerID == nil || in.TrainingTypeID == nil || in.Weekday == nil || in.StartTime == nil {
		return Template{}, domain.Fail(domain.Validation)
	}
	var item Template
	err := d.write(ctx, func(tx *sql.Tx) error {
		kind, err := loadType(ctx, tx, *in.TrainingTypeID)
		if err != nil {
			return missing(err, domain.Validation)
		}
		if _, err := trainerName(ctx, tx, *in.TrainerID); err != nil {
			return missing(err, domain.Validation)
		}
		capacity := kind.Capacity
		if in.Capacity != nil {
			capacity = *in.Capacity
		}
		active := true
		if in.Active != nil {
			active = *in.Active
		}
		item = Template{
			ID: d.ids(), TrainerID: *in.TrainerID, TrainingTypeID: kind.ID, Title: kind.Title,
			Weekday: *in.Weekday, StartTime: *in.StartTime, Capacity: capacity, Active: active,
		}
		if err := validateTemplate(&item); err != nil {
			return err
		}
		_, err = tx.ExecContext(ctx, `
			insert into class_templates (id, trainer_id, training_type_id, title, weekday, start_time, capacity, active, created_at)
			values (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			item.ID, item.TrainerID, item.TrainingTypeID, item.Title, item.Weekday, item.StartTime, item.Capacity, bit(item.Active), utc(now))
		return err
	})
	return item, err
}

func (d *DB) PatchTemplate(ctx context.Context, id string, in TemplateInput) (Template, error) {
	var item Template
	err := d.write(ctx, func(tx *sql.Tx) error {
		current, err := loadTemplate(ctx, tx, id)
		if err != nil {
			return missing(err, domain.Validation)
		}
		if in.TrainerID != nil {
			if _, err := trainerName(ctx, tx, *in.TrainerID); err != nil {
				return missing(err, domain.Validation)
			}
			current.TrainerID = *in.TrainerID
		}
		if in.TrainingTypeID != nil {
			kind, err := loadType(ctx, tx, *in.TrainingTypeID)
			if err != nil {
				return missing(err, domain.Validation)
			}
			current.TrainingTypeID = kind.ID
			current.Title = kind.Title
		}
		if in.Weekday != nil {
			current.Weekday = *in.Weekday
		}
		if in.StartTime != nil {
			current.StartTime = *in.StartTime
		}
		if in.Capacity != nil {
			current.Capacity = *in.Capacity
		}
		if in.Active != nil {
			current.Active = *in.Active
		}
		if err := validateTemplate(&current); err != nil {
			return err
		}
		_, err = tx.ExecContext(ctx, `
			update class_templates
			set trainer_id = ?, training_type_id = ?, title = ?, weekday = ?, start_time = ?, capacity = ?, active = ?
			where id = ?`,
			current.TrainerID, current.TrainingTypeID, current.Title, current.Weekday, current.StartTime, current.Capacity, bit(current.Active), id)
		item = current
		return err
	})
	return item, err
}

func loadTemplate(ctx context.Context, q querier, id string) (Template, error) {
	return scanTemplate(q.QueryRowContext(ctx, `
		select id, trainer_id, training_type_id, title, weekday, start_time, capacity, active
		from class_templates where id = ?`, id))
}

func scanTemplate(row scanner) (Template, error) {
	var item Template
	var active int
	err := row.Scan(&item.ID, &item.TrainerID, &item.TrainingTypeID, &item.Title, &item.Weekday, &item.StartTime, &item.Capacity, &active)
	item.Active = active == 1
	return item, err
}

func validateTemplate(item *Template) *domain.Error {
	hour, minute, err := domain.ParseClock(item.StartTime)
	if err != nil || item.Weekday < 1 || item.Weekday > 7 || item.Capacity < 1 {
		return domain.Fail(domain.Validation)
	}
	item.StartTime = clock(hour, minute)
	return nil
}

func clock(hour, minute int) string {
	return fmtClock(hour) + ":" + fmtClock(minute)
}

func fmtClock(value int) string {
	text := strconv.Itoa(value)
	if value < 10 {
		return "0" + text
	}
	return text
}

func (d *DB) Studio(ctx context.Context) (Studio, error) {
	var studio Studio
	err := d.sql.QueryRowContext(ctx, `select address, phone, instagram_url, contact_url, map_url from studio where id = 1`).
		Scan(&studio.Address, &studio.Phone, &studio.InstagramURL, &studio.ContactURL, &studio.MapURL)
	return studio, err
}

func (d *DB) PutStudio(ctx context.Context, studio Studio) (Studio, error) {
	err := d.write(ctx, func(tx *sql.Tx) error {
		_, err := tx.ExecContext(ctx, `
			update studio set address = ?, phone = ?, instagram_url = ?, contact_url = ?, map_url = ? where id = 1`,
			studio.Address, studio.Phone, studio.InstagramURL, studio.ContactURL, studio.MapURL)
		return err
	})
	return studio, err
}
