package store

import "context"

// Порядок совпадает с web/src/lib/color.ts. Новый тип получает первый свободный
// цвет и хранит его. Скрытие или удаление другого типа этот цвет не меняет.
var typePalette = []string{
	"#1d4ed8", "#c2410c", "#047857", "#7c3aed", "#be123c", "#a16207", "#0e7490", "#4d7c0f",
}

// Цвета известных названий. Новое название получает первый свободный цвет палитры.
var keptTypeColors = map[string]string{
	"K-POP KIDS (Dance)": "#1d4ed8",
	"HIGH HEELS":         "#c2410c",
	"Aero Stretching":    "#be123c",
	"K-POP 16+":          "#7c3aed",
	"DYNAMIC STRETCHING": "#0e7490",
	"Aero Yoga":          "#4d7c0f",
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
