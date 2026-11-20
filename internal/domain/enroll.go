package domain

import "time"

// Allowance — сколько ещё записей можно поставить до занятия.
// У абонемента запас равен балансу, у постоплаты — балансу плюс одна запись.
// Будущие записи без явки этот запас уже занимают.
func Allowance(policy string, balance, futureUnattended int) int {
	extra := 0
	if policy != Restricted {
		extra = 1
	}
	left := balance + extra - futureUnattended
	if left < 0 {
		return 0
	}
	return left
}

type SlotChoice struct {
	Class       Class
	BookedCount int
	Already     bool
}

type SlotDecision struct {
	Class  Class
	Book   bool
	Notice string
}

// DecideEnrollments идёт по слотам в переданном порядке, обычно по времени начала.
// Уже стоящая запись и занятый зал запас не тратят и предупреждения не ставят.
// Предупреждение о пустом балансе получает запись в долг.
// Пропуск из-за абонемента получает отдельное предупреждение.
func DecideEnrollments(policy string, balance, allowance int, slots []SlotChoice) []SlotDecision {
	left := allowance
	out := make([]SlotDecision, 0, len(slots))
	for _, slot := range slots {
		decision := SlotDecision{Class: slot.Class}
		if slot.Already || slot.Class.Status != Scheduled || slot.BookedCount >= slot.Class.Capacity {
			out = append(out, decision)
			continue
		}
		if left <= 0 {
			if policy == Restricted {
				decision.Notice = NoticeSkippedEmpty
			}
			out = append(out, decision)
			continue
		}
		decision.Book = true
		left--
		if balance <= 0 {
			decision.Notice = NoticeBookedEmpty
		}
		out = append(out, decision)
	}
	return out
}

type BalanceCandidate struct {
	ID          string
	Kind        string
	State       string
	NoticeAt    time.Time
	ClassStatus string
	StartsAt    time.Time
	Title       string
	Policy      string
	Balance     int
	Future      int
	Booked      bool
}

type BalanceDecision struct {
	Send []BalanceCandidate
	Skip []string
}

func BalanceNotices(now time.Time, items []BalanceCandidate) BalanceDecision {
	var decision BalanceDecision
	for _, item := range items {
		if item.State != ReminderPending {
			continue
		}
		if item.ClassStatus != Scheduled || !item.StartsAt.After(now) {
			decision.Skip = append(decision.Skip, item.ID)
			continue
		}
		applies := false
		if item.Kind == NoticeBookedEmpty {
			applies = item.Booked && item.Balance <= 0
		}
		if item.Kind == NoticeSkippedEmpty {
			applies = !item.Booked && item.Policy == Restricted && Allowance(item.Policy, item.Balance, item.Future) <= 0
		}
		if !applies {
			decision.Skip = append(decision.Skip, item.ID)
			continue
		}
		if !item.NoticeAt.After(now) {
			decision.Send = append(decision.Send, item)
		}
	}
	return decision
}
