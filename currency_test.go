package locale

import (
	"fmt"
	"testing"

	"golang.org/x/text/currency"

	"github.com/tdewolff/test"
)

var EUR = currency.EUR

func TestNewAmountFromFloat64(t *testing.T) {
	var tests = []struct {
		cur currency.Unit
		f   float64
		r   string
	}{
		{EUR, 16, "EUR 16"},
		{EUR, 16.5, "EUR 16.5"},
		{EUR, 16.50, "EUR 16.5"},
		{EUR, 16.51234, "EUR 16.51234"},
		{EUR, 16.512344, "EUR 16.51234"},
		{EUR, 16.512346, "EUR 16.51235"},
		{EUR, 16.512345, "EUR 16.51234"},
		{EUR, 16.512355, "EUR 16.51236"},
	}

	for _, tt := range tests {
		t.Run(tt.r, func(t *testing.T) {
			amount := NewAmountFromFloat64(tt.cur, tt.f)
			test.T(t, amount.Unit.String()+" "+amount.StringAmount(), tt.r)
		})
	}
}

func TestAmountRounded(t *testing.T) {
	var tests = []struct {
		cur currency.Unit
		a   string
		r   string
	}{
		{EUR, "16", "EUR 16.00"},
		{EUR, "16.5", "EUR 16.50"},
		{EUR, "16.50", "EUR 16.50"},
		{EUR, "16.505", "EUR 16.50"},
		{EUR, "16.506", "EUR 16.51"},
		{EUR, "16.514", "EUR 16.51"},
		{EUR, "16.515", "EUR 16.52"},
	}

	for _, tt := range tests {
		t.Run(tt.a, func(t *testing.T) {
			amount, err := ParseAmount(tt.cur, tt.a)
			test.Error(t, err)
			test.T(t, amount.String(), tt.r)
		})
	}
}

func TestAmountOperation(t *testing.T) {
	tests := []struct {
		a Amount
		r Amount
	}{
		{NewAmount(EUR, 105, 3).Round(), NewAmount(EUR, 100, 3)},
		{NewAmount(EUR, 115, 3).Round(), NewAmount(EUR, 120, 3)},
		{NewAmount(EUR, 1000, 3).Mul(2).Div(3), NewAmount(EUR, 66667, 5)},
	}
	for _, tt := range tests {
		t.Run(tt.a.String(), func(t *testing.T) {
			test.T(t, tt.a, tt.r)
		})
	}
}

func TestAmountScanValue(t *testing.T) {
	var tests = []struct {
		s string
		r string
	}{
		{"EUR16.00", "EUR16"},
		{"EUR16.51", "EUR16.51"},
		{"EUR16.51234", "EUR16.51234"},
		{"EUR16.512344", "EUR16.51234"},
		{"EUR16.512346", "EUR16.51235"},
		{"EUR16.512345", "EUR16.51234"},
		{"EUR16.512355", "EUR16.51236"},
	}

	for _, tt := range tests {
		t.Run(tt.s, func(t *testing.T) {
			var amount Amount
			err := amount.Scan(tt.s)
			test.Error(t, err)
			v, _ := amount.Value()
			test.T(t, v, tt.r)
		})
	}
}

func TestAmountFormat(t *testing.T) {
	var tests = []struct {
		f string
		s string
		r string
	}{
		{"100", "USD16.00", "16"},
		{"US$ 100", "USD16.00", "US$\u00A016"},
		{"USD 100", "USD16.00", "USD\u00A016"},
		{"$100", "USD16.00", "$\u00A016"},
		{"100", "EUR16.00", "16"},
		{"US$ 100", "EUR16.00", "€\u00A016"},
		{"USD 100", "EUR16.00", "EUR\u00A016"},
		{"$100", "EUR16.00", "€\u00A016"},
		{"$100", "EUR16.01", "€\u00A016.01"},
		{"$100", "EUR16.001", "€\u00A016"},
		{"$100.", "EUR16.00", "€\u00A016.00"},
		{"$100.", "EUR16.01", "€\u00A016.01"},
		{"$100.", "EUR16.001", "€\u00A016.00"},
		{"$100.0", "EUR16.00", "€\u00A016.0"},
		{"$100.0", "EUR16.06", "€\u00A016.1"},
		{"$100.00", "EUR16.00", "€\u00A016.00"},
		{"$100.00", "EUR16.006", "€\u00A016.01"},
		{"$100.9", "EUR16.00", "€\u00A016"},
		{"$100.9", "EUR16.06", "€\u00A016.1"},
		{"$100.99", "EUR16.00", "€\u00A016"},
		{"$100.99", "EUR16.006", "€\u00A016.01"},
		{"$100.99", "EUR16.10", "€\u00A016.1"},
	}

	for _, tt := range tests {
		t.Run(tt.s, func(t *testing.T) {
			var amount Amount
			err := amount.Scan(tt.s)
			test.Error(t, err)

			v := fmt.Sprintf("%v", AmountFormatter{amount, tt.f})
			test.T(t, v, tt.r)
		})
	}
}
