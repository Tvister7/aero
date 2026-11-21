package store

import (
	"context"
	"strings"
)

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
	found, err := columnCount(ctx, q, "studio", "guide")
	if err != nil {
		return err
	}
	if found == 0 {
		if _, err := q.ExecContext(ctx, `alter table studio add column guide text not null default ''`); err != nil {
			return err
		}
	}
	if err := ensurePricingColumn(ctx, q); err != nil {
		return err
	}
	var guide, pricing string
	if err := q.QueryRowContext(ctx, `select guide, pricing from studio where id = 1`).Scan(&guide, &pricing); err != nil {
		return err
	}
	if strings.TrimSpace(pricing) != "" && strings.TrimSpace(pricing) != "{}" {
		return nil
	}
	if guide == "" {
		guide = defaultGuide
	}
	_, err = q.ExecContext(ctx, `update studio set guide = ?, pricing = ? where id = 1`, guide, defaultPricing)
	return err
}

func ensurePricingColumn(ctx context.Context, q querier) error {
	found, err := columnCount(ctx, q, "studio", "pricing")
	if err != nil {
		return err
	}
	if found == 0 {
		if _, err := q.ExecContext(ctx, `alter table studio add column pricing jsonb not null default '{}'::jsonb`); err != nil {
			return err
		}
	} else {
		var dataType string
		if err := q.QueryRowContext(ctx, `
			select data_type from information_schema.columns
			where table_schema = current_schema() and table_name = ? and column_name = ?`,
			"studio", "pricing").Scan(&dataType); err != nil {
			return err
		}
		if dataType == "text" {
			if _, err := q.ExecContext(ctx, `
				alter table studio
				  alter column pricing drop default,
				  alter column pricing type jsonb using case when btrim(pricing) = '' then '{}'::jsonb else pricing::jsonb end,
				  alter column pricing set default '{}'::jsonb,
				  alter column pricing set not null`); err != nil {
				return err
			}
		}
	}
	var ready bool
	if err := q.QueryRowContext(ctx, `
		select exists (
			select 1 from pg_constraint
			where conname = 'studio_pricing_check' and conrelid = 'studio'::regclass
		)`).Scan(&ready); err != nil {
		return err
	}
	if ready {
		return nil
	}
	_, err = q.ExecContext(ctx, `alter table studio add constraint studio_pricing_check check (studio_pricing_ok(pricing))`)
	return err
}
