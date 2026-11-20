package domain

import "time"

// Blocks сообщает, что интервал пересекает другой scheduled-слот.
// exclusive выключен, когда у студии несколько залов: проверку тогда не делаем.
func Blocks(exclusive bool, start, end time.Time, existing []ClassSnapshot, ignoreID string) bool {
	if !exclusive || !end.After(start) {
		return false
	}
	for _, item := range existing {
		class := item.Class
		if class.ID == ignoreID || class.Status != Scheduled {
			continue
		}
		if start.Before(class.EndsAt) && end.After(class.StartsAt) {
			return true
		}
	}
	return false
}
