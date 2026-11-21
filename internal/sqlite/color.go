package sqlite

import "context"

// Порядок совпадает с web/src/lib/color.ts. Новый тип получает первый свободный
// цвет и хранит его. Скрытие или удаление другого типа этот цвет не меняет.
var typePalette = []string{
	"#1d4ed8", "#c2410c", "#047857", "#7c3aed", "#be123c", "#a16207", "#0e7490", "#4d7c0f",
}

// Старые цвета слишком близки друг к другу. При запуске они заменяются на новую палитру.
var paletteUpgrade = map[string]string{
	"#3a7ca5": "#1d4ed8",
	"#d46a3a": "#c2410c",
	"#2f8f6b": "#047857",
	"#7a5ea7": "#7c3aed",
	"#c43d5a": "#be123c",
	"#b8860b": "#a16207",
	"#3d6b8c": "#0e7490",
	"#5c7a3a": "#4d7c0f",
}

// Цвета известных названий. Нужны только чтобы заполнить старые строки без цвета.
// Новое название сюда не добавляется: ему хватает свободного цвета палитры.
var keptTypeColors = map[string]string{
	"K-POP KIDS (Dance)": "#1d4ed8",
	"HIGH HEELS":         "#c2410c",
	"Aero Stretching":    "#be123c",
	"K-POP 16+":          "#7c3aed",
	"DYNAMIC STRETCHING": "#0e7490",
	"Aero Yoga":          "#4d7c0f",
}

func ensureTrainerColumns(ctx context.Context, q querier) error {
	for _, column := range []struct{ name, ddl string }{
		{"about", `alter table trainers add column about text not null default ''`},
		{"phone", `alter table trainers add column phone text not null default ''`},
	} {
		var found int
		if err := q.QueryRowContext(ctx, `select count(*) from pragma_table_info('trainers') where name = ?`, column.name).Scan(&found); err != nil {
			return err
		}
		if found == 0 {
			if _, err := q.ExecContext(ctx, column.ddl); err != nil {
				return err
			}
		}
	}
	return nil
}

func ensureTypeColors(ctx context.Context, q querier) error {
	var columns int
	if err := q.QueryRowContext(ctx, `select count(*) from pragma_table_info('training_types') where name = 'color'`).Scan(&columns); err != nil {
		return err
	}
	if columns == 0 {
		if _, err := q.ExecContext(ctx, `alter table training_types add column color text not null default ''`); err != nil {
			return err
		}
	}
	rows, err := q.QueryContext(ctx, `select id, title, color from training_types order by created_at, id`)
	if err != nil {
		return err
	}
	type blankType struct{ id, title string }
	type recolor struct{ id, color string }
	used := map[string]bool{}
	var blank []blankType
	var upgrades []recolor
	for rows.Next() {
		var id, title, color string
		if err := rows.Scan(&id, &title, &color); err != nil {
			rows.Close()
			return err
		}
		if next, ok := paletteUpgrade[color]; ok {
			upgrades = append(upgrades, recolor{id, next})
			color = next
		}
		if color != "" {
			used[color] = true
			continue
		}
		blank = append(blank, blankType{id, title})
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return err
	}
	rows.Close()
	for _, row := range upgrades {
		if err := setTypeColor(ctx, q, row.id, row.color); err != nil {
			return err
		}
	}
	var rest []blankType
	for _, row := range blank {
		color, ok := keptTypeColors[row.title]
		if !ok || used[color] {
			rest = append(rest, row)
			continue
		}
		if err := setTypeColor(ctx, q, row.id, color); err != nil {
			return err
		}
		used[color] = true
	}
	for _, row := range rest {
		color := freeTypeColor(used)
		if err := setTypeColor(ctx, q, row.id, color); err != nil {
			return err
		}
		used[color] = true
	}
	return nil
}

func setTypeColor(ctx context.Context, q querier, id, color string) error {
	_, err := q.ExecContext(ctx, `update training_types set color = ? where id = ?`, color, id)
	return err
}

func nextTypeColor(ctx context.Context, q querier, title string) (string, error) {
	rows, err := q.QueryContext(ctx, `select color from training_types where color != ''`)
	if err != nil {
		return "", err
	}
	defer rows.Close()
	used := map[string]bool{}
	for rows.Next() {
		var color string
		if err := rows.Scan(&color); err != nil {
			return "", err
		}
		used[color] = true
	}
	if err := rows.Err(); err != nil {
		return "", err
	}
	if color, ok := keptTypeColors[title]; ok && !used[color] {
		return color, nil
	}
	return freeTypeColor(used), nil
}

func freeTypeColor(used map[string]bool) string {
	for _, color := range typePalette {
		if !used[color] {
			return color
		}
	}
	return typePalette[len(used)%len(typePalette)]
}
