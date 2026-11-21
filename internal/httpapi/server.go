package httpapi

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/Tvister7/aero/internal/domain"
	_ "github.com/Tvister7/aero/internal/httpapi/docs"
	"github.com/Tvister7/aero/internal/sqlite"
	"github.com/Tvister7/aero/internal/telegram"
	httpSwagger "github.com/swaggo/http-swagger/v2"
)

type Server struct {
	DB     *sqlite.DB
	Bot    telegram.Sender
	Admins map[int64]struct{}
	Origin string
	Token  string
	Now    func() time.Time
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.Handle("GET /swagger/", httpSwagger.Handler(httpSwagger.PersistAuthorization(true)))
	mux.HandleFunc("GET /studio", s.auth(s.studio))
	mux.HandleFunc("GET /trainers", s.auth(s.publicTrainers))
	mux.HandleFunc("GET /trainers/{id}", s.auth(s.publicTrainer))
	mux.HandleFunc("GET /schedule", s.auth(s.schedule))
	mux.HandleFunc("GET /classes/{id}", s.auth(s.class))
	mux.HandleFunc("POST /classes/{id}/bookings", s.auth(s.book))
	mux.HandleFunc("GET /me", s.auth(s.me))
	mux.HandleFunc("PATCH /me", s.auth(s.patchMe))
	mux.HandleFunc("POST /me/ping", s.auth(s.ping))
	mux.HandleFunc("GET /me/bookings", s.auth(s.myBookings))
	mux.HandleFunc("GET /me/visits", s.auth(s.visits))
	mux.HandleFunc("GET /me/enrollments", s.auth(s.enrollments))
	mux.HandleFunc("PUT /me/enrollments", s.auth(s.putEnrollments))
	mux.HandleFunc("POST /bookings/{id}/cancel", s.auth(s.cancel))
	mux.HandleFunc("POST /bookings/{id}/reschedule", s.auth(s.reschedule))

	mux.HandleFunc("GET /trainer", s.trainer(s.trainerHome))
	mux.HandleFunc("PATCH /trainer", s.trainer(s.patchTrainerSelf))
	mux.HandleFunc("GET /trainer/schedule", s.trainer(s.trainerSchedule))
	mux.HandleFunc("GET /trainer/types", s.trainer(s.trainerTypes))
	mux.HandleFunc("POST /trainer/classes", s.trainer(s.trainerCreateClass))
	mux.HandleFunc("POST /trainer/individuals", s.trainer(s.trainerCreateIndividual))
	mux.HandleFunc("PATCH /trainer/classes/{id}", s.trainer(s.trainerMoveClass))
	mux.HandleFunc("GET /trainer/clients", s.trainer(s.trainerClients))
	mux.HandleFunc("POST /trainer/classes/{id}/cancel", s.trainer(s.trainerCancelClass))
	mux.HandleFunc("GET /trainer/classes/{id}/roster", s.trainer(s.trainerRoster))
	mux.HandleFunc("POST /trainer/bookings/{id}/attendance", s.trainer(s.trainerAttendance))
	mux.HandleFunc("GET /trainer/stats", s.trainer(s.trainerStats))

	mux.HandleFunc("GET /admin/trainers", s.admin(s.trainers))
	mux.HandleFunc("POST /admin/trainers", s.admin(s.createTrainer))
	mux.HandleFunc("GET /admin/trainers/{id}", s.admin(s.trainerCard))
	mux.HandleFunc("PATCH /admin/trainers/{id}", s.admin(s.patchTrainer))
	mux.HandleFunc("DELETE /admin/trainers/{id}", s.admin(s.deleteTrainer))
	mux.HandleFunc("POST /admin/trainers/{id}/absences", s.admin(s.addAbsence))
	mux.HandleFunc("DELETE /admin/trainers/{id}/absences/{absenceId}", s.admin(s.deleteAbsence))
	mux.HandleFunc("POST /admin/trainers/{id}/cancel-upcoming", s.admin(s.cancelUpcoming))
	mux.HandleFunc("GET /admin/training-types", s.admin(s.types))
	mux.HandleFunc("POST /admin/training-types", s.admin(s.createType))
	mux.HandleFunc("PATCH /admin/training-types/{id}", s.admin(s.patchType))
	mux.HandleFunc("GET /admin/templates", s.admin(s.templates))
	mux.HandleFunc("POST /admin/templates", s.admin(s.createTemplate))
	mux.HandleFunc("PATCH /admin/templates/{id}", s.admin(s.patchTemplate))
	mux.HandleFunc("POST /admin/weeks", s.admin(s.week))
	mux.HandleFunc("GET /admin/schedule", s.admin(s.adminSchedule))
	mux.HandleFunc("POST /admin/classes", s.admin(s.createClass))
	mux.HandleFunc("POST /admin/holds", s.admin(s.createHold))
	mux.HandleFunc("POST /admin/individuals", s.admin(s.createIndividual))
	mux.HandleFunc("PATCH /admin/classes/{id}", s.admin(s.patchClass))
	mux.HandleFunc("GET /admin/classes/{id}/roster", s.admin(s.roster))
	mux.HandleFunc("POST /admin/bookings/{id}/attendance", s.admin(s.attendance))
	mux.HandleFunc("GET /admin/clients.csv", s.admin(s.clientsCSV))
	mux.HandleFunc("GET /admin/clients", s.admin(s.clients))
	mux.HandleFunc("GET /admin/clients/{id}", s.admin(s.client))
	mux.HandleFunc("PATCH /admin/clients/{id}", s.admin(s.patchClient))
	mux.HandleFunc("POST /admin/clients/{id}/credits", s.admin(s.grant))
	mux.HandleFunc("GET /admin/stats", s.admin(s.stats))
	mux.HandleFunc("GET /admin/studio", s.admin(s.adminStudio))
	mux.HandleFunc("PUT /admin/studio", s.admin(s.putStudio))
	return s.cors(mux)
}

type endpoint func(http.ResponseWriter, *http.Request, telegram.User)

func (s *Server) auth(next endpoint) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		raw, ok := initData(r)
		if !ok {
			writeDomain(w, domain.Fail(domain.Unauthorized))
			return
		}
		user, err := telegram.Verify(s.Token, raw, s.now())
		if err != nil {
			writeDomain(w, err)
			return
		}
		next(w, r, user)
	}
}

type trainerEndpoint func(http.ResponseWriter, *http.Request, telegram.User, string)

func (s *Server) trainer(next trainerEndpoint) http.HandlerFunc {
	return s.auth(func(w http.ResponseWriter, r *http.Request, user telegram.User) {
		id, err := s.DB.ActiveTrainerID(r.Context(), user.ID)
		if finish(w, err) {
			return
		}
		if id == "" {
			writeDomain(w, domain.Fail(domain.Forbidden))
			return
		}
		next(w, r, user, id)
	})
}

func (s *Server) admin(next endpoint) http.HandlerFunc {
	return s.auth(func(w http.ResponseWriter, r *http.Request, user telegram.User) {
		if _, ok := s.Admins[user.ID]; !ok {
			writeDomain(w, domain.Fail(domain.Forbidden))
			return
		}
		next(w, r, user)
	})
}

func (s *Server) cors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if s.Origin != "" && r.Header.Get("Origin") == s.Origin {
			w.Header().Set("Access-Control-Allow-Origin", s.Origin)
			w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PATCH, PUT, DELETE, OPTIONS")
			w.Header().Set("Vary", "Origin")
		}
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
		next.ServeHTTP(w, r)
	})
}

func (s *Server) now() time.Time {
	if s.Now != nil {
		return s.Now()
	}
	return time.Now()
}

func (s *Server) isAdmin(id int64) bool {
	_, ok := s.Admins[id]
	return ok
}

func (s *Server) notifyFirstBooking(ctx context.Context, clientTelegram int64, out sqlite.BookOutcome) {
	starts, err := domain.ParseInstant(out.Class.StartsAt)
	if err != nil {
		slog.Error("first booking", "err", err)
		return
	}
	link := ""
	if studio, err := s.DB.Studio(ctx); err != nil {
		slog.Error("first booking studio", "err", err)
	} else {
		link = domain.PlaceLink(studio.MapURL, s.Origin)
	}
	notices := []sqlite.Outbound{{
		TelegramID: clientTelegram,
		Text:       domain.FirstClassText(out.Class.Title, starts, link),
	}}
	admins := make([]int64, 0, len(s.Admins))
	for id := range s.Admins {
		admins = append(admins, id)
	}
	slices.Sort(admins)
	for _, id := range admins {
		notices = append(notices, sqlite.Outbound{
			TelegramID: id,
			Text:       domain.AdminFirstBookingText(out.ClientName, out.Username, out.ClientPhone, out.Class.Title, starts),
		})
	}
	s.notify(ctx, notices)
}

func (s *Server) notify(ctx context.Context, notices []sqlite.Outbound) {
	if s.Bot == nil {
		return
	}
	for _, notice := range notices {
		sendCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		err := s.Bot.Send(sendCtx, notice.TelegramID, notice.Text)
		cancel()
		if err != nil {
			slog.Error("bot", "telegram_id", notice.TelegramID, "err", err)
		}
	}
}

func initData(r *http.Request) (string, bool) {
	header := r.Header.Get("Authorization")
	const prefix = "tma "
	if !strings.HasPrefix(header, prefix) {
		return "", false
	}
	raw := strings.TrimSpace(header[len(prefix):])
	return raw, raw != ""
}

func asUser(user telegram.User) sqlite.User {
	return sqlite.User{TelegramID: user.ID, Username: user.Username}
}

func studioFrom(raw map[string]json.RawMessage) (sqlite.Studio, *domain.Error) {
	var studio sqlite.Studio
	fields := []struct {
		key string
		dst *string
	}{
		{"address", &studio.Address},
		{"phone", &studio.Phone},
		{"instagram_url", &studio.InstagramURL},
		{"contact_url", &studio.ContactURL},
		{"map_url", &studio.MapURL},
		{"guide", &studio.Guide},
		{"pricing", &studio.Pricing},
	}
	for _, field := range fields {
		value, ok := raw[field.key]
		if !ok || json.Unmarshal(value, field.dst) != nil {
			return sqlite.Studio{}, domain.Fail(domain.Validation)
		}
	}
	return studio, nil
}
