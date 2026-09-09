package service

import (
	"errors"
	"math/big"
	"strings"
)

func parseDecimal(value string, maxScale int) (*big.Rat, error) {
	value = strings.TrimSpace(value)

	if value == "" {
		return nil, errors.New("value is required")
	}

	if strings.Count(value, ".") > 1 {
		return nil, errors.New("invalid decimal format")
	}

	parts := strings.Split(value, ".")

	if !isDigits(parts[0]) {
		return nil, errors.New("invalid decimal format")
	}

	if len(parts) == 2 {
		if !isDigits(parts[1]) {
			return nil, errors.New("invalid decimal format")
		}

		if len(parts[1]) > maxScale {
			return nil, errors.New("too many decimal places")
		}
	}

	rat := new(big.Rat)

	if _, ok := rat.SetString(value); !ok {
		return nil, errors.New("invalid decimal format")
	}

	if rat.Sign() < 0 {
		return nil, errors.New("value cannot be negative")
	}

	return rat, nil
}

func isDigits(value string) bool {
	if value == "" {
		return false
	}

	for _, char := range value {
		if char < '0' || char > '9' {
			return false
		}
	}

	return true
}
