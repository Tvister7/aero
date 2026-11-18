// Command aero — один процесс Mini App.
//
// Без аргументов поднимает HTTP, открывает SQLite по DATA_PATH и в том же процессе
// раз в минуту рассылает напоминания, в понедельник в 00:10 по Тбилиси собирает
// следующую неделю и в 03:00 снимает копию базы. Команда seed переносит тренеров,
// типы и шаблоны из seed_data.sql.
//
// Обязательны BOT_TOKEN и DATA_PATH. ADMIN_IDS — telegram id через запятую.
// ADDR по умолчанию 127.0.0.1:8080. CORS_ORIGIN — origin статики.
// Токен бота и сырой initData в логи не пишутся.
//
//go:generate go tool swag init -g cmd/aero/main.go -o ../../internal/httpapi/docs --parseInternal --parseDependency --useStructName --dir ../.. --exclude internal/httpapi/docs
package main
