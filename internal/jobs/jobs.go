package jobs

import (
	"context"
	"log/slog"
	"time"

	"github.com/Tvister7/aero/internal/domain"
	"github.com/Tvister7/aero/internal/sqlite"
	"github.com/Tvister7/aero/internal/telegram"
)

type Scheduler struct {
	lastWeek   string
	lastBackup string
}

func (s *Scheduler) Tick(ctx context.Context, db *sqlite.DB, bot telegram.Sender, now time.Time) {
	if err := remind(ctx, db, bot, now); err != nil {
		slog.Error("reminders", "err", err)
	}
	if err := balanceWarn(ctx, db, bot, now); err != nil {
		slog.Error("balance notices", "err", err)
	}
	if domain.IsMonday0010(now) && s.lastWeek != domain.DateKey(now) {
		if _, err := db.GenerateWeek(ctx, now, ""); err != nil {
			slog.Error("week", "err", err)
		} else {
			s.lastWeek = domain.DateKey(now)
		}
	}
	if domain.IsBackupTime(now) && s.lastBackup != domain.DateKey(now) {
		if err := db.Backup(ctx); err != nil {
			slog.Error("backup", "err", err)
		} else {
			s.lastBackup = domain.DateKey(now)
		}
	}
}

func remind(ctx context.Context, db *sqlite.DB, bot telegram.Sender, now time.Time) error {
	rows, err := db.PendingReminders(ctx)
	if err != nil {
		return err
	}
	items := make([]domain.ReminderCandidate, 0, len(rows))
	byID := make(map[string]sqlite.DueReminder, len(rows))
	for _, row := range rows {
		items = append(items, row.ReminderCandidate)
		byID[row.BookingID] = row
	}
	decision := domain.Reminders(now, items)
	for _, id := range decision.Skip {
		if err := db.SetReminderState(ctx, now, id, domain.ReminderSkipped); err != nil {
			return err
		}
	}
	for _, item := range decision.Send {
		row := byID[item.BookingID]
		if bot == nil {
			continue
		}
		sendCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		err := bot.Send(sendCtx, row.TelegramID, domain.ReminderText(item.Title, item.StartsAt))
		cancel()
		if err != nil {
			slog.Error("reminder", "booking_id", item.BookingID, "err", err)
			continue
		}
		if err := db.SetReminderState(ctx, now, item.BookingID, domain.ReminderSent); err != nil {
			return err
		}
	}
	return nil
}

func balanceWarn(ctx context.Context, db *sqlite.DB, bot telegram.Sender, now time.Time) error {
	rows, err := db.PendingBalanceNotices(ctx, now)
	if err != nil {
		return err
	}
	items := make([]domain.BalanceCandidate, 0, len(rows))
	byID := make(map[string]sqlite.DueBalanceNotice, len(rows))
	for _, row := range rows {
		items = append(items, row.BalanceCandidate)
		byID[row.ID] = row
	}
	decision := domain.BalanceNotices(now, items)
	for _, id := range decision.Skip {
		if err := db.SetBalanceNoticeState(ctx, id, domain.ReminderSkipped); err != nil {
			return err
		}
	}
	for _, item := range decision.Send {
		row := byID[item.ID]
		if bot == nil {
			continue
		}
		sendCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		err := bot.Send(sendCtx, row.TelegramID, domain.EmptyBalanceText(item.Kind == domain.NoticeBookedEmpty, item.Title, item.StartsAt))
		cancel()
		if err != nil {
			slog.Error("balance notice", "id", item.ID, "err", err)
			continue
		}
		if err := db.SetBalanceNoticeState(ctx, item.ID, domain.ReminderSent); err != nil {
			return err
		}
	}
	return nil
}
