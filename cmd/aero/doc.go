// Command aero — один процесс Mini App.
//
// Без аргументов поднимает HTTP, открывает SQLite по DATA_PATH и в том же процессе
// раз в минуту рассылает напоминания, в понедельник в 00:10 по Тбилиси собирает
// следующую неделю и в 03:00 снимает копию базы. Команда seed переносит тренеров,
// типы и шаблоны из seed_data.sql.
//
// Обязательны BOT_TOKEN и DATA_PATH. ADMIN_IDS — telegram id через запятую.
// STUDIO_EXCLUSIVE по умолчанию включён: зал один, и новые интервалы не пересекаются.
// Значения 0, false, no и off выключают проверку.
// Несколько залов — будущая работа. Флаг для неё заготовка, комнаты сейчас не моделируются.
// ADDR по умолчанию 127.0.0.1:8080. В compose процесс слушает 8181, туннель Cloudflare ходит на http://aero:8181.
// CORS_ORIGIN — origin Mini App, целиком и с https.
// Токен бота и сырой initData в логи не пишутся.
//
//go:generate go tool swag init -g cmd/aero/main.go -o ../../internal/httpapi/docs --parseInternal --parseDependency --useStructName --dir ../.. --exclude internal/httpapi/docs
package main
