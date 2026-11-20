package httpapi

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"

	"github.com/Tvister7/aero/internal/domain"
)

var messages = map[string]string{
	domain.Unauthorized:       "Откройте приложение заново из Telegram",
	domain.Forbidden:          "Недостаточно прав",
	domain.Validation:         "Проверьте поля",
	domain.ClassNotFound:      "Занятие не найдено",
	domain.NewClassNotFound:   "Новое занятие недоступно",
	domain.BookingNotFound:    "Запись не найдена",
	domain.ClassStarted:       "Занятие уже началось",
	domain.NoPlaces:           "Свободных мест нет",
	domain.AlreadyBooked:      "Вы уже записаны",
	domain.CutoffExceeded:     "Отмена и перенос доступны не позднее чем за 6 часов",
	domain.SameClass:          "Это то же занятие",
	domain.NoCredits:          "Занятия на балансе закончились",
	domain.AttendanceTooEarly: "Отметить явку можно за 30 минут до начала",
	domain.StudioBusy:         "Это время уже занято",
}

var statuses = map[string]int{
	domain.Unauthorized:       http.StatusUnauthorized,
	domain.Forbidden:          http.StatusForbidden,
	domain.Validation:         http.StatusBadRequest,
	domain.ClassNotFound:      http.StatusNotFound,
	domain.NewClassNotFound:   http.StatusNotFound,
	domain.BookingNotFound:    http.StatusNotFound,
	domain.ClassStarted:       http.StatusConflict,
	domain.NoPlaces:           http.StatusConflict,
	domain.AlreadyBooked:      http.StatusConflict,
	domain.CutoffExceeded:     http.StatusConflict,
	domain.SameClass:          http.StatusConflict,
	domain.NoCredits:          http.StatusConflict,
	domain.AttendanceTooEarly: http.StatusConflict,
	domain.StudioBusy:         http.StatusConflict,
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		slog.Error("write json", "err", err)
	}
}

func writeDomain(w http.ResponseWriter, err error) {
	var domainErr *domain.Error
	if errors.As(err, &domainErr) {
		status := statuses[domainErr.Code]
		if status == 0 {
			status = http.StatusBadRequest
		}
		message := messages[domainErr.Code]
		if message == "" {
			message = messages[domain.Validation]
		}
		writeJSON(w, status, map[string]any{"ok": false, "error_code": domainErr.Code, "message": message})
		return
	}
	slog.Error("request", "err", err)
	writeJSON(w, http.StatusInternalServerError, map[string]any{
		"ok": false, "error_code": "INTERNAL", "message": "Не удалось выполнить запрос",
	})
}

func finish(w http.ResponseWriter, err error) bool {
	if err == nil {
		return false
	}
	writeDomain(w, err)
	return true
}

func writeOK(w http.ResponseWriter, payload map[string]any) {
	payload["ok"] = true
	writeJSON(w, http.StatusOK, payload)
}

func readJSON(r *http.Request, dst any) *domain.Error {
	err := json.NewDecoder(r.Body).Decode(dst)
	if err != nil {
		return domain.Fail(domain.Validation)
	}
	return nil
}

func readOptional(r *http.Request, dst any) *domain.Error {
	err := json.NewDecoder(r.Body).Decode(dst)
	if err == nil || errors.Is(err, io.EOF) {
		return nil
	}
	return domain.Fail(domain.Validation)
}
