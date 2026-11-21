package domain

import "strings"

const MessageStart = "start"

// MessageTemplate текст, который можно заменить, не меняя код.
// Сейчас в каталоге только приветствие /start. Следующие уведомления
// добавляются новой строкой каталога и чтением Body в месте отправки.
type MessageTemplate struct {
	Key   string
	Title string
	Body  string
}

func MessageCatalog() []MessageTemplate {
	return []MessageTemplate{{
		Key:   MessageStart,
		Title: "Приветствие /start",
		Body:  StartText(true),
	}}
}

func MessageByKey(key string) (MessageTemplate, bool) {
	for _, item := range MessageCatalog() {
		if item.Key == key {
			return item, true
		}
	}
	return MessageTemplate{}, false
}

func NormalizeMessageBody(key, body string) (string, *Error) {
	if _, ok := MessageByKey(key); !ok {
		return "", FailMessage(Validation, "Такого сообщения нет")
	}
	body = trimMessage(body)
	if body == "" {
		return "", FailMessage(Validation, "Текст сообщения пустой")
	}
	if len([]rune(body)) > 4000 {
		return "", FailMessage(Validation, "Текст длиннее 4000 знаков")
	}
	return body, nil
}

func trimMessage(value string) string {
	return strings.TrimSpace(value)
}
