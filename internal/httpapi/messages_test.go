package httpapi

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/Tvister7/aero/internal/domain"
	"github.com/Tvister7/aero/internal/sqlite"
	"github.com/Tvister7/aero/internal/telegram"
)

func TestAdminMessages(t *testing.T) {
	db, err := sqlite.Open(filepath.Join(t.TempDir(), "aero.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	now := time.Now()
	api := &Server{DB: db, Admins: map[int64]struct{}{7: {}}, Token: "token", Origin: "https://aero.example", Now: func() time.Time { return now }}
	server := httptest.NewServer(api.Handler())
	t.Cleanup(server.Close)
	admin := telegram.Sign("token", userValues(7, "boss", now))

	rows := list(callOK(t, server, http.MethodGet, "/admin/messages", admin, nil))
	if len(rows) != 1 || rows[0]["key"] != domain.MessageStart || rows[0]["body"] != domain.StartText(true) {
		t.Fatalf("catalog %v", rows)
	}
	saved := data(callOK(t, server, http.MethodPut, "/admin/messages/start", admin, map[string]any{"body": "Ждём вас."}))
	if saved["body"] != "Ждём вас." || saved["title"] != "Приветствие /start" {
		t.Fatalf("saved %v", saved)
	}
	status, body := call(t, server, http.MethodPut, "/admin/messages/reminder", admin, map[string]any{"body": "позже"})
	if status != http.StatusBadRequest || body["error_code"] != domain.Validation {
		t.Fatalf("unknown %d %v", status, body)
	}
	status, body = call(t, server, http.MethodPut, "/admin/messages/start", admin, map[string]any{"body": "  "})
	if status != http.StatusBadRequest {
		t.Fatalf("empty %d %v", status, body)
	}
	again := list(callOK(t, server, http.MethodGet, "/admin/messages", admin, nil))
	if again[0]["body"] != "Ждём вас." {
		t.Fatalf("stored %v", again)
	}
}
