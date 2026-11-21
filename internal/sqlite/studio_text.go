package sqlite

import "context"

const defaultGuide = `📍 Как нас найти?

Наша студия находится в жилом комплексе Green Budapest, ул. Панджикидзе 22.

Ориентир — магазин Nikora. За магазином, сразу за углом, находится вход в подъезд.

Над входом вы увидите букву C — это нужный подъезд.

Поднимитесь по лестнице на 2-й этаж.

Найдите дверь с табличкой aero.ge — вы на месте! 🤍

🔐 Как попасть в подъезд?

Вход открывается с помощью кода через маленький домофон у двери. Мы отправим вам актуальный код за пару часов до занятия, поскольку он обновляется каждые 24 часа.

До встречи на тренировке! ✨`

const defaultPricing = `{"base":{"1":35,"5":140,"8":210,"12":280},"personal":{"single":80,"split":130}}`

func ensureStudioColumns(ctx context.Context, q querier) error {
	for _, column := range []struct{ name, ddl string }{
		{"guide", `alter table studio add column guide text not null default ''`},
		{"pricing", `alter table studio add column pricing text not null default ''`},
	} {
		var found int
		if err := q.QueryRowContext(ctx, `select count(*) from pragma_table_info('studio') where name = ?`, column.name).Scan(&found); err != nil {
			return err
		}
		if found == 0 {
			if _, err := q.ExecContext(ctx, column.ddl); err != nil {
				return err
			}
		}
	}
	var guide, pricing string
	if err := q.QueryRowContext(ctx, `select guide, pricing from studio where id = 1`).Scan(&guide, &pricing); err != nil {
		return err
	}
	if pricing != "" {
		return nil
	}
	if guide == "" {
		guide = defaultGuide
	}
	_, err := q.ExecContext(ctx, `update studio set guide = ?, pricing = ? where id = 1`, guide, defaultPricing)
	return err
}
