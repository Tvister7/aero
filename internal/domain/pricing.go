package domain

import (
	"bytes"
	"encoding/json"
	"strings"
	"unicode/utf8"
)

const pricingLimit = 8000

var pricingSections = []string{"base", "aero", "kids", "personal"}

// NormalizeGuide обрезает края и не даёт памятке разрастись.
func NormalizeGuide(raw string) (string, *Error) {
	guide := strings.TrimSpace(raw)
	if utf8.RuneCountInString(guide) > 4000 {
		return "", FailMessage(Validation, "Памятка слишком длинная")
	}
	return guide, nil
}

// NormalizePricing проверяет сырой JSON прайса и возвращает каноническую запись.
// Пустая строка сохраняется как пустой объект. Чужие разделы и нецелые цены отклоняются.
func NormalizePricing(raw string) (string, *Error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		raw = "{}"
	}
	if len(raw) > pricingLimit {
		return "", FailMessage(Validation, "Прайс слишком длинный")
	}
	decoder := json.NewDecoder(strings.NewReader(raw))
	decoder.UseNumber()
	var sections map[string]json.RawMessage
	if err := decoder.Decode(&sections); err != nil || decoder.More() {
		return "", FailMessage(Validation, "Прайс должен быть JSON-объектом")
	}
	if sections == nil {
		return "", FailMessage(Validation, "Прайс должен быть JSON-объектом")
	}
	allowed := map[string]bool{}
	for _, name := range pricingSections {
		allowed[name] = true
	}
	clean := map[string]map[string]int64{}
	for name, value := range sections {
		if !allowed[name] {
			return "", FailMessage(Validation, "В прайсе можно указать только base, aero, kids и personal")
		}
		offers, err := pricingOffers(value)
		if err != nil {
			return "", err
		}
		clean[name] = offers
	}
	encoded, err := json.Marshal(clean)
	if err != nil {
		return "", FailMessage(Validation, "Прайс должен быть JSON-объектом")
	}
	return string(encoded), nil
}

func pricingOffers(raw json.RawMessage) (map[string]int64, *Error) {
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 || trimmed[0] != '{' {
		return nil, FailMessage(Validation, "Раздел прайса должен быть объектом")
	}
	decoder := json.NewDecoder(bytes.NewReader(trimmed))
	decoder.UseNumber()
	var offers map[string]json.Number
	if err := decoder.Decode(&offers); err != nil || decoder.More() {
		return nil, FailMessage(Validation, "В прайсе нужны целые положительные числа")
	}
	if offers == nil {
		return map[string]int64{}, nil
	}
	clean := map[string]int64{}
	for label, number := range offers {
		if strings.TrimSpace(label) == "" || label != strings.TrimSpace(label) {
			return nil, FailMessage(Validation, "В прайсе нужны целые положительные числа")
		}
		price, err := number.Int64()
		if err != nil || price <= 0 {
			return nil, FailMessage(Validation, "В прайсе нужны целые положительные числа")
		}
		clean[label] = price
	}
	return clean, nil
}
