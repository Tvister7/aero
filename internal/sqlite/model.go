package sqlite

// User — человек из проверенного initData. В JSON API не сериализуется.
type User struct {
	TelegramID int64
	Username   string
}

// Trainer тренер студии.
// @Description Тренер студии.
type Trainer struct {
	ID string `json:"id" example:"6ba7b811-9dad-11d1-80b4-00c04fd430c8"`
	// Имя на экране расписания. Уникально.
	Name string `json:"name" example:"Лилия"`
	// false скрывает тренера из новых шаблонов. Уже созданные занятия остаются.
	Active bool `json:"active" example:"true"`
	// Короткий текст «о тренере». Пустая строка — не заполнено.
	About string `json:"about" example:"Йога и растяжка"`
	Phone string `json:"phone" example:"+995555000000"`
	// Telegram id клиента, если роль тренера выдана. Пусто — роли нет.
	TelegramUserID string `json:"telegram_user_id" example:"1001"`
}

// TrainingType вид занятия. Цена — подпись на экране, не число занятий на балансе.
// @Description Вид занятия. Цена — подпись на экране, не число занятий на балансе.
type TrainingType struct {
	ID    string `json:"id" example:"6ba7b812-9dad-11d1-80b4-00c04fd430c8"`
	Title string `json:"title" example:"Aero Yoga"`
	// Длительность одного слота, от 1 до 600 минут.
	DurationMinutes int `json:"duration_minutes" example:"60"`
	// Вместимость по умолчанию для новых шаблонов и разовых занятий.
	Capacity int `json:"capacity" example:"6"`
	// Нижняя подпись стоимости, два знака после точки.
	PriceMin string `json:"price_min" example:"17.00"`
	// Верхняя подпись стоимости, не меньше price_min.
	PriceMax string `json:"price_max" example:"40.00"`
	Active   bool   `json:"active" example:"true"`
	// Цвет индикации. Назначается при создании и не меняется, когда другие типы скрывают.
	Color string `json:"color" example:"#5c7a3a"`
}

// Template еженедельный слот. weekday: 1 понедельник … 7 воскресенье. start_time — местное HH:MM.
// @Description Еженедельный слот. weekday: 1 понедельник … 7 воскресенье. start_time — местное HH:MM.
type Template struct {
	ID             string `json:"id" example:"6ba7b813-9dad-11d1-80b4-00c04fd430c8"`
	TrainerID      string `json:"trainer_id" example:"6ba7b811-9dad-11d1-80b4-00c04fd430c8"`
	TrainingTypeID string `json:"training_type_id" example:"6ba7b812-9dad-11d1-80b4-00c04fd430c8"`
	Title          string `json:"title" example:"Aero Yoga"`
	Weekday        int    `json:"weekday" example:"4"`
	StartTime      string `json:"start_time" example:"19:00"`
	Capacity       int    `json:"capacity" example:"6"`
	Active         bool   `json:"active" example:"true"`
}

// TrainerRef тренер внутри занятия.
// @Description Тренер внутри занятия.
type TrainerRef struct {
	ID   string `json:"id" example:"6ba7b811-9dad-11d1-80b4-00c04fd430c8"`
	Name string `json:"name" example:"Лилия"`
}

// ClassView занятие в ответе API.
// У клиента поля status нет. В админском расписании, создании, правке и журнале status равен scheduled или cancelled.
// price_min и price_max — снимок подписи на момент слота. На баланс они не переводятся.
// @Description Занятие в ответе API. У клиента поля status нет. В админском расписании, создании, правке и журнале status равен scheduled или cancelled. price_min и price_max — снимок подписи на момент слота. На баланс они не переводятся.
type ClassView struct {
	ID      string     `json:"id" example:"6ba7b810-9dad-11d1-80b4-00c04fd430c8"`
	Title   string     `json:"title" example:"Aero Yoga"`
	Trainer TrainerRef `json:"trainer"`
	// Начало, Asia/Tbilisi, ISO-8601 со смещением +04:00.
	StartsAt string `json:"starts_at" example:"2026-09-03T19:00:00+04:00"`
	EndsAt   string `json:"ends_at" example:"2026-09-03T20:00:00+04:00"`
	Capacity int    `json:"capacity" example:"6"`
	// Число записей со статусом booked.
	Booked     int    `json:"booked" example:"2"`
	FreePlaces int    `json:"free_places" example:"4"`
	PriceMin   string `json:"price_min" example:"17.00"`
	PriceMax   string `json:"price_max" example:"40.00"`
	// Цвет типа занятия. Хранится на типе, поэтому скрытие другого типа его не меняет.
	Color string `json:"color" example:"#5c7a3a"`
	// Вид занятия. Перенос записи возможен только на тот же вид.
	TrainingTypeID string `json:"training_type_id" example:"6ba7b812-9dad-11d1-80b4-00c04fd430c8"`
	// Шаблон, из которого собрано занятие. Пусто у разового слота: только его можно перенести.
	TemplateID *string `json:"template_id,omitempty" example:"6ba7b813-9dad-11d1-80b4-00c04fd430c8"`
	// Только админские ответы. У клиента поле отсутствует.
	Status *string `json:"status,omitempty" example:"scheduled" enums:"scheduled,cancelled"`
	// group — обычное занятие, individual — закрытый час одного клиента, hold — резерв зала.
	Kind string `json:"kind" example:"group" enums:"group,individual,hold"`
}

// ClientName имя клиента для выбора на индивидуальное. Телефон сюда не входит.
// @Description Имя клиента для выбора на индивидуальное. Телефон сюда не входит.
type ClientName struct {
	ID   string `json:"id" example:"6ba7b815-9dad-11d1-80b4-00c04fd430c8"`
	Name string `json:"name" example:"Анна"`
}

// HoldInput резерв зала. duration_minutes по умолчанию 60.
// @Description Резерв зала. duration_minutes по умолчанию 60.
type HoldInput struct {
	Title           string `json:"title" example:"Съёмка"`
	StartsAt        string `json:"starts_at" example:"2026-09-12T19:00:00+04:00"`
	DurationMinutes *int   `json:"duration_minutes" example:"60"`
}

// IndividualInput закрытое занятие с уже выбранным клиентом. duration_minutes по умолчанию 60.
// @Description Закрытое занятие с уже выбранным клиентом. duration_minutes по умолчанию 60.
type IndividualInput struct {
	TrainerID       string `json:"trainer_id" example:"6ba7b811-9dad-11d1-80b4-00c04fd430c8"`
	ClientID        string `json:"client_id" example:"6ba7b815-9dad-11d1-80b4-00c04fd430c8"`
	StartsAt        string `json:"starts_at" example:"2026-09-12T19:00:00+04:00"`
	DurationMinutes *int   `json:"duration_minutes" example:"60"`
}

// BookOutcome результат записи. HTTP добавляет сверху ok.
// @Description Результат записи. HTTP добавляет сверху ok.
type BookOutcome struct {
	BookingID string `json:"booking_id" example:"6ba7b814-9dad-11d1-80b4-00c04fd430c8"`
	// Баланс в занятиях после вызова. Сама запись баланс не меняет.
	Balance int       `json:"balance" example:"4"`
	Class   ClassView `json:"class"`
}

// MyBooking будущая запись клиента.
// @Description Будущая запись клиента.
type MyBooking struct {
	BookingID string    `json:"booking_id" example:"6ba7b814-9dad-11d1-80b4-00c04fd430c8"`
	Class     ClassView `json:"class"`
}

// Visit состоявшееся посещение. payment paid — занятие списано, unpaid — явка без покрытия.
// @Description Состоявшееся посещение. payment paid — занятие списано, unpaid — явка без покрытия.
type Visit struct {
	Title       string `json:"title" example:"Aero Yoga"`
	TrainerName string `json:"trainer_name" example:"Лилия"`
	StartsAt    string `json:"starts_at" example:"2026-09-03T19:00:00+04:00"`
	Payment     string `json:"payment" example:"paid" enums:"paid,unpaid"`
}

// AttendanceView активная явка. payment: paid, если до списания баланс был больше нуля, иначе unpaid.
// @Description Активная явка. payment: paid, если до списания баланс был больше нуля, иначе unpaid.
type AttendanceView struct {
	Payment  string `json:"payment" example:"paid" enums:"paid,unpaid"`
	MarkedAt string `json:"marked_at" example:"2026-09-03T19:05:00+04:00"`
}

// AttendOutcome результат отметки явки. attendance равен null, когда активной явки нет.
// @Description Результат отметки явки. attendance равен null, когда активной явки нет.
type AttendOutcome struct {
	BookingID  string          `json:"booking_id" example:"6ba7b814-9dad-11d1-80b4-00c04fd430c8"`
	Attendance *AttendanceView `json:"attendance" extensions:"x-nullable"`
	// Баланс в занятиях. Долг — отрицательное число. Повтор той же отметки баланс не меняет.
	Balance int `json:"balance" example:"3"`
}

// Me профиль человека, открывшего Mini App. id наружу не отдаётся.
// @Description Профиль человека, открывшего Mini App. id наружу не отдаётся.
type Me struct {
	ID       string `json:"-"`
	Name     string `json:"name" example:"Анна"`
	Phone    string `json:"phone" example:"+995555000000"`
	Username string `json:"username" example:"anna"`
	// Сумма проводок. Запись на занятие это число не уменьшает.
	Balance int `json:"balance" example:"4"`
	// postpay разрешает запись в долг. restricted запрещает запись при балансе 0 и ниже.
	Policy string `json:"policy" example:"postpay" enums:"postpay,restricted"`
	// Была хотя бы одна явка, в том числе позже снятая.
	Visited bool `json:"visited" example:"false"`
	IsAdmin bool `json:"is_admin" example:"false"`
	// Telegram id совпадает с активным тренером.
	IsTrainer bool `json:"is_trainer" example:"false"`
}

// ClientSummary строка списка клиентов.
// @Description Строка списка клиентов.
type ClientSummary struct {
	ID         string `json:"id" example:"6ba7b815-9dad-11d1-80b4-00c04fd430c8"`
	TelegramID string `json:"telegram_id" example:"1001"`
	Username   string `json:"username" example:"anna"`
	Name       string `json:"name" example:"Анна"`
	Phone      string `json:"phone" example:"+995555000000"`
	Policy     string `json:"policy" example:"postpay" enums:"postpay,restricted"`
	Balance    int    `json:"balance" example:"4"`
}

// LedgerView проводка баланса. delta +N начисляет занятия, −1 списывает явку, +1 возвращает снятие явки.
// booking_id пустой у начисления.
// @Description Проводка баланса. delta +N начисляет занятия, −1 списывает явку, +1 возвращает снятие явки. booking_id пустой у начисления.
type LedgerView struct {
	ID        string  `json:"id" example:"6ba7b816-9dad-11d1-80b4-00c04fd430c8"`
	Delta     int     `json:"delta" example:"8"`
	Reason    string  `json:"reason" example:"grant" enums:"grant,attend,reversal"`
	Note      string  `json:"note" example:"пакет сентябрь"`
	BookingID *string `json:"booking_id" extensions:"x-nullable" example:"6ba7b814-9dad-11d1-80b4-00c04fd430c8"`
	CreatedAt string  `json:"created_at" example:"2026-09-01T12:00:00+04:00"`
}

// ClientCard клиент и его проводки, новые сверху.
// @Description Клиент и его проводки, новые сверху.
type ClientCard struct {
	ClientSummary
	Ledger []LedgerView `json:"ledger"`
}

// Stats сводка за включительный диапазон дат Asia/Tbilisi.
// @Description Сводка за включительный диапазон дат Asia/Tbilisi.
type Stats struct {
	From              string `json:"from" example:"2026-09-01"`
	To                string `json:"to" example:"2026-09-30"`
	UniqueUsers       int    `json:"unique_users" example:"40"`
	BookingsCreated   int    `json:"bookings_created" example:"90"`
	BookingsCancelled int    `json:"bookings_cancelled" example:"12"`
	Arrivals          int    `json:"arrivals" example:"70"`
	NoShows           int    `json:"no_shows" example:"8"`
	UnpaidVisits      int    `json:"unpaid_visits" example:"5"`
}

// Studio контакты. Пустая строка значит «не заполнено». В PUT нужны все пять полей.
// @Description Контакты. Пустая строка значит «не заполнено». В PUT нужны все пять полей.
type Studio struct {
	Address      string `json:"address" example:"ул. Пример, 1"`
	Phone        string `json:"phone" example:"+995555000000"`
	InstagramURL string `json:"instagram_url" example:"https://instagram.com/aero"`
	ContactURL   string `json:"contact_url" example:"https://t.me/aero"`
	MapURL       string `json:"map_url" example:"https://maps.example/aero"`
}

// RosterBooking одна запись в журнале занятия. attendance равен null, пока явку не поставили.
// @Description Одна запись в журнале занятия. attendance равен null, пока явку не поставили.
type RosterBooking struct {
	BookingID  string          `json:"booking_id" example:"6ba7b814-9dad-11d1-80b4-00c04fd430c8"`
	ClientID   string          `json:"client_id" example:"6ba7b815-9dad-11d1-80b4-00c04fd430c8"`
	Name       string          `json:"name" example:"Анна"`
	Phone      string          `json:"phone" example:"+995555000000"`
	Username   string          `json:"username" example:"anna"`
	Attendance *AttendanceView `json:"attendance" extensions:"x-nullable"`
}

// Roster журнал занятия: слот и записи со статусом booked.
// @Description Журнал занятия: слот и записи со статусом booked.
type Roster struct {
	Class    ClassView       `json:"class"`
	Bookings []RosterBooking `json:"bookings"`
}

// WeekOutcome итог генерации недели. created — новые слоты, updated — уже стоявшие scheduled.
// @Description Итог генерации недели. created — новые слоты, updated — уже стоявшие scheduled.
type WeekOutcome struct {
	WeekStart string `json:"week_start" example:"2026-09-14"`
	WeekEnd   string `json:"week_end" example:"2026-09-20"`
	Created   int    `json:"created" example:"8"`
	Updated   int    `json:"updated" example:"2"`
}

// Outbound сообщение бота. Наружу HTTP его не отдаёт: процесс шлёт его после коммита.
type Outbound struct {
	TelegramID int64
	Text       string
}

// ClassChange результат правки слота вместе с текстами, которые уйдут записанным.
type ClassChange struct {
	Class   ClassView
	Notices []Outbound
}

// ClassInput разовое занятие. capacity можно не передавать: берётся вместимость типа.
// @Description Разовое занятие. capacity можно не передавать: берётся вместимость типа.
type ClassInput struct {
	TrainerID string `json:"trainer_id" example:"6ba7b811-9dad-11d1-80b4-00c04fd430c8"`
	TypeID    string `json:"training_type_id" example:"6ba7b812-9dad-11d1-80b4-00c04fd430c8"`
	StartsAt  string `json:"starts_at" example:"2026-09-12T19:00:00+04:00"`
	Capacity  *int   `json:"capacity" example:"6"`
}

// ClassPatch перенос слота или его отмена.
// Если status равен cancelled, остальные поля игнорируются. Вернуть слот в scheduled этим объектом нельзя.
// capacity ниже числа текущих записей не принимается.
// @Description Перенос слота или его отмена. Если status равен cancelled, остальные поля игнорируются. Вернуть слот в scheduled этим объектом нельзя. capacity ниже числа текущих записей не принимается.
type ClassPatch struct {
	Status    *string `json:"status" enums:"cancelled" example:"cancelled"`
	StartsAt  *string `json:"starts_at" example:"2026-09-12T20:00:00+04:00"`
	TrainerID *string `json:"trainer_id" example:"6ba7b811-9dad-11d1-80b4-00c04fd430c8"`
	Capacity  *int    `json:"capacity" example:"8"`
	Title     *string `json:"title" example:"Aero Yoga"`
}

// TypeInput поля вида занятия без id.
// При создании нужны title, duration_minutes, capacity, price_min и price_max.
// При правке передаются только меняющиеся поля. Уже созданные слоты правка не переписывает.
// @Description Поля вида занятия без id. При создании нужны title, duration_minutes, capacity, price_min и price_max. При правке передаются только меняющиеся поля. Уже созданные слоты правка не переписывает.
type TypeInput struct {
	Title           *string `json:"title" example:"Aero Stretching"`
	DurationMinutes *int    `json:"duration_minutes" example:"60"`
	Capacity        *int    `json:"capacity" example:"6"`
	PriceMin        *string `json:"price_min" example:"17.00"`
	PriceMax        *string `json:"price_max" example:"40.00"`
	Active          *bool   `json:"active" example:"true"`
}

// TemplateInput поля шаблона.
// При создании нужны trainer_id, training_type_id, weekday и start_time. capacity можно опустить.
// title копируется с типа. Правка шаблона не меняет уже созданные занятия.
// @Description Поля шаблона. При создании нужны trainer_id, training_type_id, weekday и start_time. capacity можно опустить. title копируется с типа. Правка шаблона не меняет уже созданные занятия.
type TemplateInput struct {
	TrainerID      *string `json:"trainer_id" example:"6ba7b811-9dad-11d1-80b4-00c04fd430c8"`
	TrainingTypeID *string `json:"training_type_id" example:"6ba7b812-9dad-11d1-80b4-00c04fd430c8"`
	Weekday        *int    `json:"weekday" example:"4"`
	StartTime      *string `json:"start_time" example:"19:00"`
	Capacity       *int    `json:"capacity" example:"6"`
	Active         *bool   `json:"active" example:"true"`
}

// TrainerFields правка карточки тренера. Отсутствующее поле не меняется.
// Пустой telegram_user_id снимает привязку.
// @Description Правка карточки тренера. Отсутствующее поле не меняется. Пустой telegram_user_id снимает привязку.
type TrainerFields struct {
	Name           *string `json:"name" example:"Лилия"`
	Active         *bool   `json:"active" example:"false"`
	About          *string `json:"about" example:"Йога и растяжка"`
	Phone          *string `json:"phone" example:"+995555000000"`
	TelegramUserID *string `json:"telegram_user_id" example:"1001"`
}

// Absence отпуск тренера, даты включительно, Asia/Tbilisi.
// @Description Отпуск тренера, даты включительно, Asia/Tbilisi.
type Absence struct {
	ID       string `json:"id" example:"6ba7b817-9dad-11d1-80b4-00c04fd430c8"`
	StartsOn string `json:"starts_on" example:"2026-12-01"`
	EndsOn   string `json:"ends_on" example:"2026-12-07"`
}

// TrainerDetail карточка тренера и его отпуска.
// @Description Карточка тренера и его отпуска.
type TrainerDetail struct {
	Trainer
	Absences []Absence `json:"absences"`
}

// AbsenceInput новый отпуск. Обе даты обязательны, конец не раньше начала.
// @Description Новый отпуск. Обе даты обязательны, конец не раньше начала.
type AbsenceInput struct {
	StartsOn string `json:"starts_on" example:"2026-12-01"`
	EndsOn   string `json:"ends_on" example:"2026-12-07"`
}

// EnrollmentOption активный шаблон, который клиент может отметить.
// @Description Активный шаблон, который клиент может отметить.
type EnrollmentOption struct {
	TemplateID  string `json:"template_id" example:"6ba7b813-9dad-11d1-80b4-00c04fd430c8"`
	TrainerID   string `json:"trainer_id" example:"6ba7b811-9dad-11d1-80b4-00c04fd430c8"`
	TrainerName string `json:"trainer_name" example:"Лилия"`
	Title       string `json:"title" example:"Aero Yoga"`
	Weekday     int    `json:"weekday" example:"4"`
	StartTime   string `json:"start_time" example:"19:00"`
	Enrolled    bool   `json:"enrolled" example:"true"`
}

// EnrollmentInput полный набор отмеченных шаблонов. Пустой список снимает все отметки.
// @Description Полный набор отмеченных шаблонов. Пустой список снимает все отметки.
type EnrollmentInput struct {
	TemplateIDs []string `json:"template_ids"`
}

// TrainerProfile текст и телефон, которые тренер правит сам. Имя здесь не меняется.
// @Description Текст и телефон, которые тренер правит сам. Имя здесь не меняется.
type TrainerProfile struct {
	About string `json:"about" example:"Йога и растяжка"`
	Phone string `json:"phone" example:"+995555000000"`
}

// TrainerStats сводка тренера за включительный диапазон дат Asia/Tbilisi.
// clients — разные клиенты с активной явкой на его занятиях.
// attends — проводки attend по этим занятиям.
// @Description Сводка тренера за включительный диапазон дат Asia/Tbilisi. clients — разные клиенты с активной явкой на его занятиях. attends — проводки attend по этим занятиям.
type TrainerStats struct {
	From    string `json:"from" example:"2026-09-01"`
	To      string `json:"to" example:"2026-09-30"`
	Clients int    `json:"clients" example:"12"`
	Attends int    `json:"attends" example:"20"`
}
