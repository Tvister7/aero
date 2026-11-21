package httpapi

import (
	"net/http"

	"github.com/Tvister7/aero/internal/domain"
	"github.com/Tvister7/aero/internal/sqlite"
	"github.com/Tvister7/aero/internal/telegram"
)

// Контакты студии.
//
// @Summary Контакты студии
// @Description Адрес, телефон, ссылки, памятка и прайс. Пустая строка контакта — поле не заполнено.
// @Tags Клиент
// @Security TMA
// @Success 200 {object} StudioResponse
// @Failure 401 {object} ErrorBody
// @Failure 500 {object} ErrorBody
// @Router /studio [get]
func (s *Server) studio(w http.ResponseWriter, r *http.Request, _ telegram.User) {
	studio, err := s.DB.Studio(r.Context())
	if finish(w, err) {
		return
	}
	writeOK(w, map[string]any{"data": studio})
}

// Расписание клиента.
//
// @Summary Расписание
// @Description Только scheduled, у которых тбилисская дата начала попадает в диапазон включительно.
// @Tags Клиент
// @Security TMA
// @Param from query string true "Начало диапазона, YYYY-MM-DD, Asia/Tbilisi" example(2026-09-01)
// @Param to query string true "Конец диапазона, включительно" example(2026-09-07)
// @Success 200 {object} ClassesResponse
// @Failure 400 {object} ErrorBody
// @Failure 401 {object} ErrorBody
// @Failure 500 {object} ErrorBody
// @Router /schedule [get]
func (s *Server) schedule(w http.ResponseWriter, r *http.Request, _ telegram.User) {
	from, to := r.URL.Query().Get("from"), r.URL.Query().Get("to")
	if from == "" || to == "" {
		writeDomain(w, domain.Fail(domain.Validation))
		return
	}
	data, err := s.DB.Schedule(r.Context(), from, to, false)
	if finish(w, err) {
		return
	}
	writeOK(w, map[string]any{"data": data})
}

// Карточка занятия для клиента.
//
// @Summary Карточка занятия
// @Description Отменённое и неизвестное занятие отвечают CLASS_NOT_FOUND. Поля status в ответе нет.
// @Tags Клиент
// @Security TMA
// @Param id path string true "Идентификатор занятия"
// @Success 200 {object} ClassResponse
// @Failure 401 {object} ErrorBody
// @Failure 404 {object} ErrorBody
// @Failure 500 {object} ErrorBody
// @Router /classes/{id} [get]
func (s *Server) class(w http.ResponseWriter, r *http.Request, _ telegram.User) {
	view, err := s.DB.Class(r.Context(), r.PathValue("id"), false)
	if finish(w, err) {
		return
	}
	writeOK(w, map[string]any{"data": view})
}

// Запись на занятие.
//
// @Summary Записаться
// @Description Баланс не меняется. Пустые имя и телефон не стирают сохранённые. Ник берётся из initData. Первая запись клиента отправляет ему ссылку, как найти студию, и сообщение каждому администратору.
// @Tags Клиент
// @Security TMA
// @Param id path string true "Идентификатор занятия"
// @Param body body ProfileInput false "Можно прислать пустое тело"
// @Success 200 {object} BookedResponse
// @Failure 400 {object} ErrorBody
// @Failure 401 {object} ErrorBody
// @Failure 404 {object} ErrorBody
// @Failure 409 {object} ErrorBody
// @Failure 500 {object} ErrorBody
// @Router /classes/{id}/bookings [post]
func (s *Server) book(w http.ResponseWriter, r *http.Request, user telegram.User) {
	var body ProfileInput
	if err := readOptional(r, &body); err != nil {
		writeDomain(w, err)
		return
	}
	out, err := s.DB.Book(r.Context(), s.now(), asUser(user), r.PathValue("id"), body.Name, body.Phone)
	if finish(w, err) {
		return
	}
	if out.First {
		s.notifyFirstBooking(r.Context(), user.ID, out)
	}
	writeOK(w, map[string]any{"booking_id": out.BookingID, "balance": out.Balance, "class": out.Class})
}

// Профиль открывшего приложение.
//
// @Summary Профиль
// @Description is_admin истинен, если telegram id есть в ADMIN_IDS. is_trainer истинен, если этот id привязан к активному тренеру. Новый человек создаётся с policy postpay. visited истинен, если была хотя бы одна явка, в том числе позже снятая.
// @Tags Клиент
// @Security TMA
// @Success 200 {object} MeResponse
// @Failure 401 {object} ErrorBody
// @Failure 500 {object} ErrorBody
// @Router /me [get]
func (s *Server) me(w http.ResponseWriter, r *http.Request, user telegram.User) {
	me, err := s.DB.Me(r.Context(), s.now(), asUser(user))
	if finish(w, err) {
		return
	}
	me.IsAdmin = s.isAdmin(user.ID)
	writeOK(w, map[string]any{"data": me})
}

// Правка имени и телефона.
//
// @Summary Изменить профиль
// @Description Пустая строка стирает сохранённое имя или телефон. Непустое имя — буквы, пробел, дефис и апостроф. Непустой телефон — цифры, пробел, +, скобки и дефис, не меньше 9 цифр.
// @Tags Клиент
// @Security TMA
// @Param body body ProfileInput false "Можно прислать пустое тело"
// @Success 200 {object} MeResponse
// @Failure 400 {object} ErrorBody
// @Failure 401 {object} ErrorBody
// @Failure 500 {object} ErrorBody
// @Router /me [patch]
func (s *Server) patchMe(w http.ResponseWriter, r *http.Request, user telegram.User) {
	var body ProfileInput
	if err := readOptional(r, &body); err != nil {
		writeDomain(w, err)
		return
	}
	me, err := s.DB.PatchMe(r.Context(), s.now(), asUser(user), body.Name, body.Phone)
	if finish(w, err) {
		return
	}
	me.IsAdmin = s.isAdmin(user.ID)
	writeOK(w, map[string]any{"data": me})
}

// Открытие Mini App.
//
// @Summary Отметить открытие
// @Description Пишет одну отметку открытия для сводки unique_users.
// @Tags Клиент
// @Security TMA
// @Success 200 {object} OpenedResponse
// @Failure 401 {object} ErrorBody
// @Failure 500 {object} ErrorBody
// @Router /me/ping [post]
func (s *Server) ping(w http.ResponseWriter, r *http.Request, user telegram.User) {
	if finish(w, s.DB.Ping(r.Context(), s.now(), asUser(user))) {
		return
	}
	writeOK(w, map[string]any{})
}

// Будущие записи клиента.
//
// @Summary Мои записи
// @Description Запись booked, занятие scheduled, starts_at ещё впереди. Сортировка по началу.
// @Tags Клиент
// @Security TMA
// @Success 200 {object} MyBookingsResponse
// @Failure 401 {object} ErrorBody
// @Failure 500 {object} ErrorBody
// @Router /me/bookings [get]
func (s *Server) myBookings(w http.ResponseWriter, r *http.Request, user telegram.User) {
	data, err := s.DB.MyBookings(r.Context(), s.now(), asUser(user))
	if finish(w, err) {
		return
	}
	writeOK(w, map[string]any{"data": data})
}

// История посещений.
//
// @Summary Мои посещения
// @Description Только активные явки. Снятая отметка, отмена и неявка сюда не попадают. Новые сверху.
// @Tags Клиент
// @Security TMA
// @Success 200 {object} VisitsResponse
// @Failure 401 {object} ErrorBody
// @Failure 500 {object} ErrorBody
// @Router /me/visits [get]
func (s *Server) visits(w http.ResponseWriter, r *http.Request, user telegram.User) {
	data, err := s.DB.Visits(r.Context(), asUser(user))
	if finish(w, err) {
		return
	}
	writeOK(w, map[string]any{"data": data})
}

// Отмена своей записи.
//
// @Summary Отменить запись
// @Description Доступно строго больше чем за 6 часов до начала. Ровно 6 часов уже поздно.
// @Tags Клиент
// @Security TMA
// @Param id path string true "Идентификатор записи"
// @Success 200 {object} CancelledResponse
// @Failure 401 {object} ErrorBody
// @Failure 404 {object} ErrorBody
// @Failure 409 {object} ErrorBody
// @Failure 500 {object} ErrorBody
// @Router /bookings/{id}/cancel [post]
func (s *Server) cancel(w http.ResponseWriter, r *http.Request, user telegram.User) {
	id, err := s.DB.CancelBooking(r.Context(), s.now(), asUser(user), r.PathValue("id"))
	if finish(w, err) {
		return
	}
	writeOK(w, map[string]any{"booking_id": id})
}

// Перенос своей записи.
//
// @Summary Перенести запись
// @Description Порог 6 часов считается по старому началу. Занятие из шаблона не переносится. Новое занятие должно быть того же вида. Баланс не проверяется и не меняется.
// @Tags Клиент
// @Security TMA
// @Param id path string true "Идентификатор записи"
// @Param body body RescheduleInput true "Новое занятие"
// @Success 200 {object} RescheduledResponse
// @Failure 400 {object} ErrorBody
// @Failure 401 {object} ErrorBody
// @Failure 404 {object} ErrorBody
// @Failure 409 {object} ErrorBody
// @Failure 500 {object} ErrorBody
// @Router /bookings/{id}/reschedule [post]
func (s *Server) reschedule(w http.ResponseWriter, r *http.Request, user telegram.User) {
	var body RescheduleInput
	if err := readJSON(r, &body); err != nil || body.ClassID == "" {
		writeDomain(w, domain.Fail(domain.Validation))
		return
	}
	view, err := s.DB.Reschedule(r.Context(), s.now(), asUser(user), r.PathValue("id"), body.ClassID)
	if finish(w, err) {
		return
	}
	writeOK(w, map[string]any{"booking_id": r.PathValue("id"), "class": view})
}

// Постоянные отметки шаблонов.
//
// @Summary Мои постоянные записи
// @Description Активные шаблоны активных тренеров. enrolled истинен, если клиент отметил слот.
// @Tags Клиент
// @Security TMA
// @Success 200 {object} EnrollmentsResponse
// @Failure 401 {object} ErrorBody
// @Failure 500 {object} ErrorBody
// @Router /me/enrollments [get]
func (s *Server) enrollments(w http.ResponseWriter, r *http.Request, user telegram.User) {
	data, err := s.DB.EnrollmentOptions(r.Context(), s.now(), asUser(user))
	if finish(w, err) {
		return
	}
	writeOK(w, map[string]any{"data": data})
}

// Замена постоянных отметок.
//
// @Summary Сохранить постоянные записи
// @Description Список заменяет отметки целиком. Отмена одной уже созданной записи отметку не снимает.
// @Tags Клиент
// @Security TMA
// @Param body body sqlite.EnrollmentInput true "Отмеченные шаблоны"
// @Success 200 {object} EnrollmentsResponse
// @Failure 400 {object} ErrorBody
// @Failure 401 {object} ErrorBody
// @Failure 500 {object} ErrorBody
// @Router /me/enrollments [put]
func (s *Server) putEnrollments(w http.ResponseWriter, r *http.Request, user telegram.User) {
	var body sqlite.EnrollmentInput
	if err := readJSON(r, &body); err != nil {
		writeDomain(w, err)
		return
	}
	data, err := s.DB.ReplaceEnrollments(r.Context(), s.now(), asUser(user), body.TemplateIDs)
	if finish(w, err) {
		return
	}
	writeOK(w, map[string]any{"data": data})
}

// Активные тренеры для клиента.
//
// @Summary Тренеры
// @Description Только активные. Телефон пуст, если контакт не заполнен. classes — названия активных шаблонов.
// @Tags Клиент
// @Security TMA
// @Success 200 {object} PublicTrainersResponse
// @Failure 401 {object} ErrorBody
// @Failure 500 {object} ErrorBody
// @Router /trainers [get]
func (s *Server) publicTrainers(w http.ResponseWriter, r *http.Request, _ telegram.User) {
	data, err := s.DB.PublicTrainers(r.Context())
	if finish(w, err) {
		return
	}
	writeOK(w, map[string]any{"data": data})
}

// Карточка активного тренера.
//
// @Summary Тренер
// @Tags Клиент
// @Security TMA
// @Param id path string true "id тренера"
// @Success 200 {object} PublicTrainerResponse
// @Failure 400 {object} ErrorBody
// @Failure 401 {object} ErrorBody
// @Failure 500 {object} ErrorBody
// @Router /trainers/{id} [get]
func (s *Server) publicTrainer(w http.ResponseWriter, r *http.Request, _ telegram.User) {
	item, err := s.DB.PublicTrainer(r.Context(), r.PathValue("id"))
	if finish(w, err) {
		return
	}
	writeOK(w, map[string]any{"data": item})
}
