package domain

import (
	"strconv"
	"strings"
)

func NormalizePrice(min, max string) (string, string, *Error) {
	left, okLeft := cents(min)
	right, okRight := cents(max)
	if !okLeft || !okRight || right < left {
		return "", "", Fail(Validation)
	}
	return formatCents(left), formatCents(right), nil
}

func cents(value string) (int, bool) {
	value = strings.TrimSpace(value)
	if value == "" || strings.HasPrefix(value, "-") {
		return 0, false
	}
	wholePart, fracPart, found := strings.Cut(value, ".")
	if strings.Contains(fracPart, ".") || wholePart == "" {
		return 0, false
	}
	whole, err := strconv.Atoi(wholePart)
	if err != nil {
		return 0, false
	}
	frac := 0
	if found {
		if fracPart == "" || len(fracPart) > 2 {
			return 0, false
		}
		parsed, err := strconv.Atoi(fracPart)
		if err != nil {
			return 0, false
		}
		if len(fracPart) == 1 {
			parsed *= 10
		}
		frac = parsed
	}
	return whole*100 + frac, true
}

func formatCents(value int) string {
	return strconv.Itoa(value/100) + "." + two(value%100)
}

func two(value int) string {
	if value < 10 {
		return "0" + strconv.Itoa(value)
	}
	return strconv.Itoa(value)
}
