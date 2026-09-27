package locale

import (
	"fmt"
	"testing"

	"golang.org/x/text/currency"

	"github.com/tdewolff/test"
)

var EUR = currency.EUR
var CLP = currency.MustParseISO("CLP")

func TestParseMoney(t *testing.T) {
	var tests = []struct {
		s string
		r string
	}{
		{"EUR16", "EUR16.00000"},
		{"EUR 16.5", "EUR16.50000"},
		{"EUR16.50", "EUR16.50000"},
		{"EUR16.51234", "EUR16.51234"},
		{"EUR16.512344", "EUR16.51234"},
		{"EUR16.512346", "EUR16.51235"},
		{"EUR16.512345", "EUR16.51234"},
		{"EUR16.512355", "EUR16.51236"},
	}

	for _, tt := range tests {
		t.Run(tt.r, func(t *testing.T) {
			money, err := ParseMoney(tt.s)
			test.Error(t, err)
			test.T(t, money.String(), tt.r)
		})
	}
}

func TestMoneyFromFloat64(t *testing.T) {
	var tests = []struct {
		cur currency.Unit
		f   float64
		r   string
	}{
		{EUR, 16, "EUR16.00000"},
		{EUR, 16.5, "EUR16.50000"},
		{EUR, 16.50, "EUR16.50000"},
		{EUR, 16.51234, "EUR16.51234"},
		{EUR, 16.512344, "EUR16.51234"},
		{EUR, 16.512346, "EUR16.51235"},
		{EUR, 16.512345, "EUR16.51234"},
		{EUR, 16.512355, "EUR16.51236"},
	}

	for _, tt := range tests {
		t.Run(tt.r, func(t *testing.T) {
			money, err := MoneyFromFloat64(tt.cur, tt.f)
			test.Error(t, err)
			test.T(t, money.String(), tt.r)
		})
	}
}

func TestMoneyRounded(t *testing.T) {
	var tests = []struct {
		a      string
		digits int
		incr   int
		r      string
	}{
		{"16", 2, 0, "16.00"},
		{"16.5", 2, 0, "16.50"},
		{"16.50", 2, 0, "16.50"},
		{"16.505", 2, 0, "16.50"},
		{"16.506", 2, 0, "16.51"},
		{"16.514", 2, 0, "16.51"},
		{"16.515", 2, 0, "16.52"},
		{"16.515", 3, 10, "16.520"},
		{"16.515", 4, 100, "16.5200"},

		{"16.515", 0, 5, "15"},
		{"16.515", 1, 5, "16.5"},
		{"16.515", 2, 50, "16.50"},
	}

	for _, tt := range tests {
		t.Run(tt.a, func(t *testing.T) {
			amount, err := ParseAmount(tt.a, 5)
			test.Error(t, err)
			money := Money{EUR, amount, tt.digits, tt.incr}
			test.T(t, money.RoundedString()[3:], tt.r)
		})
	}
}

func TestMoneyOperation(t *testing.T) {
	test.T(t, MustMakeMoney(EUR, 1000, 3).MustMuli(2).MustDivi(3), MustMakeMoney(EUR, 66667, 5))
	test.T(t, MustMakeMoney(CLP, -7933335, 3).MustAdd(MustMakeMoney(CLP, 3767540, 3)).RoundedString(), "CLP-4,166")
}

func TestMoneyScanValue(t *testing.T) {
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
			var money Money
			err := money.Scan(tt.s)
			test.Error(t, err)
			v, _ := money.Value()
			test.T(t, v, tt.r)
		})
	}
}

func TestMoneyFormat(t *testing.T) {
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
		t.Run(tt.f+": "+tt.s, func(t *testing.T) {
			var money Money
			err := money.Scan(tt.s)
			test.Error(t, err)

			v := fmt.Sprintf("%v", MoneyFormatter{money, tt.f})
			test.T(t, v, tt.r)
		})
	}
}
