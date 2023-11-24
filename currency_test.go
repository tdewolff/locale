package locale

import (
	"fmt"
	"testing"

	"github.com/tdewolff/test"
	"golang.org/x/text/currency"
)

var EUR = currency.EUR

func TestAmount(t *testing.T) {
	var tests = []struct {
		cur currency.Unit
		a   string
		r   string
	}{
		{EUR, "16", "EUR16.00"},
		{EUR, "16.5", "EUR16.50"},
		{EUR, "16.50", "EUR16.50"},
		{EUR, "16.505", "EUR16.50"},
		{EUR, "16.506", "EUR16.51"},
		{EUR, "16.514", "EUR16.51"},
		{EUR, "16.515", "EUR16.52"},
	}

	for _, tt := range tests {
		t.Run(tt.a, func(t *testing.T) {
			c, err := ParseAmount(tt.cur, tt.a)
			test.Error(t, err)
			test.T(t, c.String(), tt.r)
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
		{NewAmount(EUR, 1000, 3).Mul(2).Div(3), NewAmount(EUR, 667, 3)},
	}
	for _, tt := range tests {
		t.Run(tt.a.String(), func(t *testing.T) {
			fmt.Printf("%#v %#v\n", tt.a, tt.r)
			test.T(t, tt.a, tt.r)
		})
	}
}
