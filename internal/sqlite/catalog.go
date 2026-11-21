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
	rows, err := d.sql.QueryContext(ctx, `select id, name, active, about, phone, telegram_user_id from trainers order by name, id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Trainer{}
	for rows.Next() {
		item, err := scanTrainer(rows)
		if err != nil {
			return nil, err
		}
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
		_, err := tx.ExecContext(ctx, `insert into trainers (id, name, active, about, phone, created_at) values (?, ?, 1, '', '', ?)`, item.ID, item.Name, utc(now))
		return err
	})
	return item, err
}

func (d *DB) PatchTrainer(ctx context.Context, id string, patch TrainerFields) (Trainer, error) {
	var item Trainer
	err := d.write(ctx, func(tx *sql.Tx) error {
		current, err := applyTrainerPatch(ctx, tx, id, patch)
		if err != nil {
			return err
		}
		if err := saveTrainer(ctx, tx, current); err != nil {
			return err
		}
		item = current
		return nil
	})
	return item, err
}

func applyTrainerPatch(ctx context.Context, q querier, id string, patch TrainerFields) (Trainer, error) {
	current, err := trainerByID(ctx, q, id)
	if err != nil {
		return Trainer{}, missing(err, domain.Validation)
	}
	if patch.Name != nil {
		current.Name = strings.TrimSpace(*patch.Name)
		if current.Name == "" {
			return Trainer{}, domain.Fail(domain.Validation)
		}
	}
	if patch.Active != nil {
		current.Active = *patch.Active
	}
	if patch.About != nil {
		current.About = strings.TrimSpace(*patch.About)
		if len([]rune(current.About)) > 500 {
			return Trainer{}, domain.Fail(domain.Validation)
		}
	}
	if patch.Phone != nil {
		phone := strings.TrimSpace(*patch.Phone)
		if err := domain.ValidatePhone(phone); err != nil {
			return Trainer{}, err
		}
		current.Phone = phone
	}
	if patch.TelegramUserID != nil {
		current.TelegramUserID = strings.TrimSpace(*patch.TelegramUserID)
		if current.TelegramUserID != "" {
			if _, err := strconv.ParseInt(current.TelegramUserID, 10, 64); err != nil {
				return Trainer{}, domain.Fail(domain.Validation)
			}
			var known int
			if err := q.QueryRowContext(ctx, `
				select count(*) from clients
				where messenger_platform = 'telegram' and messenger_user_id = ?`, current.TelegramUserID).Scan(&known); err != nil {
				return Trainer{}, err
			}
			if known == 0 {
				return Trainer{}, domain.Fail(domain.Validation)
			}
		}
	}
	return current, nil
}

func saveTrainer(ctx context.Context, q querier, item Trainer) error {
	_, err := q.ExecContext(ctx, `
		update trainers set name = ?, active = ?, about = ?, phone = ?, telegram_user_id = ? where id = ?`,
		item.Name, bit(item.Active), item.About, item.Phone, nullable(item.TelegramUserID), item.ID)
	return err
}

func trainerByID(ctx context.Context, q querier, id string) (Trainer, error) {
	return scanTrainer(q.QueryRowContext(ctx, `
		select id, name, active, about, phone, telegram_user_id from trainers where id = ?`, id))
}

func scanTrainer(row scanner) (Trainer, error) {
	var item Trainer
	var active int
	var telegram sql.NullString
	err := row.Scan(&item.ID, &item.Name, &active, &item.About, &item.Phone, &telegram)
	if err != nil {
		return Trainer{}, err
	}
	item.Active = active == 1
	if telegram.Valid {
		item.TelegramUserID = telegram.String
	}
	return item, nil
}

func trainerName(ctx context.Context, q querier, id string) (string, error) {
	var name string
	err := q.QueryRowContext(ctx, `select name from trainers where id = ?`, id).Scan(&name)
	return name, err
}

func (d *DB) ListTypes(ctx context.Context) ([]TrainingType, error) {
	rows, err := d.sql.QueryContext(ctx, `
		select id, title, duration_minutes, capacity, price_min, price_max, active, color
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
		item.Color, err = nextTypeColor(ctx, tx, item.Title)
		if err != nil {
			return err
		}
		_, err = tx.ExecContext(ctx, `
			insert into training_types (id, title, duration_minutes, capacity, price_min, price_max, active, color, created_at)
			values (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			item.ID, item.Title, item.DurationMinutes, item.Capacity, item.PriceMin, item.PriceMax, bit(item.Active), item.Color, utc(now))
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
		select id, title, duration_minutes, capacity, price_min, price_max, active, color
		from training_types where id = ?`, id))
}

func scanType(row scanner) (TrainingType, error) {
	var item TrainingType
	var active int
	err := row.Scan(&item.ID, &item.Title, &item.DurationMinutes, &item.Capacity, &item.PriceMin, &item.PriceMax, &active, &item.Color)
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
	err := d.sql.QueryRowContext(ctx, `
		select address, phone, instagram_url, contact_url, map_url, guide, pricing from studio where id = 1`).
		Scan(&studio.Address, &studio.Phone, &studio.InstagramURL, &studio.ContactURL, &studio.MapURL, &studio.Guide, &studio.Pricing)
	return studio, err
}

func (d *DB) PutStudio(ctx context.Context, studio Studio) (Studio, error) {
	guide, err := domain.NormalizeGuide(studio.Guide)
	if err != nil {
		return Studio{}, err
	}
	pricing, err := domain.NormalizePricing(studio.Pricing)
	if err != nil {
		return Studio{}, err
	}
	studio.Guide = guide
	studio.Pricing = pricing
	writeErr := d.write(ctx, func(tx *sql.Tx) error {
		_, err := tx.ExecContext(ctx, `
			update studio set address = ?, phone = ?, instagram_url = ?, contact_url = ?, map_url = ?, guide = ?, pricing = ? where id = 1`,
			studio.Address, studio.Phone, studio.InstagramURL, studio.ContactURL, studio.MapURL, studio.Guide, studio.Pricing)
		return err
	})
	return studio, writeErr
}
