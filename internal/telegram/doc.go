// Package telegram проверяет initData Mini App и шлёт sendMessage.
// Секрет подписи — HMAC-SHA256: ключ WebAppData, сообщение — токен бота.
// auth_date старше суток отклоняется. Токен и сырой initData в логи не попадают.
package telegram
