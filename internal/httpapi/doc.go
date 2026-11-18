// Package httpapi публикует контракт Mini App.
// Проверяет Authorization: tma <initData>, пускает под /admin/ только id из ADMIN_IDS
// и переводит *domain.Error в статус и русский message.
// Страница /swagger/index.html собрана из комментариев моделей и этих методов
// и initData не требует. Остальные пути требуют.
package httpapi
