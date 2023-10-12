package locale

import (
	"testing"

	"github.com/tdewolff/test"
	"golang.org/x/text/currency"
)

func TestCurrency(t *testing.T) {
	var tests = []struct {
		cur currency.Unit
		a   string
		r   string
	}{
		{currency.EUR, "16", "EUR16.00"},
		{currency.EUR, "16.5", "EUR16.50"},
		{currency.EUR, "16.50", "EUR16.50"},
		{currency.EUR, "16.505", "EUR16.50"},
		{currency.EUR, "16.506", "EUR16.51"},
		{currency.EUR, "16.514", "EUR16.51"},
		{currency.EUR, "16.515", "EUR16.52"},
	}

	for _, tt := range tests {
		t.Run(tt.a, func(t *testing.T) {
			amount, err := ParseAmount(tt.cur, tt.a)
			test.Error(t, err)
			test.T(t, amount.String(), tt.r)
		})
	}
}

func TestAmountRound(t *testing.T) {
	tests := []struct {
		a Amount
		r Amount
	}{
		{},
	}
	for _, tt := range tests {
		t.Run(tt.a.String(), func(t *testing.T) {
			a := tt.a
			a.Round()
			test.T(t, a, tt.r)
		})
	}
}
