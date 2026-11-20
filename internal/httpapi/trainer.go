package httpapi

import (
	"net/http"
	"strconv"

	"github.com/Tvister7/aero/internal/domain"
	"github.com/Tvister7/aero/internal/sqlite"
	"github.com/Tvister7/aero/internal/telegram"
)

// Карточка тренера.
//
// @Summary Карточка тренера
// @Tags Админ
// @Security TMA
// @Param id path string true "Идентификатор тренера"
// @Success 200 {object} TrainerDetailResponse
// @Failure 400 {object} ErrorBody
// @Failure 401 {object} ErrorBody
// @Failure 403 {object} ErrorBody
// @Failure 500 {object} ErrorBody
// @Router /admin/trainers/{id} [get]
func (s *Server) trainerCard(w http.ResponseWriter, r *http.Request, _ telegram.User) {
	item, err := s.DB.TrainerCard(r.Context(), r.PathValue("id"))
	if finish(w, err) {
		return
	}
	writeOK(w, map[string]any{"data": item})
}

// Отпуск тренера.
//
// @Summary Добавить отпуск
// @Description Занятия этого тренера в указанных датах отменяются вместе с записями.
// @Tags Админ
// @Security TMA
// @Param id path string true "Идентификатор тренера"
// @Param body body sqlite.AbsenceInput true "Даты включительно"
// @Success 200 {object} TrainerDetailResponse
// @Failure 400 {object} ErrorBody
// @Failure 401 {object} ErrorBody
// @Failure 403 {object} ErrorBody
// @Failure 500 {object} ErrorBody
// @Router /admin/trainers/{id}/absences [post]
func (s *Server) addAbsence(w http.ResponseWriter, r *http.Request, _ telegram.User) {
	var body sqlite.AbsenceInput
	if err := readJSON(r, &body); err != nil {
		writeDomain(w, err)
		return
	}
	item, notes, err := s.DB.AddAbsence(r.Context(), s.now(), r.PathValue("id"), body)
	if finish(w, err) {
		return
	}
	s.notify(r.Context(), notes)
	writeOK(w, map[string]any{"data": item})
}

// Снятие одной строки отпуска. Уже отменённые занятия не возвращаются.
//
// @Summary Удалить отпуск
// @Tags Админ
// @Security TMA
// @Param id path string true "Идентификатор тренера"
// @Param absenceId path string true "Идентификатор отпуска"
// @Success 200 {object} TrainerDetailResponse
// @Failure 400 {object} ErrorBody
// @Failure 401 {object} ErrorBody
// @Failure 403 {object} ErrorBody
// @Failure 500 {object} ErrorBody
// @Router /admin/trainers/{id}/absences/{absenceId} [delete]
func (s *Server) deleteAbsence(w http.ResponseWriter, r *http.Request, _ telegram.User) {
	item, err := s.DB.DeleteAbsence(r.Context(), r.PathValue("id"), r.PathValue("absenceId"))
	if finish(w, err) {
		return
	}
	writeOK(w, map[string]any{"data": item})
}

// Снятие будущих занятий тренера.
//
// @Summary Снять будущие занятия
// @Description Отменяет scheduled с началом позже текущего момента. Прошедшие остаются.
// @Tags Админ
// @Security TMA
// @Param id path string true "Идентификатор тренера"
// @Success 200 {object} OKResponse
// @Failure 400 {object} ErrorBody
// @Failure 401 {object} ErrorBody
// @Failure 403 {object} ErrorBody
// @Failure 500 {object} ErrorBody
// @Router /admin/trainers/{id}/cancel-upcoming [post]
func (s *Server) cancelUpcoming(w http.ResponseWriter, r *http.Request, _ telegram.User) {
	notes, err := s.DB.CancelUpcoming(r.Context(), s.now(), r.PathValue("id"))
	if finish(w, err) {
		return
	}
	s.notify(r.Context(), notes)
	writeOK(w, map[string]any{})
}

// Удаление тренера.
//
// @Summary Удалить тренера
// @Description Будущие занятия отменяются, шаблоны выключаются, Telegram снимается. Строка удаляется, только если занятий в истории нет.
// @Tags Админ
// @Security TMA
// @Param id path string true "Идентификатор тренера"
// @Success 200 {object} OKResponse
// @Failure 400 {object} ErrorBody
// @Failure 401 {object} ErrorBody
// @Failure 403 {object} ErrorBody
// @Failure 500 {object} ErrorBody
// @Router /admin/trainers/{id} [delete]
func (s *Server) deleteTrainer(w http.ResponseWriter, r *http.Request, _ telegram.User) {
	notes, err := s.DB.DeleteTrainer(r.Context(), s.now(), r.PathValue("id"))
	if finish(w, err) {
		return
	}
	s.notify(r.Context(), notes)
	writeOK(w, map[string]any{})
}

// Своя карточка тренера.
//
// @Summary Карточка тренера
// @Description Имя, текст и телефон. Отпуск и удаление здесь недоступны.
// @Tags Тренер
// @Security TMA
// @Success 200 {object} TrainerResponse
// @Failure 401 {object} ErrorBody
// @Failure 403 {object} ErrorBody
// @Failure 500 {object} ErrorBody
// @Router /trainer [get]
func (s *Server) trainerHome(w http.ResponseWriter, r *http.Request, _ telegram.User, trainerID string) {
	item, err := s.DB.TrainerCard(r.Context(), trainerID)
	if finish(w, err) {
		return
	}
	writeOK(w, map[string]any{"data": item.Trainer})
}

// Свой текст и телефон.
//
// @Summary Изменить свою карточку
// @Tags Тренер
// @Security TMA
// @Param body body sqlite.TrainerProfile true "Текст и телефон"
// @Success 200 {object} TrainerResponse
// @Failure 400 {object} ErrorBody
// @Failure 401 {object} ErrorBody
// @Failure 403 {object} ErrorBody
// @Failure 500 {object} ErrorBody
// @Router /trainer [patch]
func (s *Server) patchTrainerSelf(w http.ResponseWriter, r *http.Request, _ telegram.User, trainerID string) {
	var body sqlite.TrainerProfile
	if err := readJSON(r, &body); err != nil {
		writeDomain(w, err)
		return
	}
	item, err := s.DB.PatchOwnTrainer(r.Context(), trainerID, body.About, body.Phone)
	if finish(w, err) {
		return
	}
	writeOK(w, map[string]any{"data": item})
}

// Свои занятия.
//
// @Summary Свои занятия
// @Tags Тренер
// @Security TMA
// @Param from query string true "YYYY-MM-DD" example(2026-09-01)
// @Param to query string true "YYYY-MM-DD" example(2026-09-07)
// @Success 200 {object} ClassesResponse
// @Failure 400 {object} ErrorBody
// @Failure 401 {object} ErrorBody
// @Failure 403 {object} ErrorBody
// @Failure 500 {object} ErrorBody
// @Router /trainer/schedule [get]
func (s *Server) trainerSchedule(w http.ResponseWriter, r *http.Request, _ telegram.User, trainerID string) {
	from, to := r.URL.Query().Get("from"), r.URL.Query().Get("to")
	if from == "" || to == "" {
		writeDomain(w, domain.Fail(domain.Validation))
		return
	}
	data, err := s.DB.TrainerSchedule(r.Context(), from, to, trainerID)
	if finish(w, err) {
		return
	}
	writeOK(w, map[string]any{"data": data})
}

// Виды занятий для разового слота.
//
// @Summary Виды занятий
// @Tags Тренер
// @Security TMA
// @Success 200 {object} TypesResponse
// @Failure 401 {object} ErrorBody
// @Failure 403 {object} ErrorBody
// @Failure 500 {object} ErrorBody
// @Router /trainer/types [get]
func (s *Server) trainerTypes(w http.ResponseWriter, r *http.Request, _ telegram.User, _ string) {
	data, err := s.DB.ListTypes(r.Context())
	if finish(w, err) {
		return
	}
	writeOK(w, map[string]any{"data": data})
}

// Разовое своё занятие.
//
// @Summary Создать разовое занятие
// @Description trainer_id из тела игнорируется: слот всегда принадлежит этому тренеру.
// @Tags Тренер
// @Security TMA
// @Param body body sqlite.ClassInput true "Тип и начало"
// @Success 200 {object} ClassResponse
// @Failure 400 {object} ErrorBody
// @Failure 401 {object} ErrorBody
// @Failure 403 {object} ErrorBody
// @Failure 500 {object} ErrorBody
// @Router /trainer/classes [post]
func (s *Server) trainerCreateClass(w http.ResponseWriter, r *http.Request, _ telegram.User, trainerID string) {
	var body sqlite.ClassInput
	if err := readJSON(r, &body); err != nil {
		writeDomain(w, err)
		return
	}
	body.TrainerID = trainerID
	view, err := s.DB.CreateClass(r.Context(), s.now(), body)
	if finish(w, err) {
		return
	}
	writeOK(w, map[string]any{"data": view})
}

// Отмена своего будущего занятия.
//
// @Summary Отменить своё занятие
// @Tags Тренер
// @Security TMA
// @Param id path string true "Идентификатор занятия"
// @Success 200 {object} ClassResponse
// @Failure 401 {object} ErrorBody
// @Failure 403 {object} ErrorBody
// @Failure 404 {object} ErrorBody
// @Failure 409 {object} ErrorBody
// @Failure 500 {object} ErrorBody
// @Router /trainer/classes/{id}/cancel [post]
func (s *Server) trainerCancelClass(w http.ResponseWriter, r *http.Request, _ telegram.User, trainerID string) {
	change, err := s.DB.TrainerCancelClass(r.Context(), s.now(), trainerID, r.PathValue("id"))
	if finish(w, err) {
		return
	}
	s.notify(r.Context(), change.Notices)
	writeOK(w, map[string]any{"data": change.Class})
}

// Журнал своего занятия.
//
// @Summary Журнал своего занятия
// @Tags Тренер
// @Security TMA
// @Param id path string true "Идентификатор занятия"
// @Success 200 {object} RosterResponse
// @Failure 401 {object} ErrorBody
// @Failure 403 {object} ErrorBody
// @Failure 404 {object} ErrorBody
// @Failure 500 {object} ErrorBody
// @Router /trainer/classes/{id}/roster [get]
func (s *Server) trainerRoster(w http.ResponseWriter, r *http.Request, _ telegram.User, trainerID string) {
	if finish(w, s.DB.OwnClass(r.Context(), trainerID, r.PathValue("id"))) {
		return
	}
	roster, err := s.DB.Roster(r.Context(), r.PathValue("id"))
	if finish(w, err) {
		return
	}
	writeOK(w, map[string]any{"data": roster})
}

// Явка на своей записи.
//
// @Summary Отметить явку
// @Description actor_role равен trainer. Чужая запись отвечает BOOKING_NOT_FOUND.
// @Tags Тренер
// @Security TMA
// @Param id path string true "Идентификатор записи"
// @Param body body AttendanceInput true "Явка"
// @Success 200 {object} AttendanceResponse
// @Failure 400 {object} ErrorBody
// @Failure 401 {object} ErrorBody
// @Failure 403 {object} ErrorBody
// @Failure 404 {object} ErrorBody
// @Failure 409 {object} ErrorBody
// @Failure 500 {object} ErrorBody
// @Router /trainer/bookings/{id}/attendance [post]
func (s *Server) trainerAttendance(w http.ResponseWriter, r *http.Request, user telegram.User, trainerID string) {
	var body AttendanceInput
	if err := readJSON(r, &body); err != nil || body.Arrived == nil {
		writeDomain(w, domain.Fail(domain.Validation))
		return
	}
	bookingID := r.PathValue("id")
	classID, err := s.bookingClass(r, bookingID)
	if finish(w, err) {
		return
	}
	if finish(w, s.DB.OwnClass(r.Context(), trainerID, classID)) {
		return
	}
	out, err := s.DB.Attend(r.Context(), s.now(), bookingID, *body.Arrived, domain.ActorTrainer, strconv.FormatInt(user.ID, 10))
	if finish(w, err) {
		return
	}
	writeOK(w, map[string]any{"booking_id": out.BookingID, "attendance": out.Attendance, "balance": out.Balance})
}

func (s *Server) bookingClass(r *http.Request, bookingID string) (string, error) {
	return s.DB.BookingClass(r.Context(), bookingID)
}

// Сводка тренера.
//
// @Summary Сводка за период
// @Tags Тренер
// @Security TMA
// @Param from query string true "YYYY-MM-DD" example(2026-09-01)
// @Param to query string true "YYYY-MM-DD" example(2026-09-30)
// @Success 200 {object} TrainerStatsResponse
// @Failure 400 {object} ErrorBody
// @Failure 401 {object} ErrorBody
// @Failure 403 {object} ErrorBody
// @Failure 500 {object} ErrorBody
// @Router /trainer/stats [get]
func (s *Server) trainerStats(w http.ResponseWriter, r *http.Request, _ telegram.User, trainerID string) {
	from, to := r.URL.Query().Get("from"), r.URL.Query().Get("to")
	if from == "" || to == "" {
		writeDomain(w, domain.Fail(domain.Validation))
		return
	}
	stats, err := s.DB.TrainerStats(r.Context(), trainerID, from, to)
	if finish(w, err) {
		return
	}
	writeOK(w, map[string]any{"data": stats})
}

// Поиск клиента по имени для индивидуального занятия.
//
// @Summary Найти клиента
// @Description Пустой запрос возвращает пустой список. В строке только id и имя.
// @Tags Тренер
// @Security TMA
// @Param q query string false "Часть имени"
// @Success 200 {object} ClientNamesResponse
// @Failure 401 {object} ErrorBody
// @Failure 403 {object} ErrorBody
// @Failure 500 {object} ErrorBody
// @Router /trainer/clients [get]
func (s *Server) trainerClients(w http.ResponseWriter, r *http.Request, _ telegram.User, _ string) {
	data, err := s.DB.SearchClientNames(r.Context(), r.URL.Query().Get("q"))
	if finish(w, err) {
		return
	}
	writeOK(w, map[string]any{"data": data})
}

// Индивидуальное на себя.
//
// @Summary Создать индивидуальное
// @Description Клиент назначается сразу. В общем расписании слота нет. trainer_id из тела игнорируется.
// @Tags Тренер
// @Security TMA
// @Param body body sqlite.IndividualInput true "Клиент и начало"
// @Success 200 {object} ClassResponse
// @Failure 400 {object} ErrorBody
// @Failure 401 {object} ErrorBody
// @Failure 403 {object} ErrorBody
// @Failure 409 {object} ErrorBody
// @Failure 500 {object} ErrorBody
// @Router /trainer/individuals [post]
func (s *Server) trainerCreateIndividual(w http.ResponseWriter, r *http.Request, _ telegram.User, trainerID string) {
	var body sqlite.IndividualInput
	if err := readJSON(r, &body); err != nil {
		writeDomain(w, err)
		return
	}
	view, err := s.DB.CreateIndividual(r.Context(), s.now(), trainerID, body.ClientID, body.StartsAt, body.DurationMinutes)
	if finish(w, err) {
		return
	}
	writeOK(w, map[string]any{"data": view})
}

// Перенос своего индивидуального.
//
// @Summary Перенести индивидуальное
// @Description Только своё индивидуальное. Новое время не должно пересекать занятый зал.
// @Tags Тренер
// @Security TMA
// @Param id path string true "Идентификатор занятия"
// @Param body body sqlite.ClassPatch true "Новое начало"
// @Success 200 {object} ClassResponse
// @Failure 400 {object} ErrorBody
// @Failure 401 {object} ErrorBody
// @Failure 403 {object} ErrorBody
// @Failure 404 {object} ErrorBody
// @Failure 409 {object} ErrorBody
// @Failure 500 {object} ErrorBody
// @Router /trainer/classes/{id} [patch]
func (s *Server) trainerMoveClass(w http.ResponseWriter, r *http.Request, _ telegram.User, trainerID string) {
	var body sqlite.ClassPatch
	if err := readJSON(r, &body); err != nil {
		writeDomain(w, err)
		return
	}
	if body.StartsAt == nil || *body.StartsAt == "" {
		writeDomain(w, domain.Fail(domain.Validation))
		return
	}
	change, err := s.DB.MoveTrainerClass(r.Context(), s.now(), trainerID, r.PathValue("id"), *body.StartsAt)
	if finish(w, err) {
		return
	}
	s.notify(r.Context(), change.Notices)
	writeOK(w, map[string]any{"data": change.Class})
}
