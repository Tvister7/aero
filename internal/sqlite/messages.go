package sqlite

import (
	"context"
	"database/sql"

	"github.com/Tvister7/aero/internal/domain"
)

func (d *DB) Messages(ctx context.Context) ([]MessageTemplate, error) {
	stored, err := d.messageBodies(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]MessageTemplate, 0, len(domain.MessageCatalog()))
	for _, item := range domain.MessageCatalog() {
		body := item.Body
		if custom, ok := stored[item.Key]; ok && custom != "" {
			body = custom
		}
		out = append(out, MessageTemplate{Key: item.Key, Title: item.Title, Body: body})
	}
	return out, nil
}

func (d *DB) MessageBody(ctx context.Context, key string) (string, error) {
	item, ok := domain.MessageByKey(key)
	if !ok {
		return "", domain.FailMessage(domain.Validation, "Такого сообщения нет")
	}
	var body string
	err := d.sql.QueryRowContext(ctx, `select body from messages where key = ?`, key).Scan(&body)
	if err == sql.ErrNoRows || body == "" {
		return item.Body, nil
	}
	if err != nil {
		return "", err
	}
	return body, nil
}

func (d *DB) PutMessage(ctx context.Context, key, body string) (MessageTemplate, error) {
	item, ok := domain.MessageByKey(key)
	if !ok {
		return MessageTemplate{}, domain.FailMessage(domain.Validation, "Такого сообщения нет")
	}
	normalized, err := domain.NormalizeMessageBody(key, body)
	if err != nil {
		return MessageTemplate{}, err
	}
	writeErr := d.write(ctx, func(tx *sql.Tx) error {
		_, execErr := tx.ExecContext(ctx, `
			insert into messages (key, body) values (?, ?)
			on conflict (key) do update set body = excluded.body`, key, normalized)
		return execErr
	})
	if writeErr != nil {
		return MessageTemplate{}, writeErr
	}
	return MessageTemplate{Key: key, Title: item.Title, Body: normalized}, nil
}

func (d *DB) messageBodies(ctx context.Context) (map[string]string, error) {
	rows, err := d.sql.QueryContext(ctx, `select key, body from messages`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]string{}
	for rows.Next() {
		var key, body string
		if err := rows.Scan(&key, &body); err != nil {
			return nil, err
		}
		out[key] = body
	}
	return out, rows.Err()
}
