package telegram

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"time"

	"github.com/Tvister7/aero/internal/domain"
)

var errTelegram = errors.New("telegram getUpdates")

func IsStart(text string) bool {
	fields := strings.Fields(strings.TrimSpace(text))
	if len(fields) == 0 {
		return false
	}
	command := fields[0]
	if at := strings.IndexByte(command, '@'); at >= 0 {
		command = command[:at]
	}
	return command == "/start"
}

// Listen забирает новые сообщения и отвечает на /start.
// text читает текущий текст приветствия. origin с https становится кнопкой Mini App.
// Остальные команды бот пропускает.
func (b *Bot) Listen(ctx context.Context, origin string, text func(context.Context) (string, error)) {
	if b == nil || strings.TrimSpace(b.token) == "" {
		return
	}
	origin = strings.TrimRight(strings.TrimSpace(origin), "/")
	offset := 0
	for {
		if ctx.Err() != nil {
			return
		}
		next, err := b.pollOnce(ctx, offset, origin, text)
		if ctx.Err() != nil {
			return
		}
		if err != nil {
			slog.Error("telegram", "err", err.Error())
			select {
			case <-ctx.Done():
				return
			case <-time.After(2 * time.Second):
			}
			continue
		}
		offset = next
	}
}

func (b *Bot) pollOnce(ctx context.Context, offset int, origin string, text func(context.Context) (string, error)) (int, error) {
	var response struct {
		OK     bool `json:"ok"`
		Result []struct {
			UpdateID int `json:"update_id"`
			Message  *struct {
				Text string `json:"text"`
				Chat struct {
					ID int64 `json:"id"`
				} `json:"chat"`
			} `json:"message"`
		} `json:"result"`
	}
	err := b.call(ctx, b.poll, "getUpdates", map[string]any{
		"offset":          offset,
		"timeout":         25,
		"allowed_updates": []string{"message"},
	}, &response)
	if err != nil {
		return offset, err
	}
	if !response.OK {
		return offset, errTelegram
	}
	next := offset
	for _, item := range response.Result {
		if item.UpdateID+1 > next {
			next = item.UpdateID + 1
		}
		if item.Message == nil || !IsStart(item.Message.Text) {
			continue
		}
		body := ""
		if text != nil {
			var err error
			body, err = text(ctx)
			if err != nil {
				slog.Error("telegram", "err", err.Error())
				body = ""
			}
		}
		if err := b.welcome(ctx, item.Message.Chat.ID, origin, body); err != nil {
			slog.Error("telegram", "err", err.Error())
		}
	}
	return next, nil
}

func (b *Bot) welcome(ctx context.Context, chatID int64, origin, text string) error {
	withButton := strings.HasPrefix(origin, "https://")
	payload := map[string]any{"chat_id": chatID, "text": domain.WelcomeBody(text, withButton)}
	if withButton {
		payload["reply_markup"] = map[string]any{
			"inline_keyboard": []any{
				[]any{map[string]any{
					"text":    "Открыть приложение",
					"web_app": map[string]string{"url": origin},
				}},
			},
		}
	}
	return b.call(ctx, b.client, "sendMessage", payload, nil)
}
