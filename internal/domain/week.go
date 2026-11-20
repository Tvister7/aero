package domain

import "time"

type TemplateSlot struct {
	ID              string
	TrainerID       string
	TypeID          string
	Title           string
	Weekday         int
	StartTime       string
	Capacity        int
	DurationMinutes int
	PriceMin        string
	PriceMax        string
	Active          bool
	TrainerActive   bool
}

type ClassSnapshot struct {
	Class  Class
	Booked int
}

type WeekResult struct {
	Created int
	Updated int
	Upserts []Class
}

type Absence struct {
	TrainerID string
	StartsOn  string
	EndsOn    string
}

func OnLeave(absences []Absence, trainerID string, start time.Time) bool {
	day := DateKey(start)
	for _, item := range absences {
		if item.TrainerID == trainerID && item.StartsOn <= day && day <= item.EndsOn {
			return true
		}
	}
	return false
}

func GenerateWeek(now, monday time.Time, templates []TemplateSlot, existing []ClassSnapshot, absences []Absence, ids func() string, exclusive bool) WeekResult {
	var result WeekResult
	occupied := append([]ClassSnapshot{}, existing...)
	for _, template := range templates {
		if !template.Active || !template.TrainerActive {
			continue
		}
		start, err := SlotStart(monday, template.Weekday, template.StartTime)
		if err != nil || !start.After(now) {
			continue
		}
		if OnLeave(absences, template.TrainerID, start) {
			continue
		}
		end := start.Add(time.Duration(template.DurationMinutes) * time.Minute)
		if found, ok := snapshot(occupied, template.TrainerID, start); ok {
			if found.Class.Status != Scheduled || !groupKind(found.Class.Kind) {
				continue
			}
			next := applyTemplate(found, template, start)
			if Blocks(exclusive, next.StartsAt, next.EndsAt, occupied, next.ID) {
				continue
			}
			result.Updated++
			result.Upserts = append(result.Upserts, next)
			replaceSnapshot(&occupied, next)
			continue
		}
		if Blocks(exclusive, start, end, occupied, "") {
			continue
		}
		class := newClass(ids(), template, start)
		result.Created++
		result.Upserts = append(result.Upserts, class)
		occupied = append(occupied, ClassSnapshot{Class: class})
	}
	return result
}

func groupKind(kind string) bool {
	return kind == "" || kind == KindGroup
}

func replaceSnapshot(occupied *[]ClassSnapshot, class Class) {
	for i := range *occupied {
		if (*occupied)[i].Class.ID == class.ID {
			(*occupied)[i].Class = class
			return
		}
	}
}

func snapshot(existing []ClassSnapshot, trainerID string, start time.Time) (ClassSnapshot, bool) {
	for _, item := range existing {
		if item.Class.TrainerID == trainerID && item.Class.StartsAt.Equal(start) {
			return item, true
		}
	}
	return ClassSnapshot{}, false
}

func applyTemplate(found ClassSnapshot, template TemplateSlot, start time.Time) Class {
	class := found.Class
	class.TemplateID = template.ID
	class.TypeID = template.TypeID
	class.Title = template.Title
	class.EndsAt = start.Add(time.Duration(template.DurationMinutes) * time.Minute)
	class.Capacity = template.Capacity
	if found.Booked > class.Capacity {
		class.Capacity = found.Booked
	}
	class.PriceMin = template.PriceMin
	class.PriceMax = template.PriceMax
	class.Status = Scheduled
	if class.Kind == "" {
		class.Kind = KindGroup
	}
	return class
}

func newClass(id string, template TemplateSlot, start time.Time) Class {
	return Class{
		ID:         id,
		TemplateID: template.ID,
		TrainerID:  template.TrainerID,
		TypeID:     template.TypeID,
		Title:      template.Title,
		StartsAt:   start,
		EndsAt:     start.Add(time.Duration(template.DurationMinutes) * time.Minute),
		Capacity:   template.Capacity,
		PriceMin:   template.PriceMin,
		PriceMax:   template.PriceMax,
		Status:     Scheduled,
		Kind:       KindGroup,
	}
}
