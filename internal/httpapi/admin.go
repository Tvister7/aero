package httpapi

import (
	"encoding/csv"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/Tvister7/aero/internal/domain"
	"github.com/Tvister7/aero/internal/sqlite"
	"github.com/Tvister7/aero/internal/telegram"
)

// Список тренеров.
//
// @Summary Список тренеров
// @Tags Админ
// @Security TMA
// @Success 200 {object} TrainersResponse
// @Failure 401 {object} ErrorBody
// @Failure 403 {object} ErrorBody
// @Failure 500 {object} ErrorBody
// @Router /admin/trainers [get]
func (s *Server) trainers(w http.ResponseWriter, r *http.Request, _ telegram.User) {
	data, err := s.DB.ListTrainers(r.Context())
	if finish(w, err) {
		return
	}
	writeOK(w, map[string]any{"data": data})
}

// Новый тренер.
//
// @Summary Создать тренера
// @Description Имя, которое уже есть, отвечает VALIDATION.
// @Tags Админ
// @Security TMA
// @Param body body TrainerInput true "Имя"
// @Success 200 {object} TrainerResponse
// @Failure 400 {object} ErrorBody
// @Failure 401 {object} ErrorBody
// @Failure 403 {object} ErrorBody
// @Failure 500 {object} ErrorBody
// @Router /admin/trainers [post]
func (s *Server) createTrainer(w http.ResponseWriter, r *http.Request, _ telegram.User) {
	var body TrainerInput
	if err := readJSON(r, &body); err != nil {
		writeDomain(w, err)
		return
	}
	item, err := s.DB.CreateTrainer(r.Context(), s.now(), body.Name)
	if finish(w, err) {
		return
	}
	writeOK(w, map[string]any{"data": item})
}

// Правка тренера.
//
// @Summary Изменить тренера
// @Tags Админ
// @Security TMA
// @Param id path string true "Идентификатор тренера"
// @Param body body sqlite.TrainerFields true "Меняющиеся поля"
// @Success 200 {object} TrainerResponse
// @Failure 400 {object} ErrorBody
// @Failure 401 {object} ErrorBody
// @Failure 403 {object} ErrorBody
// @Failure 500 {object} ErrorBody
// @Router /admin/trainers/{id} [patch]
func (s *Server) patchTrainer(w http.ResponseWriter, r *http.Request, _ telegram.User) {
	var body sqlite.TrainerFields
	if err := readJSON(r, &body); err != nil {
		writeDomain(w, err)
		return
	}
	item, err := s.DB.PatchTrainer(r.Context(), r.PathValue("id"), body)
	if finish(w, err) {
		return
	}
	writeOK(w, map[string]any{"data": item})
}

// Список видов занятий.
//
// @Summary Список видов
// @Tags Админ
// @Security TMA
// @Success 200 {object} TypesResponse
// @Failure 401 {object} ErrorBody
// @Failure 403 {object} ErrorBody
// @Failure 500 {object} ErrorBody
// @Router /admin/training-types [get]
func (s *Server) types(w http.ResponseWriter, r *http.Request, _ telegram.User) {
	data, err := s.DB.ListTypes(r.Context())
	if finish(w, err) {
		return
	}
	writeOK(w, map[string]any{"data": data})
}

// Новый вид занятия.
//
// @Summary Создать вид
// @Description Нужны title, duration_minutes, capacity, price_min и price_max. Цена остаётся подписью.
// @Tags Админ
// @Security TMA
// @Param body body sqlite.TypeInput true "Вид"
// @Success 200 {object} TypeResponse
// @Failure 400 {object} ErrorBody
// @Failure 401 {object} ErrorBody
// @Failure 403 {object} ErrorBody
// @Failure 500 {object} ErrorBody
// @Router /admin/training-types [post]
func (s *Server) createType(w http.ResponseWriter, r *http.Request, _ telegram.User) {
	var body sqlite.TypeInput
	if err := readJSON(r, &body); err != nil {
		writeDomain(w, err)
		return
	}
	item, err := s.DB.CreateType(r.Context(), s.now(), body)
	if finish(w, err) {
		return
	}
	writeOK(w, map[string]any{"data": item})
}

// Правка вида занятия.
//
// @Summary Изменить вид
// @Description Уже созданные занятия этот вызов не переписывает.
// @Tags Админ
// @Security TMA
// @Param id path string true "Идентификатор вида"
// @Param body body sqlite.TypeInput true "Меняющиеся поля"
// @Success 200 {object} TypeResponse
// @Failure 400 {object} ErrorBody
// @Failure 401 {object} ErrorBody
// @Failure 403 {object} ErrorBody
// @Failure 500 {object} ErrorBody
// @Router /admin/training-types/{id} [patch]
func (s *Server) patchType(w http.ResponseWriter, r *http.Request, _ telegram.User) {
	var body sqlite.TypeInput
	if err := readJSON(r, &body); err != nil {
		writeDomain(w, err)
		return
	}
	item, err := s.DB.PatchType(r.Context(), r.PathValue("id"), body)
	if finish(w, err) {
		return
	}
	writeOK(w, map[string]any{"data": item})
}

// Список шаблонов недели.
//
// @Summary Список шаблонов
// @Tags Админ
// @Security TMA
// @Success 200 {object} TemplatesResponse
// @Failure 401 {object} ErrorBody
// @Failure 403 {object} ErrorBody
// @Failure 500 {object} ErrorBody
// @Router /admin/templates [get]
func (s *Server) templates(w http.ResponseWriter, r *http.Request, _ telegram.User) {
	data, err := s.DB.ListTemplates(r.Context())
	if finish(w, err) {
		return
	}
	writeOK(w, map[string]any{"data": data})
}

// Новый шаблон.
//
// @Summary Создать шаблон
// @Description title копируется с типа. Повтор четвёрки тренер, название, день и время отвечает VALIDATION.
// @Tags Админ
// @Security TMA
// @Param body body sqlite.TemplateInput true "Шаблон"
// @Success 200 {object} TemplateResponse
// @Failure 400 {object} ErrorBody
// @Failure 401 {object} ErrorBody
// @Failure 403 {object} ErrorBody
// @Failure 500 {object} ErrorBody
// @Router /admin/templates [post]
func (s *Server) createTemplate(w http.ResponseWriter, r *http.Request, _ telegram.User) {
	var body sqlite.TemplateInput
	if err := readJSON(r, &body); err != nil {
		writeDomain(w, err)
		return
	}
	item, err := s.DB.CreateTemplate(r.Context(), s.now(), body)
	if finish(w, err) {
		return
	}
	writeOK(w, map[string]any{"data": item})
}

// Правка шаблона.
//
// @Summary Изменить шаблон
// @Description Уже созданные занятия этот вызов не меняет.
// @Tags Админ
// @Security TMA
// @Param id path string true "Идентификатор шаблона"
// @Param body body sqlite.TemplateInput true "Меняющиеся поля"
// @Success 200 {object} TemplateResponse
// @Failure 400 {object} ErrorBody
// @Failure 401 {object} ErrorBody
// @Failure 403 {object} ErrorBody
// @Failure 500 {object} ErrorBody
// @Router /admin/templates/{id} [patch]
func (s *Server) patchTemplate(w http.ResponseWriter, r *http.Request, _ telegram.User) {
	var body sqlite.TemplateInput
	if err := readJSON(r, &body); err != nil {
		writeDomain(w, err)
		return
	}
	item, err := s.DB.PatchTemplate(r.Context(), r.PathValue("id"), body)
	if finish(w, err) {
		return
	}
	writeOK(w, map[string]any{"data": item})
}

// Генерация недели из активных шаблонов.
//
// @Summary Собрать неделю
// @Description Пустое тело — понедельник следующей ISO-недели в Тбилиси. Повтор сходится в те же слоты.
// @Tags Админ
// @Security TMA
// @Param body body WeekInput false "Можно прислать пустое тело"
// @Success 200 {object} WeekResponse
// @Failure 400 {object} ErrorBody
// @Failure 401 {object} ErrorBody
// @Failure 403 {object} ErrorBody
// @Failure 500 {object} ErrorBody
// @Router /admin/weeks [post]
func (s *Server) week(w http.ResponseWriter, r *http.Request, _ telegram.User) {
	var body WeekInput
	if err := readOptional(r, &body); err != nil {
		writeDomain(w, err)
		return
	}
	out, err := s.DB.GenerateWeek(r.Context(), s.now(), body.WeekStart)
	if finish(w, err) {
		return
	}
	writeOK(w, map[string]any{"week_start": out.WeekStart, "week_end": out.WeekEnd, "created": out.Created, "updated": out.Updated})
}

// Расписание админа, включая отменённые слоты.
//
// @Summary Расписание админа
// @Description Те же даты, что у клиента, плюс cancelled и поле status.
// @Tags Админ
// @Security TMA
// @Param from query string true "Начало диапазона, YYYY-MM-DD" example(2026-09-01)
// @Param to query string true "Конец диапазона, включительно" example(2026-09-07)
// @Success 200 {object} ClassesResponse
// @Failure 400 {object} ErrorBody
// @Failure 401 {object} ErrorBody
// @Failure 403 {object} ErrorBody
// @Failure 500 {object} ErrorBody
// @Router /admin/schedule [get]
func (s *Server) adminSchedule(w http.ResponseWriter, r *http.Request, _ telegram.User) {
	from, to := r.URL.Query().Get("from"), r.URL.Query().Get("to")
	if from == "" || to == "" {
		writeDomain(w, domain.Fail(domain.Validation))
		return
	}
	data, err := s.DB.Schedule(r.Context(), from, to, true)
	if finish(w, err) {
		return
	}
	writeOK(w, map[string]any{"data": data})
}

// Разовое занятие вне шаблона.
//
// @Summary Создать занятие
// @Description template_id остаётся пустым. Название, длительность и подпись стоимости снимаются с типа.
// @Tags Админ
// @Security TMA
// @Param body body sqlite.ClassInput true "Слот"
// @Success 200 {object} ClassResponse
// @Failure 400 {object} ErrorBody
// @Failure 401 {object} ErrorBody
// @Failure 403 {object} ErrorBody
// @Failure 500 {object} ErrorBody
// @Router /admin/classes [post]
func (s *Server) createClass(w http.ResponseWriter, r *http.Request, _ telegram.User) {
	var body sqlite.ClassInput
	if err := readJSON(r, &body); err != nil {
		writeDomain(w, err)
		return
	}
	view, err := s.DB.CreateClass(r.Context(), s.now(), body)
	if finish(w, err) {
		return
	}
	writeOK(w, map[string]any{"data": view})
}

// Перенос или отмена слота.
//
// @Summary Изменить занятие
// @Description status cancelled отменяет слот и записи. Иной набор полей переносит слот. Записанным уходит сообщение бота.
// @Tags Админ
// @Security TMA
// @Param id path string true "Идентификатор занятия"
// @Param body body sqlite.ClassPatch true "Перенос или отмена"
// @Success 200 {object} ClassResponse
// @Failure 400 {object} ErrorBody
// @Failure 401 {object} ErrorBody
// @Failure 403 {object} ErrorBody
// @Failure 404 {object} ErrorBody
// @Failure 500 {object} ErrorBody
// @Router /admin/classes/{id} [patch]
func (s *Server) patchClass(w http.ResponseWriter, r *http.Request, _ telegram.User) {
	var body sqlite.ClassPatch
	if err := readJSON(r, &body); err != nil {
		writeDomain(w, err)
		return
	}
	change, err := s.DB.PatchClass(r.Context(), s.now(), r.PathValue("id"), body)
	if finish(w, err) {
		return
	}
	s.notify(r.Context(), change.Notices)
	writeOK(w, map[string]any{"data": change.Class})
}

// Журнал одного занятия.
//
// @Summary Журнал занятия
// @Description Только записи booked. Отменённый слот отвечает CLASS_NOT_FOUND.
// @Tags Админ
// @Security TMA
// @Param id path string true "Идентификатор занятия"
// @Success 200 {object} RosterResponse
// @Failure 401 {object} ErrorBody
// @Failure 403 {object} ErrorBody
// @Failure 404 {object} ErrorBody
// @Failure 500 {object} ErrorBody
// @Router /admin/classes/{id}/roster [get]
func (s *Server) roster(w http.ResponseWriter, r *http.Request, _ telegram.User) {
	roster, err := s.DB.Roster(r.Context(), r.PathValue("id"))
	if finish(w, err) {
		return
	}
	writeOK(w, map[string]any{"data": roster})
}

// Явка по записи.
//
// @Summary Отметить явку
// @Description Не раньше чем за 30 минут до начала. Списание одно на активную явку. Снятие пишет обратную проводку.
// @Tags Админ
// @Security TMA
// @Param id path string true "Идентификатор записи"
// @Param body body AttendanceInput true "Пришёл или снять отметку"
// @Success 200 {object} AttendanceResponse
// @Failure 400 {object} ErrorBody
// @Failure 401 {object} ErrorBody
// @Failure 403 {object} ErrorBody
// @Failure 404 {object} ErrorBody
// @Failure 409 {object} ErrorBody
// @Failure 500 {object} ErrorBody
// @Router /admin/bookings/{id}/attendance [post]
func (s *Server) attendance(w http.ResponseWriter, r *http.Request, user telegram.User) {
	var body AttendanceInput
	if err := readJSON(r, &body); err != nil || body.Arrived == nil {
		writeDomain(w, domain.Fail(domain.Validation))
		return
	}
	out, err := s.DB.Attend(r.Context(), s.now(), r.PathValue("id"), *body.Arrived, domain.ActorAdmin, strconv.FormatInt(user.ID, 10))
	if finish(w, err) {
		return
	}
	writeOK(w, map[string]any{"booking_id": out.BookingID, "attendance": out.Attendance, "balance": out.Balance})
}

// Поиск клиентов.
//
// @Summary Список клиентов
// @Description q режет по вхождению в имя, телефон, ник и telegram id. Пустой q возвращает всех.
// @Tags Админ
// @Security TMA
// @Param q query string false "Подстрока поиска"
// @Success 200 {object} ClientsResponse
// @Failure 401 {object} ErrorBody
// @Failure 403 {object} ErrorBody
// @Failure 500 {object} ErrorBody
// @Router /admin/clients [get]
func (s *Server) clients(w http.ResponseWriter, r *http.Request, _ telegram.User) {
	data, err := s.DB.ListClients(r.Context(), r.URL.Query().Get("q"))
	if finish(w, err) {
		return
	}
	writeOK(w, map[string]any{"data": data})
}

// Карточка клиента с проводками.
//
// @Summary Карточка клиента
// @Tags Админ
// @Security TMA
// @Param id path string true "Идентификатор клиента"
// @Success 200 {object} ClientResponse
// @Failure 400 {object} ErrorBody
// @Failure 401 {object} ErrorBody
// @Failure 403 {object} ErrorBody
// @Failure 500 {object} ErrorBody
// @Router /admin/clients/{id} [get]
func (s *Server) client(w http.ResponseWriter, r *http.Request, _ telegram.User) {
	card, err := s.DB.Client(r.Context(), r.PathValue("id"))
	if finish(w, err) {
		return
	}
	writeOK(w, map[string]any{"data": card})
}

// Режим баланса клиента.
//
// @Summary Изменить режим баланса
// @Tags Админ
// @Security TMA
// @Param id path string true "Идентификатор клиента"
// @Param body body PolicyInput true "postpay или restricted"
// @Success 200 {object} ClientResponse
// @Failure 400 {object} ErrorBody
// @Failure 401 {object} ErrorBody
// @Failure 403 {object} ErrorBody
// @Failure 500 {object} ErrorBody
// @Router /admin/clients/{id} [patch]
func (s *Server) patchClient(w http.ResponseWriter, r *http.Request, _ telegram.User) {
	var body PolicyInput
	if err := readJSON(r, &body); err != nil {
		writeDomain(w, err)
		return
	}
	card, err := s.DB.PatchClient(r.Context(), s.now(), r.PathValue("id"), body.Policy)
	if finish(w, err) {
		return
	}
	writeOK(w, map[string]any{"data": card})
}

// Начисление занятий.
//
// @Summary Начислить занятия
// @Description Каждый успешный вызов пишет новую проводку grant. Повтор запроса повторяет начисление.
// @Tags Админ
// @Security TMA
// @Param id path string true "Идентификатор клиента"
// @Param body body GrantInput true "Число занятий и пометка"
// @Success 200 {object} ClientResponse
// @Failure 400 {object} ErrorBody
// @Failure 401 {object} ErrorBody
// @Failure 403 {object} ErrorBody
// @Failure 500 {object} ErrorBody
// @Router /admin/clients/{id}/credits [post]
func (s *Server) grant(w http.ResponseWriter, r *http.Request, user telegram.User) {
	var body GrantInput
	if err := readJSON(r, &body); err != nil || body.Delta == nil {
		writeDomain(w, domain.Fail(domain.Validation))
		return
	}
	card, err := s.DB.Grant(r.Context(), s.now(), r.PathValue("id"), *body.Delta, body.Note, strconv.FormatInt(user.ID, 10))
	if finish(w, err) {
		return
	}
	writeOK(w, map[string]any{"data": card})
}

// Сводка периода.
//
// @Summary Сводка
// @Description Даты включительно, календарь Asia/Tbilisi.
// @Tags Админ
// @Security TMA
// @Param from query string true "Начало, YYYY-MM-DD" example(2026-09-01)
// @Param to query string true "Конец, включительно" example(2026-09-30)
// @Success 200 {object} StatsResponse
// @Failure 400 {object} ErrorBody
// @Failure 401 {object} ErrorBody
// @Failure 403 {object} ErrorBody
// @Failure 500 {object} ErrorBody
// @Router /admin/stats [get]
func (s *Server) stats(w http.ResponseWriter, r *http.Request, _ telegram.User) {
	from, to := r.URL.Query().Get("from"), r.URL.Query().Get("to")
	if from == "" || to == "" {
		writeDomain(w, domain.Fail(domain.Validation))
		return
	}
	stats, err := s.DB.Stats(r.Context(), from, to)
	if finish(w, err) {
		return
	}
	writeOK(w, map[string]any{"data": stats})
}

// Выгрузка контактов.
//
// @Summary Выгрузка клиентов
// @Description CSV RFC 4180. Первая строка: name,phone,username,telegram_id,balance,policy.
// @Tags Админ
// @Security TMA
// @Produce text/csv
// @Success 200 {string} string "файл clients.csv"
// @Failure 401 {object} ErrorBody
// @Failure 403 {object} ErrorBody
// @Failure 500 {object} ErrorBody
// @Router /admin/clients.csv [get]
func (s *Server) clientsCSV(w http.ResponseWriter, r *http.Request, _ telegram.User) {
	rows, err := s.DB.ListClients(r.Context(), "")
	if finish(w, err) {
		return
	}
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="clients.csv"`)
	writer := csv.NewWriter(w)
	_ = writer.Write([]string{"name", "phone", "username", "telegram_id", "balance", "policy"})
	for _, row := range rows {
		_ = writer.Write([]string{row.Name, row.Phone, row.Username, row.TelegramID, strconv.Itoa(row.Balance), row.Policy})
	}
	writer.Flush()
}

// Карточка студии для админа. Тело совпадает с клиентским GET /studio.
//
// @Summary Контакты студии
// @Tags Админ
// @Security TMA
// @Success 200 {object} StudioResponse
// @Failure 401 {object} ErrorBody
// @Failure 403 {object} ErrorBody
// @Failure 500 {object} ErrorBody
// @Router /admin/studio [get]
func (s *Server) adminStudio(w http.ResponseWriter, r *http.Request, user telegram.User) {
	s.studio(w, r, user)
}

// Замена контактов студии.
//
// @Summary Сохранить контакты
// @Description Все поля обязательны. Пустая строка контакта стирает значение. pricing — JSON разделов base, aero, kids и personal.
// @Tags Админ
// @Security TMA
// @Param body body sqlite.Studio true "Контакты, памятка и прайс"
// @Success 200 {object} StudioResponse
// @Failure 400 {object} ErrorBody
// @Failure 401 {object} ErrorBody
// @Failure 403 {object} ErrorBody
// @Failure 500 {object} ErrorBody
// @Router /admin/studio [put]
func (s *Server) putStudio(w http.ResponseWriter, r *http.Request, _ telegram.User) {
	var raw map[string]json.RawMessage
	if err := readJSON(r, &raw); err != nil {
		writeDomain(w, err)
		return
	}
	studio, err := studioFrom(raw)
	if err != nil {
		writeDomain(w, err)
		return
	}
	saved, dbErr := s.DB.PutStudio(r.Context(), studio)
	if finish(w, dbErr) {
		return
	}
	writeOK(w, map[string]any{"data": saved})
}

// Каталог текстов, которые админ может заменить. Сейчас там приветствие /start.
// Новое уведомление появляется в этом же списке, когда его ключ добавляют в каталог.
//
// @Summary Тексты сообщений
// @Tags Админ
// @Security TMA
// @Success 200 {object} MessagesResponse
// @Failure 401 {object} ErrorBody
// @Failure 403 {object} ErrorBody
// @Failure 500 {object} ErrorBody
// @Router /admin/messages [get]
func (s *Server) messages(w http.ResponseWriter, r *http.Request, _ telegram.User) {
	data, err := s.DB.Messages(r.Context())
	if finish(w, err) {
		return
	}
	writeOK(w, map[string]any{"data": data})
}

// Замена одного текста из каталога. Неизвестный key отвечает VALIDATION.
//
// @Summary Сохранить текст сообщения
// @Description Пустой текст и неизвестный key отвечают VALIDATION. Длиннее 4000 знаков тоже.
// @Tags Админ
// @Security TMA
// @Param key path string true "Ключ каталога"
// @Param body body sqlite.MessageInput true "Текст"
// @Success 200 {object} MessageResponse
// @Failure 400 {object} ErrorBody
// @Failure 401 {object} ErrorBody
// @Failure 403 {object} ErrorBody
// @Failure 500 {object} ErrorBody
// @Router /admin/messages/{key} [put]
func (s *Server) putMessage(w http.ResponseWriter, r *http.Request, _ telegram.User) {
	var body sqlite.MessageInput
	if err := readJSON(r, &body); err != nil {
		writeDomain(w, err)
		return
	}
	saved, err := s.DB.PutMessage(r.Context(), r.PathValue("key"), body.Body)
	if finish(w, err) {
		return
	}
	writeOK(w, map[string]any{"data": saved})
}

// Резерв зала без записи.
//
// @Summary Зарезервировать время
// @Description Клиент видит «Студия занята». Админ и тренеры видят название.
// @Tags Админ
// @Security TMA
// @Param body body sqlite.HoldInput true "Название и начало"
// @Success 200 {object} ClassResponse
// @Failure 400 {object} ErrorBody
// @Failure 401 {object} ErrorBody
// @Failure 403 {object} ErrorBody
// @Failure 409 {object} ErrorBody
// @Failure 500 {object} ErrorBody
// @Router /admin/holds [post]
func (s *Server) createHold(w http.ResponseWriter, r *http.Request, _ telegram.User) {
	var body sqlite.HoldInput
	if err := readJSON(r, &body); err != nil {
		writeDomain(w, err)
		return
	}
	view, err := s.DB.CreateHold(r.Context(), s.now(), body.Title, body.StartsAt, body.DurationMinutes)
	if finish(w, err) {
		return
	}
	writeOK(w, map[string]any{"data": view})
}

// Индивидуальное с назначенным клиентом.
//
// @Summary Создать индивидуальное
// @Description В общее расписание не попадает. Клиент видит его в своих записях.
// @Tags Админ
// @Security TMA
// @Param body body sqlite.IndividualInput true "Тренер, клиент и начало"
// @Success 200 {object} ClassResponse
// @Failure 400 {object} ErrorBody
// @Failure 401 {object} ErrorBody
// @Failure 403 {object} ErrorBody
// @Failure 409 {object} ErrorBody
// @Failure 500 {object} ErrorBody
// @Router /admin/individuals [post]
func (s *Server) createIndividual(w http.ResponseWriter, r *http.Request, _ telegram.User) {
	var body sqlite.IndividualInput
	if err := readJSON(r, &body); err != nil {
		writeDomain(w, err)
		return
	}
	view, err := s.DB.CreateIndividual(r.Context(), s.now(), body.TrainerID, body.ClientID, body.StartsAt, body.DurationMinutes)
	if finish(w, err) {
		return
	}
	writeOK(w, map[string]any{"data": view})
}
