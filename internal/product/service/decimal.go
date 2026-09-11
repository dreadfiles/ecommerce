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

	if strings.HasPrefix(value, "-") {
		return nil, errors.New("value cannot be negative")
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

	integerPart := new(big.Int)
	integerPart.SetString(parts[0], 10)

	if len(parts) == 1 {
		return new(big.Rat).SetInt(integerPart), nil
	}

	decimalPart := new(big.Int)
	decimalPart.SetString(parts[1], 10)

	scale := len(parts[1])

	denominator := new(big.Int).Exp(
		big.NewInt(10),
		big.NewInt(int64(scale)),
		nil,
	)

	numerator := new(big.Int).Mul(
		integerPart,
		denominator,
	)

	numerator.Add(numerator, decimalPart)

	return new(big.Rat).SetFrac(
		numerator,
		denominator,
	), nil
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
