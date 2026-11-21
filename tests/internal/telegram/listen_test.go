package telegram_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	. "github.com/Tvister7/aero/internal/telegram"
)

func TestIsStart(t *testing.T) {
	for _, text := range []string{"/start", "/start@aero_bot", "/start payload", "  /start@aero_bot code "} {
		if !IsStart(text) {
			t.Fatalf("want start %q", text)
		}
	}
	for _, text := range []string{"", "/starting", "/help", "start"} {
		if IsStart(text) {
			t.Fatalf("not start %q", text)
		}
	}
}

func TestListenStart(t *testing.T) {
	updates := `{"ok":true,"result":[{"update_id":7,"message":{"text":"/help","chat":{"id":1}}},{"update_id":8,"message":{"text":"/start@aero_bot","chat":{"id":42}}}]}`
	got := make(chan map[string]any, 1)
	polls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/getUpdates") {
			polls++
			if polls == 1 {
				_, _ = io.WriteString(w, updates)
				return
			}
			<-r.Context().Done()
			return
		}
		var body map[string]any
		if json.NewDecoder(r.Body).Decode(&body) != nil {
			t.Errorf("body")
		}
		got <- body
		_, _ = io.WriteString(w, `{"ok":true}`)
	}))
	defer server.Close()

	bot := NewBot("token")
	bot.APIRoot = server.URL
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan struct{})
	go func() {
		bot.Listen(ctx, "https://aero.example/", func(context.Context) (string, error) {
			return "Ждём вас в студии.", nil
		})
		close(done)
	}()

	var body map[string]any
	select {
	case body = <-got:
	case <-time.After(2 * time.Second):
		t.Fatal("no welcome")
	}
	cancel()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("listen did not stop")
	}
	if body["text"] != "Ждём вас в студии." || body["chat_id"] != float64(42) {
		t.Fatalf("message %#v", body)
	}
	raw, err := json.Marshal(body["reply_markup"])
	if err != nil || !strings.Contains(string(raw), "https://aero.example") || !strings.Contains(string(raw), "Открыть приложение") {
		t.Fatalf("markup %s", raw)
	}
}
