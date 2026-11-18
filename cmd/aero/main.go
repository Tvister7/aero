package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/Tvister7/aero/internal/httpapi"
	"github.com/Tvister7/aero/internal/jobs"
	"github.com/Tvister7/aero/internal/sqlite"
	"github.com/Tvister7/aero/internal/telegram"
)

// Aero API — запись на занятия студии.
//
// @title Aero API
// @version 1
// @description Запись, баланс в занятиях и админка. Кроме страницы Swagger каждый запрос требует Authorization.
// @accept json
// @produce json
// @tag.name Клиент
// @tag.description Расписание, запись, профиль и контакты. Нужен initData.
// @tag.name Админ
// @tag.description Тренеры, слоты, явка и баланс. Telegram id должен быть в ADMIN_IDS.

// @securityDefinitions.apikey TMA
// @in header
// @name Authorization
// @description Значение заголовка: «tma » и initData из Telegram.WebApp.initData. Префикс tma с пробелом обязателен.
func main() {
	if err := run(os.Args[1:]); err != nil {
		slog.Error("aero", "err", err.Error())
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) > 0 && args[0] == "seed" {
		if len(args) != 2 {
			return fmt.Errorf("usage: aero seed seed_data.sql")
		}
		return seed(args[1])
	}
	if len(args) > 0 {
		return fmt.Errorf("usage: aero [seed seed_data.sql]")
	}
	return serve()
}

func serve() error {
	token := os.Getenv("BOT_TOKEN")
	path := os.Getenv("DATA_PATH")
	if token == "" || path == "" {
		return fmt.Errorf("BOT_TOKEN and DATA_PATH are required")
	}
	admins, err := parseAdmins(os.Getenv("ADMIN_IDS"))
	if err != nil {
		return err
	}
	db, err := sqlite.Open(path)
	if err != nil {
		return err
	}
	defer db.Close()
	bot := telegram.NewBot(token)
	api := &httpapi.Server{
		DB: db, Bot: bot, Admins: admins, Origin: os.Getenv("CORS_ORIGIN"), Token: token,
	}
	addr := os.Getenv("ADDR")
	if addr == "" {
		addr = "127.0.0.1:8080"
	}
	server := &http.Server{Addr: addr, Handler: api.Handler(), ReadHeaderTimeout: 5 * time.Second}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	go runJobs(ctx, db, bot)
	go func() {
		<-ctx.Done()
		shutdown, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = server.Shutdown(shutdown)
	}()
	slog.Info("listen", "addr", addr)
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

func runJobs(ctx context.Context, db *sqlite.DB, bot telegram.Sender) {
	var scheduler jobs.Scheduler
	scheduler.Tick(ctx, db, bot, time.Now())
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case now := <-ticker.C:
			scheduler.Tick(ctx, db, bot, now)
		}
	}
}

func seed(path string) error {
	data := os.Getenv("DATA_PATH")
	if data == "" {
		return fmt.Errorf("DATA_PATH is required")
	}
	db, err := sqlite.Open(data)
	if err != nil {
		return err
	}
	defer db.Close()
	counts, err := sqlite.ImportFile(context.Background(), db, path, time.Now())
	if err != nil {
		return err
	}
	fmt.Printf("trainers %d, types %d, templates %d\n", counts.Trainers, counts.Types, counts.Templates)
	return nil
}

func parseAdmins(raw string) (map[int64]struct{}, error) {
	out := map[int64]struct{}{}
	if strings.TrimSpace(raw) == "" {
		return out, nil
	}
	for _, part := range strings.Split(raw, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		id, err := strconv.ParseInt(part, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("ADMIN_IDS: %s", part)
		}
		out[id] = struct{}{}
	}
	return out, nil
}
