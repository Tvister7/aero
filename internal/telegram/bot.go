package telegram

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type Bot struct {
	token  string
	client *http.Client
	poll   *http.Client
	base   string
}

func NewBot(token string) *Bot {
	return &Bot{
		token:  token,
		client: &http.Client{Timeout: 10 * time.Second},
		poll:   &http.Client{Timeout: 40 * time.Second},
	}
}

func (b *Bot) Send(ctx context.Context, chatID int64, text string) error {
	return b.call(ctx, b.client, "sendMessage", map[string]any{"chat_id": chatID, "text": text}, nil)
}

func (b *Bot) endpoint(method string) string {
	base := b.base
	if base == "" {
		base = "https://api.telegram.org"
	}
	return base + "/bot" + b.token + "/" + method
}

func (b *Bot) call(ctx context.Context, client *http.Client, method string, payload any, dest any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, b.endpoint(method), bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return err
	}
	if resp.StatusCode >= 300 {
		return fmt.Errorf("telegram status %d", resp.StatusCode)
	}
	if dest != nil && json.Unmarshal(raw, dest) != nil {
		return fmt.Errorf("telegram response")
	}
	return nil
}
