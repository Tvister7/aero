package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/Tvister7/aero/internal/domain"
	"github.com/Tvister7/aero/internal/jobs"
	"github.com/Tvister7/aero/internal/sqlite"
	"github.com/Tvister7/aero/internal/telegram"
)

func TestStudioFlow(t *testing.T) {
	db, err := sqlite.Open(filepath.Join(t.TempDir(), "aero.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	now := time.Date(2026, 11, 18, 10, 0, 0, 0, domain.Tbilisi)
	bot := &recordingBot{}
	server := &Server{
		DB: db, Bot: bot, Admins: map[int64]struct{}{7: {}},
		Origin: "https://aero.example", Token: "token",
		Now: func() time.Time { return now },
	}
	httpServer := httptest.NewServer(server.Handler())
	t.Cleanup(httpServer.Close)
	admin := telegram.Sign("token", userValues(7, "boss", now))
	anna := telegram.Sign("token", userValues(1001, "anna", now))

	status, body := call(t, httpServer, http.MethodGet, "/me", "", nil)
	if status != http.StatusUnauthorized || body["message"] != "Откройте приложение заново из Telegram" {
		t.Fatalf("auth %d %v", status, body)
	}
	status, body = call(t, httpServer, http.MethodGet, "/admin/trainers", anna, nil)
	if status != http.StatusForbidden {
		t.Fatalf("forbidden %d %v", status, body)
	}

	trainer := data(callOK(t, httpServer, http.MethodPost, "/admin/trainers", admin, map[string]any{"name": "Татьяна"}))
	kind := data(callOK(t, httpServer, http.MethodPost, "/admin/training-types", admin, map[string]any{
		"title": "Aero Stretching", "duration_minutes": 60, "capacity": 6, "price_min": "17", "price_max": "40",
	}))
	if kind["price_min"] != "17.00" || kind["price_max"] != "40.00" {
		t.Fatalf("price %v", kind)
	}
	far := data(callOK(t, httpServer, http.MethodPost, "/admin/classes", admin, map[string]any{
		"trainer_id": trainer["id"], "training_type_id": kind["id"], "starts_at": "2026-11-21T10:00:00+04:00",
	}))
	soon := data(callOK(t, httpServer, http.MethodPost, "/admin/classes", admin, map[string]any{
		"trainer_id": trainer["id"], "training_type_id": kind["id"], "starts_at": "2026-11-18T14:00:00+04:00",
	}))

	me := data(callOK(t, httpServer, http.MethodGet, "/me", anna, nil))
	if me["policy"] != "postpay" || me["balance"] != float64(0) || me["username"] != "anna" || me["is_admin"] != false || me["visited"] != false {
		t.Fatalf("me %v", me)
	}
	booked := callOK(t, httpServer, http.MethodPost, "/classes/"+far["id"].(string)+"/bookings", anna, map[string]string{"name": "Анна", "phone": "+995"})
	if booked["balance"] != float64(0) || booked["class"].(map[string]any)["free_places"] != float64(5) {
		t.Fatalf("book %v", booked)
	}
	soonBook := callOK(t, httpServer, http.MethodPost, "/classes/"+soon["id"].(string)+"/bookings", anna, nil)
	status, body = call(t, httpServer, http.MethodPost, "/bookings/"+soonBook["booking_id"].(string)+"/cancel", anna, nil)
	if status != http.StatusConflict || body["error_code"] != domain.CutoffExceeded {
		t.Fatalf("cutoff %d %v", status, body)
	}
	var scheduler jobs.Scheduler
	scheduler.Tick(t.Context(), db, bot, now)
	if len(bot.texts) != 1 || bot.texts[0] == "" {
		t.Fatalf("reminder %v", bot.texts)
	}
	state, err := db.ReminderState(t.Context(), soonBook["booking_id"].(string))
	if err != nil || state != domain.ReminderSent {
		t.Fatalf("reminder state %s %v", state, err)
	}
	callOK(t, httpServer, http.MethodPost, "/me/ping", anna, nil)
	stats := data(callOK(t, httpServer, http.MethodGet, "/admin/stats?from=2026-11-18&to=2026-11-18", admin, nil))
	if stats["unique_users"] != float64(1) || stats["bookings_created"] != float64(2) || stats["no_shows"] != float64(1) {
		t.Fatalf("stats %v", stats)
	}
	status, body = call(t, httpServer, http.MethodPatch, "/admin/classes/"+soon["id"].(string), admin, map[string]any{"capacity": 0})
	if status != http.StatusBadRequest {
		t.Fatalf("capacity %d %v", status, body)
	}
	cancelled := data(callOK(t, httpServer, http.MethodPatch, "/admin/classes/"+soon["id"].(string), admin, map[string]any{"status": "cancelled"}))
	if cancelled["status"] != "cancelled" || len(bot.texts) != 2 {
		t.Fatalf("cancel slot %v texts %v", cancelled, bot.texts)
	}

	callOK(t, httpServer, http.MethodPost, "/bookings/"+booked["booking_id"].(string)+"/cancel", anna, nil)
	clients := list(callOK(t, httpServer, http.MethodGet, "/admin/clients?q=анна", admin, nil))
	if len(clients) != 1 {
		t.Fatalf("clients %v", clients)
	}
	clientID := clients[0]["id"].(string)
	callOK(t, httpServer, http.MethodPatch, "/admin/clients/"+clientID, admin, map[string]any{"policy": "restricted"})
	status, body = call(t, httpServer, http.MethodPost, "/classes/"+far["id"].(string)+"/bookings", anna, nil)
	if status != http.StatusConflict || body["message"] != "Занятия на балансе закончились" {
		t.Fatalf("credits %d %v", status, body)
	}
	card := data(callOK(t, httpServer, http.MethodPost, "/admin/clients/"+clientID+"/credits", admin, map[string]any{"delta": 3, "note": "пакет"}))
	if card["balance"] != float64(3) {
		t.Fatalf("grant %v", card)
	}
	booked = callOK(t, httpServer, http.MethodPost, "/classes/"+far["id"].(string)+"/bookings", anna, nil)
	if booked["balance"] != float64(3) {
		t.Fatalf("balance changed on book %v", booked)
	}
	status, _ = call(t, httpServer, http.MethodPost, "/classes/"+far["id"].(string)+"/bookings", anna, nil)
	if status != http.StatusConflict {
		t.Fatalf("second book %d", status)
	}
	next := data(callOK(t, httpServer, http.MethodPost, "/admin/classes", admin, map[string]any{
		"trainer_id": trainer["id"], "training_type_id": kind["id"], "starts_at": "2026-11-22T12:00:00+04:00",
	}))
	moved := callOK(t, httpServer, http.MethodPost, "/bookings/"+booked["booking_id"].(string)+"/reschedule", anna, map[string]any{"class_id": next["id"]})
	if moved["class"].(map[string]any)["id"] != next["id"] {
		t.Fatalf("reschedule %v", moved)
	}

	now = time.Date(2026, 11, 22, 11, 30, 0, 0, domain.Tbilisi)
	admin = telegram.Sign("token", userValues(7, "boss", now))
	arrived := callOK(t, httpServer, http.MethodPost, "/admin/bookings/"+booked["booking_id"].(string)+"/attendance", admin, map[string]any{"arrived": true})
	if arrived["balance"] != float64(2) || arrived["attendance"].(map[string]any)["payment"] != "paid" {
		t.Fatalf("attend %v", arrived)
	}
	again := callOK(t, httpServer, http.MethodPost, "/admin/bookings/"+booked["booking_id"].(string)+"/attendance", admin, map[string]any{"arrived": true})
	if again["balance"] != float64(2) {
		t.Fatalf("repeat attend %v", again)
	}
	cleared := callOK(t, httpServer, http.MethodPost, "/admin/bookings/"+booked["booking_id"].(string)+"/attendance", admin, map[string]any{"arrived": false})
	if cleared["balance"] != float64(3) || cleared["attendance"] != nil {
		t.Fatalf("reversal %v", cleared)
	}

	callOK(t, httpServer, http.MethodPost, "/admin/templates", admin, map[string]any{
		"trainer_id": trainer["id"], "training_type_id": kind["id"], "weekday": 4, "start_time": "10:00",
	})
	now = time.Date(2026, 11, 18, 12, 0, 0, 0, domain.Tbilisi)
	week := callOK(t, httpServer, http.MethodPost, "/admin/weeks", admin, map[string]any{})
	if week["week_start"] != "2026-11-23" || week["created"] != float64(1) {
		t.Fatalf("week %v", week)
	}
	againWeek := callOK(t, httpServer, http.MethodPost, "/admin/weeks", admin, map[string]any{"week_start": "2026-11-23"})
	if againWeek["created"] != float64(0) || againWeek["updated"] != float64(1) {
		t.Fatalf("week repeat %v", againWeek)
	}

	profile := data(callOK(t, httpServer, http.MethodPatch, "/me", anna, map[string]any{"name": "Анна, студия"}))
	if profile["visited"] != true || profile["name"] != "Анна, студия" {
		t.Fatalf("visited after cleared attendance %v", profile)
	}
	saved := data(callOK(t, httpServer, http.MethodPut, "/admin/studio", admin, map[string]any{
		"address": "ул. Чавчавадзе", "phone": "+995", "instagram_url": "https://instagram.com/aero", "contact_url": "", "map_url": "",
	}))
	if saved["instagram_url"] != "https://instagram.com/aero" {
		t.Fatalf("studio %v", saved)
	}
	csvReq, err := http.NewRequest(http.MethodGet, httpServer.URL+"/admin/clients.csv", nil)
	if err != nil {
		t.Fatal(err)
	}
	csvReq.Header.Set("Authorization", "tma "+admin)
	csvResp, err := httpServer.Client().Do(csvReq)
	if err != nil {
		t.Fatal(err)
	}
	csvBody, _ := io.ReadAll(csvResp.Body)
	csvResp.Body.Close()
	if csvResp.StatusCode != http.StatusOK || !bytes.Contains(csvBody, []byte(`"Анна, студия"`)) {
		t.Fatalf("csv %d %s", csvResp.StatusCode, csvBody)
	}

	req, err := http.NewRequest(http.MethodOptions, httpServer.URL+"/studio", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Origin", "https://aero.example")
	resp, err := httpServer.Client().Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent || resp.Header.Get("Access-Control-Allow-Origin") != "https://aero.example" {
		t.Fatalf("cors %d %s", resp.StatusCode, resp.Header.Get("Access-Control-Allow-Origin"))
	}
}

type recordingBot struct {
	texts []string
}

func (b *recordingBot) Send(_ context.Context, _ int64, text string) error {
	b.texts = append(b.texts, text)
	return nil
}

func userValues(id int64, username string, now time.Time) url.Values {
	values := url.Values{}
	values.Set("auth_date", strconv.FormatInt(now.Unix(), 10))
	values.Set("user", `{"id":`+strconv.FormatInt(id, 10)+`,"username":"`+username+`"}`)
	return values
}

func callOK(t *testing.T, server *httptest.Server, method, path, auth string, payload any) map[string]any {
	t.Helper()
	status, body := call(t, server, method, path, auth, payload)
	if status != http.StatusOK || body["ok"] != true {
		t.Fatalf("%s %s -> %d %v", method, path, status, body)
	}
	return body
}

func call(t *testing.T, server *httptest.Server, method, path, auth string, payload any) (int, map[string]any) {
	t.Helper()
	var reader io.Reader
	if payload != nil {
		raw, err := json.Marshal(payload)
		if err != nil {
			t.Fatal(err)
		}
		reader = bytes.NewReader(raw)
	}
	req, err := http.NewRequest(method, server.URL+path, reader)
	if err != nil {
		t.Fatal(err)
	}
	if auth != "" {
		req.Header.Set("Authorization", "tma "+auth)
	}
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := server.Client().Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var body map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil && method != http.MethodOptions {
		t.Fatal(err)
	}
	return resp.StatusCode, body
}

func data(body map[string]any) map[string]any {
	return body["data"].(map[string]any)
}

func list(body map[string]any) []map[string]any {
	raw := body["data"].([]any)
	out := make([]map[string]any, 0, len(raw))
	for _, item := range raw {
		out = append(out, item.(map[string]any))
	}
	return out
}
