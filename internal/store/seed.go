package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/Tvister7/aero/internal/domain"
)

type SeedCounts struct {
	Trainers  int
	Types     int
	Templates int
}

func ImportFile(ctx context.Context, db *DB, path string, now time.Time) (SeedCounts, error) {
	text, err := os.ReadFile(path)
	if err != nil {
		return SeedCounts{}, err
	}
	return ImportSQL(ctx, db, string(text), now)
}

func ImportSQL(ctx context.Context, db *DB, text string, now time.Time) (SeedCounts, error) {
	trainers, err := tuples(section(text, "insert into public.trainers", "on conflict"))
	if err != nil {
		return SeedCounts{}, err
	}
	types, err := tuples(section(text, "insert into public.training_types", "on conflict"))
	if err != nil {
		return SeedCounts{}, err
	}
	templates, err := tuples(section(text, "insert into public.class_templates", "on conflict"))
	if err != nil {
		return SeedCounts{}, err
	}
	var counts SeedCounts
	err = db.write(ctx, func(tx querier) error {
		trainerIDs := map[string]string{}
		for _, row := range trainers {
			if len(row) < 2 {
				return fmt.Errorf("trainer row")
			}
			name := row[0]
			id, err := upsertTrainer(ctx, tx, db.ids, now, name, row[1] == "true")
			if err != nil {
				return err
			}
			trainerIDs[name] = id
			counts.Trainers++
		}
		typeIDs := map[string]string{}
		for _, row := range types {
			if len(row) < 7 {
				return fmt.Errorf("type row")
			}
			duration, err := strconv.Atoi(row[2])
			if err != nil {
				return err
			}
			capacity, err := strconv.Atoi(row[3])
			if err != nil {
				return err
			}
			id, err := upsertType(ctx, tx, db.ids, now, row[0], duration, capacity, row[4], row[5], row[6] == "true")
			if err != nil {
				return err
			}
			typeIDs[row[0]] = id
			counts.Types++
		}
		for _, row := range templates {
			if len(row) < 6 {
				return fmt.Errorf("template row")
			}
			trainerID := trainerIDs[row[0]]
			typeID := typeIDs[row[1]]
			if trainerID == "" || typeID == "" {
				return fmt.Errorf("template %s / %s has no trainer or type", row[0], row[1])
			}
			weekday, err := strconv.Atoi(row[2])
			if err != nil {
				return err
			}
			capacity, err := strconv.Atoi(row[5])
			if err != nil {
				return err
			}
			if err := upsertTemplate(ctx, tx, db.ids, now, trainerID, typeID, row[1], weekday, row[3], capacity); err != nil {
				return err
			}
			counts.Templates++
		}
		return ensureTypeColors(ctx, tx)
	})
	return counts, err
}

func section(text, start, end string) string {
	i := strings.Index(text, start)
	if i < 0 {
		return ""
	}
	rest := text[i:]
	j := strings.Index(rest, end)
	if j < 0 {
		return rest
	}
	return rest[:j]
}

func tuples(block string) ([][]string, error) {
	var out [][]string
	for _, line := range strings.Split(block, "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "(") {
			continue
		}
		fields, err := splitTuple(line)
		if err != nil {
			return nil, err
		}
		out = append(out, fields)
	}
	return out, nil
}

func splitTuple(line string) ([]string, error) {
	line = strings.TrimSpace(line)
	line = strings.TrimSuffix(line, ",")
	if len(line) < 2 || line[0] != '(' || line[len(line)-1] != ')' {
		return nil, fmt.Errorf("tuple %s", line)
	}
	inner := line[1 : len(line)-1]
	var fields []string
	var b strings.Builder
	inQuote := false
	for i := 0; i < len(inner); i++ {
		char := inner[i]
		if inQuote {
			if char == '\'' {
				if i+1 < len(inner) && inner[i+1] == '\'' {
					b.WriteByte('\'')
					i++
					continue
				}
				inQuote = false
				continue
			}
			b.WriteByte(char)
			continue
		}
		switch char {
		case '\'':
			inQuote = true
		case ',':
			fields = append(fields, strings.TrimSpace(b.String()))
			b.Reset()
		default:
			b.WriteByte(char)
		}
	}
	fields = append(fields, strings.TrimSpace(b.String()))
	return fields, nil
}

func upsertTrainer(ctx context.Context, q querier, ids func() string, now time.Time, name string, active bool) (string, error) {
	var id string
	err := q.QueryRowContext(ctx, `select id from trainers where name = ?`, name).Scan(&id)
	if err == nil {
		_, err = q.ExecContext(ctx, `update trainers set active = ? where id = ?`, active, id)
		return id, err
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return "", err
	}
	id = ids()
	_, err = q.ExecContext(ctx, `insert into trainers (id, name, active, created_at) values (?, ?, ?, ?)`, id, name, active, utc(now))
	return id, err
}

func upsertType(ctx context.Context, q querier, ids func() string, now time.Time, title string, duration, capacity int, priceMin, priceMax string, active bool) (string, error) {
	item, err := normalizeType(TrainingType{
		Title: title, DurationMinutes: duration, Capacity: capacity, PriceMin: priceMin, PriceMax: priceMax, Active: active,
	})
	if err != nil {
		return "", err
	}
	var id string
	found := q.QueryRowContext(ctx, `select id from training_types where title = ?`, item.Title).Scan(&id)
	if found != nil && !errors.Is(found, sql.ErrNoRows) {
		return "", found
	}
	if found == nil {
		_, err = q.ExecContext(ctx, `
			update training_types set duration_minutes = ?, capacity = ?, price_min = ?, price_max = ?, active = ? where id = ?`,
			item.DurationMinutes, item.Capacity, item.PriceMin, item.PriceMax, item.Active, id)
		return id, err
	}
	id = ids()
	_, err = q.ExecContext(ctx, `
		insert into training_types (id, title, duration_minutes, capacity, price_min, price_max, active, created_at)
		values (?, ?, ?, ?, ?, ?, ?, ?)`,
		id, item.Title, item.DurationMinutes, item.Capacity, item.PriceMin, item.PriceMax, item.Active, utc(now))
	return id, err
}

func upsertTemplate(ctx context.Context, q querier, ids func() string, now time.Time, trainerID, typeID, title string, weekday int, start string, capacity int) error {
	hour, minute, clockErr := domain.ParseClock(start)
	if clockErr != nil {
		return clockErr
	}
	clockText := clock(hour, minute)
	var id string
	found := q.QueryRowContext(ctx, `
		select id from class_templates where trainer_id = ? and title = ? and weekday = ? and start_time = ?`,
		trainerID, title, weekday, clockText).Scan(&id)
	if found != nil && !errors.Is(found, sql.ErrNoRows) {
		return found
	}
	if found == nil {
		_, err := q.ExecContext(ctx, `
			update class_templates set training_type_id = ?, capacity = ?, active = true where id = ?`, typeID, capacity, id)
		return err
	}
	_, err := q.ExecContext(ctx, `
		insert into class_templates (id, trainer_id, training_type_id, title, weekday, start_time, capacity, active, created_at)
		values (?, ?, ?, ?, ?, ?, ?, true, ?)`,
		ids(), trainerID, typeID, title, weekday, clockText, capacity, utc(now))
	return err
}
