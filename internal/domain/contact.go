package domain

import (
	"strings"
	"unicode"
)

const nameMessage = "Имя может содержать буквы, пробел, дефис и апостроф"
const phoneCharsMessage = "Телефон может содержать цифры, пробел, +, скобки и дефис"
const phoneDigitsMessage = "В телефоне должно быть не меньше 9 цифр"

// ValidateName пустое имя пропускает. Иначе нужны буквы и только пробел, дефис или апостроф рядом с ними.
func ValidateName(name string) *Error {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil
	}
	letters := false
	for _, r := range name {
		if unicode.IsLetter(r) {
			letters = true
			continue
		}
		if r == ' ' || r == '-' || r == '\'' || r == '’' {
			continue
		}
		return FailMessage(Validation, nameMessage)
	}
	if !letters {
		return FailMessage(Validation, nameMessage)
	}
	return nil
}

// ValidatePhone пустой телефон пропускает. Иначе допустимы цифры, пробел, плюс, скобки и дефис, цифр не меньше 9.
func ValidatePhone(phone string) *Error {
	phone = strings.TrimSpace(phone)
	if phone == "" {
		return nil
	}
	digits := 0
	for _, r := range phone {
		switch r {
		case ' ', '+', '-', '(', ')':
		default:
			if r < '0' || r > '9' {
				return FailMessage(Validation, phoneCharsMessage)
			}
			digits++
		}
	}
	if digits < 9 {
		return FailMessage(Validation, phoneDigitsMessage)
	}
	return nil
}
