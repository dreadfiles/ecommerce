package service

import (
	"testing"
)

func TestParseDecimal(t *testing.T) {
	tests := []struct {
		name      string
		value     string
		maxScale  int
		wantValid bool
		wantValue string
	}{
		{
			name:      "accepts integer",
			value:     "10",
			maxScale:  2,
			wantValid: true,
			wantValue: "10",
		},
		{
			name:      "accepts decimal",
			value:     "10.50",
			maxScale:  2,
			wantValid: true,
			wantValue: "21/2",
		},
		{
			name:      "accepts zero",
			value:     "0",
			maxScale:  2,
			wantValid: true,
			wantValue: "0",
		},
		{
			name:      "accepts maximum scale",
			value:     "10.999",
			maxScale:  3,
			wantValid: true,
			wantValue: "10999/1000",
		},
		{
			name:      "trims surrounding spaces",
			value:     " 10.50 ",
			maxScale:  2,
			wantValid: true,
			wantValue: "21/2",
		},
		{
			name:      "rejects empty value",
			value:     "",
			maxScale:  2,
			wantValid: false,
		},
		{
			name:      "rejects multiple decimal separators",
			value:     "10.5.0",
			maxScale:  2,
			wantValid: false,
		},
		{
			name:      "rejects non numeric integer part",
			value:     "abc.10",
			maxScale:  2,
			wantValid: false,
		},
		{
			name:      "rejects non numeric decimal part",
			value:     "10.ab",
			maxScale:  2,
			wantValid: false,
		},
		{
			name:      "rejects too many decimal places",
			value:     "10.123",
			maxScale:  2,
			wantValid: false,
		},
		{
			name:      "rejects negative value",
			value:     "-10",
			maxScale:  2,
			wantValid: false,
		},
		{
			name:      "rejects negative decimal",
			value:     "-10.50",
			maxScale:  2,
			wantValid: false,
		},
		{
			name:      "rejects missing integer part",
			value:     ".50",
			maxScale:  2,
			wantValid: false,
		},
		{
			name:      "rejects missing decimal part",
			value:     "10.",
			maxScale:  2,
			wantValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseDecimal(tt.value, tt.maxScale)

			if !tt.wantValid {
				if err == nil {
					t.Fatalf(
						"expected error for value %q",
						tt.value,
					)
				}

				return
			}

			if err != nil {
				t.Fatalf(
					"expected no error, got %v",
					err,
				)
			}

			if got.RatString() != tt.wantValue {
				t.Errorf(
					"expected %s, got %s",
					tt.wantValue,
					got.RatString(),
				)
			}
		})
	}
}

func TestIsDigits(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  bool
	}{
		{
			name:  "accepts digits",
			value: "12345",
			want:  true,
		},
		{
			name:  "accepts zero",
			value: "0",
			want:  true,
		},
		{
			name:  "rejects empty",
			value: "",
			want:  false,
		},
		{
			name:  "rejects letters",
			value: "123a",
			want:  false,
		},
		{
			name:  "rejects spaces",
			value: "12 3",
			want:  false,
		},
		{
			name:  "rejects sign",
			value: "+123",
			want:  false,
		},
		{
			name:  "rejects negative sign",
			value: "-123",
			want:  false,
		},
		{
			name:  "rejects decimal",
			value: "12.3",
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isDigits(tt.value); got != tt.want {
				t.Errorf(
					"isDigits(%q) = %v, want %v",
					tt.value,
					got,
					tt.want,
				)
			}
		})
	}
}
