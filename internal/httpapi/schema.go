package httpapi

import "github.com/Tvister7/aero/internal/sqlite"

// ErrorBody ошибка API. ok всегда false.
// Тексты: UNAUTHORIZED «Откройте приложение заново из Telegram»,
// FORBIDDEN «Недостаточно прав», VALIDATION «Проверьте поля»,
// CLASS_NOT_FOUND «Занятие не найдено», NEW_CLASS_NOT_FOUND «Новое занятие недоступно»,
// BOOKING_NOT_FOUND «Запись не найдена», CLASS_STARTED «Занятие уже началось»,
// NO_PLACES «Свободных мест нет», ALREADY_BOOKED «Вы уже записаны»,
// CUTOFF_EXCEEDED «Отмена и перенос доступны не позднее чем за 6 часов»,
// SAME_CLASS «Это то же занятие», NO_CREDITS «Занятия на балансе закончились»,
// ATTENDANCE_TOO_EARLY «Отметить явку можно за 30 минут до начала»,
// INTERNAL «Не удалось выполнить запрос».
// @Description Ошибка API. ok всегда false. Тексты: UNAUTHORIZED «Откройте приложение заново из Telegram», FORBIDDEN «Недостаточно прав», VALIDATION «Проверьте поля», CLASS_NOT_FOUND «Занятие не найдено», NEW_CLASS_NOT_FOUND «Новое занятие недоступно», BOOKING_NOT_FOUND «Запись не найдена», CLASS_STARTED «Занятие уже началось», NO_PLACES «Свободных мест нет», ALREADY_BOOKED «Вы уже записаны», CUTOFF_EXCEEDED «Отмена и перенос доступны не позднее чем за 6 часов», SAME_CLASS «Это то же занятие», NO_CREDITS «Занятия на балансе закончились», ATTENDANCE_TOO_EARLY «Отметить явку можно за 30 минут до начала», INTERNAL «Не удалось выполнить запрос».
type ErrorBody struct {
	OK        bool   `json:"ok" example:"false"`
	ErrorCode string `json:"error_code" example:"NO_CREDITS" enums:"UNAUTHORIZED,FORBIDDEN,VALIDATION,CLASS_NOT_FOUND,NEW_CLASS_NOT_FOUND,BOOKING_NOT_FOUND,CLASS_STARTED,NO_PLACES,ALREADY_BOOKED,CUTOFF_EXCEEDED,SAME_CLASS,NO_CREDITS,ATTENDANCE_TOO_EARLY,INTERNAL"`
	Message   string `json:"message" example:"Занятия на балансе закончились"`
}

// ProfileInput имя и телефон. Пустая строка и отсутствие поля не стирают сохранённое значение.
// Ник из этого объекта не читается: он берётся из initData.
// @Description Имя и телефон. Пустая строка и отсутствие поля не стирают сохранённое значение. Ник из этого объекта не читается: он берётся из initData.
type ProfileInput struct {
	Name  string `json:"name" example:"Анна"`
	Phone string `json:"phone" example:"+995555000000"`
}

// TrainerInput новый тренер. После создания active равен true.
// @Description Новый тренер. После создания active равен true.
type TrainerInput struct {
	Name string `json:"name" example:"Лилия"`
}

// TrainerPatch правка тренера. Отсутствующее поле не меняется. Чужое занятое имя отклоняется.
// @Description Правка тренера. Отсутствующее поле не меняется. Чужое занятое имя отклоняется.
type TrainerPatch struct {
	Name   *string `json:"name" example:"Лилия"`
	Active *bool   `json:"active" example:"false"`
}

// RescheduleInput занятие, на которое переносится запись.
// @Description Занятие, на которое переносится запись.
type RescheduleInput struct {
	ClassID string `json:"class_id" example:"6ba7b810-9dad-11d1-80b4-00c04fd430c8"`
}

// WeekInput неделя генерации. Пустой week_start — понедельник следующей ISO-недели в Тбилиси.
// Переданная дата приводится к понедельнику своей недели.
// @Description Неделя генерации. Пустой week_start — понедельник следующей ISO-недели в Тбилиси. Переданная дата приводится к понедельнику своей недели.
type WeekInput struct {
	WeekStart string `json:"week_start" example:"2026-09-14"`
}

// AttendanceInput отметка явки. true ставит или подтверждает явку, false снимает активную.
// Поле обязательно. Повтор true и снятие при отсутствии явки баланс не меняют.
// @Description Отметка явки. true ставит или подтверждает явку, false снимает активную. Поле обязательно. Повтор true и снятие при отсутствии явки баланс не меняют.
type AttendanceInput struct {
	Arrived *bool `json:"arrived" example:"true"`
}

// PolicyInput режим баланса: postpay или restricted.
// @Description Режим баланса: postpay или restricted.
type PolicyInput struct {
	Policy string `json:"policy" example:"restricted" enums:"postpay,restricted"`
}

// GrantInput начисление занятий. delta — целое не меньше 1. Каждый вызов пишет новую проводку grant.
// @Description Начисление занятий. delta — целое не меньше 1. Каждый вызов пишет новую проводку grant.
type GrantInput struct {
	Delta *int   `json:"delta" example:"8"`
	Note  string `json:"note" example:"пакет сентябрь"`
}

// StudioResponse ответ с карточкой студии.
// @Description Ответ с карточкой студии.
type StudioResponse struct {
	OK   bool          `json:"ok" example:"true"`
	Data sqlite.Studio `json:"data"`
}

// ClassResponse ответ с одним занятием.
// @Description Ответ с одним занятием.
type ClassResponse struct {
	OK   bool             `json:"ok" example:"true"`
	Data sqlite.ClassView `json:"data"`
}

// ClassesResponse ответ со списком занятий по starts_at.
// @Description Ответ со списком занятий по starts_at.
type ClassesResponse struct {
	OK   bool               `json:"ok" example:"true"`
	Data []sqlite.ClassView `json:"data"`
}

// MeResponse ответ с профилем.
// @Description Ответ с профилем.
type MeResponse struct {
	OK   bool      `json:"ok" example:"true"`
	Data sqlite.Me `json:"data"`
}

// BookedResponse запись создана. Баланс в этом ответе ещё прежний.
// @Description Запись создана. Баланс в этом ответе ещё прежний.
type BookedResponse struct {
	OK        bool             `json:"ok" example:"true"`
	BookingID string           `json:"booking_id" example:"6ba7b814-9dad-11d1-80b4-00c04fd430c8"`
	Balance   int              `json:"balance" example:"4"`
	Class     sqlite.ClassView `json:"class"`
}

// OpenedResponse отметка открытия приложения принята.
// @Description Отметка открытия приложения принята.
type OpenedResponse struct {
	OK bool `json:"ok" example:"true"`
}

// MyBookingsResponse будущие записи клиента.
// @Description Будущие записи клиента.
type MyBookingsResponse struct {
	OK   bool               `json:"ok" example:"true"`
	Data []sqlite.MyBooking `json:"data"`
}

// CancelledResponse запись отменена.
// @Description Запись отменена.
type CancelledResponse struct {
	OK        bool   `json:"ok" example:"true"`
	BookingID string `json:"booking_id" example:"6ba7b814-9dad-11d1-80b4-00c04fd430c8"`
}

// RescheduledResponse запись перенесена на другое занятие.
// @Description Запись перенесена на другое занятие.
type RescheduledResponse struct {
	OK        bool             `json:"ok" example:"true"`
	BookingID string           `json:"booking_id" example:"6ba7b814-9dad-11d1-80b4-00c04fd430c8"`
	Class     sqlite.ClassView `json:"class"`
}

// TrainersResponse список тренеров.
// @Description Список тренеров.
type TrainersResponse struct {
	OK   bool             `json:"ok" example:"true"`
	Data []sqlite.Trainer `json:"data"`
}

// TrainerResponse один тренер.
// @Description Один тренер.
type TrainerResponse struct {
	OK   bool           `json:"ok" example:"true"`
	Data sqlite.Trainer `json:"data"`
}

// TypesResponse список видов занятий.
// @Description Список видов занятий.
type TypesResponse struct {
	OK   bool                  `json:"ok" example:"true"`
	Data []sqlite.TrainingType `json:"data"`
}

// TypeResponse один вид занятия.
// @Description Один вид занятия.
type TypeResponse struct {
	OK   bool                `json:"ok" example:"true"`
	Data sqlite.TrainingType `json:"data"`
}

// TemplatesResponse список шаблонов.
// @Description Список шаблонов.
type TemplatesResponse struct {
	OK   bool              `json:"ok" example:"true"`
	Data []sqlite.Template `json:"data"`
}

// TemplateResponse один шаблон.
// @Description Один шаблон.
type TemplateResponse struct {
	OK   bool            `json:"ok" example:"true"`
	Data sqlite.Template `json:"data"`
}

// WeekResponse итог генерации недели. Поля лежат рядом с ok, не внутри data.
// @Description Итог генерации недели. Поля лежат рядом с ok, не внутри data.
type WeekResponse struct {
	OK        bool   `json:"ok" example:"true"`
	WeekStart string `json:"week_start" example:"2026-09-14"`
	WeekEnd   string `json:"week_end" example:"2026-09-20"`
	Created   int    `json:"created" example:"8"`
	Updated   int    `json:"updated" example:"2"`
}

// RosterResponse журнал занятия.
// @Description Журнал занятия.
type RosterResponse struct {
	OK   bool          `json:"ok" example:"true"`
	Data sqlite.Roster `json:"data"`
}

// AttendanceResponse явка после отметки. attendance равен null, если активной явки нет.
// @Description Явка после отметки. attendance равен null, если активной явки нет.
type AttendanceResponse struct {
	OK         bool                   `json:"ok" example:"true"`
	BookingID  string                 `json:"booking_id" example:"6ba7b814-9dad-11d1-80b4-00c04fd430c8"`
	Attendance *sqlite.AttendanceView `json:"attendance" extensions:"x-nullable"`
	Balance    int                    `json:"balance" example:"3"`
}

// ClientsResponse список клиентов.
// @Description Список клиентов.
type ClientsResponse struct {
	OK   bool                   `json:"ok" example:"true"`
	Data []sqlite.ClientSummary `json:"data"`
}

// ClientResponse карточка клиента.
// @Description Карточка клиента.
type ClientResponse struct {
	OK   bool              `json:"ok" example:"true"`
	Data sqlite.ClientCard `json:"data"`
}

// StatsResponse сводка периода.
// @Description Сводка периода.
type StatsResponse struct {
	OK   bool         `json:"ok" example:"true"`
	Data sqlite.Stats `json:"data"`
}
