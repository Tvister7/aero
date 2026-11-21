package domain

import (
	"fmt"
	"strings"
	"time"
)

var Tbilisi = time.FixedZone("Asia/Tbilisi", 4*60*60)

const day = "2006-01-02"

func DateKey(t time.Time) string {
	return t.In(Tbilisi).Format(day)
}

func FormatISO(t time.Time) string {
	return t.In(Tbilisi).Format("2006-01-02T15:04:05Z07:00")
}

func ParseInstant(value string) (time.Time, *Error) {
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return time.Time{}, Fail(Validation)
	}
	return parsed, nil
}

func ParseDay(value string) (time.Time, *Error) {
	parsed, err := time.ParseInLocation(day, value, Tbilisi)
	if err != nil {
		return time.Time{}, Fail(Validation)
	}
	return parsed, nil
}

func midnight(t time.Time) time.Time {
	local := t.In(Tbilisi)
	return time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, Tbilisi)
}

func isoWeekday(t time.Time) int {
	weekday := int(t.In(Tbilisi).Weekday())
	if weekday == 0 {
		return 7
	}
	return weekday
}

func MondayOf(t time.Time) time.Time {
	start := midnight(t)
	return start.AddDate(0, 0, -(isoWeekday(start) - 1))
}

func NextMonday(now time.Time) time.Time {
	return MondayOf(now).AddDate(0, 0, 7)
}

func IsMonday0010(now time.Time) bool {
	local := now.In(Tbilisi)
	return local.Weekday() == time.Monday && local.Hour() == 0 && local.Minute() == 10
}

func IsBackupTime(now time.Time) bool {
	local := now.In(Tbilisi)
	return local.Hour() == 3 && local.Minute() == 0
}

func ParseClock(value string) (hour, minute int, err *Error) {
	if len(value) == 8 && value[2] == ':' && value[5] == ':' {
		value = value[:5]
	}
	if len(value) != 5 || value[2] != ':' {
		return 0, 0, Fail(Validation)
	}
	if _, scanErr := fmt.Sscanf(value, "%02d:%02d", &hour, &minute); scanErr != nil || hour > 23 || minute > 59 {
		return 0, 0, Fail(Validation)
	}
	return hour, minute, nil
}

func SlotStart(monday time.Time, weekday int, clock string) (time.Time, *Error) {
	if weekday < 1 || weekday > 7 {
		return time.Time{}, Fail(Validation)
	}
	hour, minute, err := ParseClock(clock)
	if err != nil {
		return time.Time{}, err
	}
	dayStart := midnight(monday).AddDate(0, 0, weekday-1)
	return time.Date(dayStart.Year(), dayStart.Month(), dayStart.Day(), hour, minute, 0, 0, Tbilisi), nil
}

func when(t time.Time) string {
	return t.In(Tbilisi).Format("02.01 15:04")
}

func ReminderText(title string, starts time.Time) string {
	return fmt.Sprintf("Напоминание: %s начнётся %s.", title, when(starts))
}

// PlaceLink — ссылка, как найти студию. Карта важнее страницы памятки.
func PlaceLink(mapURL, origin string) string {
	if link := strings.TrimSpace(mapURL); link != "" {
		return link
	}
	origin = strings.TrimRight(strings.TrimSpace(origin), "/")
	if origin == "" {
		return ""
	}
	return origin + "/#/studio/guide"
}

func FirstClassText(title string, starts time.Time, link string) string {
	text := fmt.Sprintf("Вы записаны на первое занятие: %s, %s.", title, when(starts))
	if link != "" {
		text += "\nКак нас найти: " + link
	}
	return text + "\nВ день занятия администратор напишет вам код от подъезда."
}

func AdminFirstBookingText(name, username, phone, title string, starts time.Time) string {
	who := strings.TrimSpace(name)
	if who == "" {
		who = "без имени"
	}
	username = strings.TrimPrefix(strings.TrimSpace(username), "@")
	if username != "" {
		who += ", @" + username
	}
	if phone = strings.TrimSpace(phone); phone != "" {
		who += ", " + phone
	}
	return fmt.Sprintf("Первая запись: %s.\n%s, %s.\nНапишите код от подъезда в день занятия.", who, title, when(starts))
}

func MoveText(title string, starts time.Time) string {
	return fmt.Sprintf("Занятие %s перенесено на %s.", title, when(starts))
}

func CancelText(title string, starts time.Time) string {
	return fmt.Sprintf("Занятие %s %s отменено.", title, when(starts))
}

func EmptyBalanceText(booked bool, title string, starts time.Time) string {
	if booked {
		return fmt.Sprintf("Вы записаны на %s %s, занятия на балансе закончились.", title, when(starts))
	}
	return fmt.Sprintf("Вас не записали на %s %s, занятия на балансе закончились.", title, when(starts))
}
