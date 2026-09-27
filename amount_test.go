package locale

import (
	"fmt"
	"testing"

	"github.com/tdewolff/test"
)

func TestParseAmount(t *testing.T) {
	var tests = []struct {
		a    string
		prec int
		r    Amount
	}{
		{"16", 2, Amount{1600, 2}},
		{"16.5", 2, Amount{1650, 2}},
		{"16.50", 2, Amount{1650, 2}},
		{"16.505", 2, Amount{1650, 2}},
		{"16.506", 2, Amount{1651, 2}},
		{"16.514", 2, Amount{1651, 2}},
		{"16.515", 2, Amount{1652, 2}},

		{"16", 0, Amount{16, 0}},
		{"16", -1, Amount{2, -1}},
	}

	for _, tt := range tests {
		t.Run(tt.a, func(t *testing.T) {
			amount, err := ParseAmount(tt.a, tt.prec)
			test.Error(t, err)
			test.T(t, amount, tt.r)
		})
	}
}

func TestAmountFromFloat64(t *testing.T) {
	var tests = []struct {
		f float64
		r string
	}{
		{16, "16.00000"},
		{16.5, "16.50000"},
		{16.50, "16.50000"},
		{16.51234, "16.51234"},
		{16.512344, "16.51234"},
		{16.512346, "16.51235"},
		{16.512345, "16.51234"},
		{16.512355, "16.51236"},
	}

	for _, tt := range tests {
		t.Run(tt.r, func(t *testing.T) {
			amount, err := AmountFromFloat64(tt.f, 5)
			test.Error(t, err)
			test.T(t, amount.String(), tt.r)
		})
	}
}

func TestAmountString(t *testing.T) {
	var tests = []struct {
		a   Amount
		opt bool
		r   string
	}{
		{Amount{1600, 2}, false, "16.00"},
		{Amount{1610, 2}, false, "16.10"},
		{Amount{1600, 2}, true, "16"},
		{Amount{1610, 2}, true, "16.1"},
	}

	for _, tt := range tests {
		t.Run(tt.a.String(), func(t *testing.T) {
			if tt.opt {
				test.T(t, tt.a.MinString(), tt.r)
			} else {
				test.T(t, tt.a.String(), tt.r)
			}
		})
	}
}

func TestAmountFormat(t *testing.T) {
	var tests = []struct {
		f string
		s string
		r string
	}{
		{"100", "16.00", "16"},
		{"100", "16.01", "16.01"},
		{"100", "16.001", "16.001"},
		{"100.", "16.00", "16.00"},
		{"100.0", "16.00", "16.0"},
		{"100.0", "16.06", "16.1"},
		{"100.00", "16.00", "16.00"},
		{"100.00", "16.006", "16.01"},
		{"100.9", "16.00", "16"},
		{"100.9", "16.06", "16.1"},
		{"100.99", "16.00", "16"},
		{"100.99", "16.006", "16.01"},
		{"100.99", "16.10", "16.1"},
	}

	for _, tt := range tests {
		t.Run(tt.f+": "+tt.s, func(t *testing.T) {
			var amount Amount
			err := amount.Scan(tt.s)
			test.Error(t, err)

			v := fmt.Sprintf("%v", AmountFormatter{amount, tt.f})
			test.T(t, v, tt.r)
		})
	}
}

func TestAmountScanValue(t *testing.T) {
	var tests = []struct {
		s string
		r string
	}{
		{"16.00", "16"},
		{"16.51", "16.51"},
		{"16.51234", "16.51234"},
		{"16.512344", "16.512344"},
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

func TestAmountEqual(t *testing.T) {
	tests := []struct {
		a     Amount
		b     Amount
		equal bool
	}{
		{Amount{500, 2}, Amount{500, 2}, true},
		{Amount{500, 2}, Amount{600, 2}, false},
		{Amount{500, 2}, Amount{500, 1}, false},
		{Amount{500, 2}, Amount{5000, 3}, true},
		{Amount{500, 2}, Amount{5, 0}, true},

		{Amount{AmountInvalid, 0}, Amount{AmountInvalid, 0}, false},
		{Amount{AmountInvalid, 2}, Amount{AmountInvalid, 0}, false},
	}
	for _, tt := range tests {
		t.Run(tt.a.String(), func(t *testing.T) {
			test.T(t, tt.a.Equal(tt.b), tt.equal)
		})
	}
}

func TestAmountCompare(t *testing.T) {
	tests := []struct {
		a   Amount
		b   Amount
		cmp int
	}{
		{Amount{500, 2}, Amount{500, 2}, 0},
		{Amount{500, 2}, Amount{600, 2}, -1},
		{Amount{500, 2}, Amount{500, 1}, -1},
		{Amount{500, 2}, Amount{5000, 3}, 0},
		{Amount{500, 2}, Amount{5, 0}, 0},

		{Amount{AmountInvalid, 0}, Amount{AmountInvalid, 0}, 0},
		{Amount{AmountInvalid, 2}, Amount{AmountInvalid, 0}, 0},
	}
	for _, tt := range tests {
		t.Run(tt.a.String(), func(t *testing.T) {
			test.T(t, tt.a.Compare(tt.b), tt.cmp)
		})
	}
}

func TestAmountRoundTo(t *testing.T) {
	var tests = []struct {
		a    Amount
		prec int
		r    string
	}{
		{Amount{1600, 2}, 2, "16.00"},
		{Amount{1650, 2}, 2, "16.50"},
		{Amount{1650, 2}, 2, "16.50"},
		{Amount{16505, 3}, 2, "16.500"},
		{Amount{16506, 3}, 2, "16.510"},
		{Amount{16514, 3}, 2, "16.510"},
		{Amount{16515, 3}, 2, "16.520"},
		{Amount{16514, 3}, 1, "16.500"},
		{Amount{16514, 3}, 0, "17.000"},

		{Amount{1600, 2}, 0, "16.00"},
		{Amount{1600, 2}, -1, "20.00"},
	}

	for _, tt := range tests {
		t.Run(tt.a.String(), func(t *testing.T) {
			amount := tt.a.RoundTo(tt.prec)
			test.T(t, amount.String(), tt.r)
		})
	}
}

func TestAmountAdd(t *testing.T) {
	tests := []struct {
		a Amount
		b Amount
		r Amount
	}{
		{Amount{500, 2}, Amount{250, 2}, Amount{750, 2}},
		{Amount{-7933335, 3}, Amount{3767540, 3}, Amount{-4165795, 3}},

		{Amount{AmountMax, 3}, Amount{1, 3}, InvalidAmount},
		{Amount{-AmountMax, 3}, Amount{-1, 3}, InvalidAmount},
	}
	for _, tt := range tests {
		t.Run(tt.a.String(), func(t *testing.T) {
			test.T(t, tt.a.Add(tt.b), tt.r)
		})
	}
}

func TestAmountSub(t *testing.T) {
	tests := []struct {
		a Amount
		b Amount
		r Amount
	}{
		{Amount{500, 2}, Amount{250, 2}, Amount{250, 2}},
		{Amount{-7933335, 3}, Amount{3767540, 3}, Amount{-11700875, 3}},

		{Amount{AmountMax, 3}, Amount{-1, 3}, InvalidAmount},
		{Amount{-AmountMax, 3}, Amount{1, 3}, InvalidAmount},
	}
	for _, tt := range tests {
		t.Run(tt.a.String(), func(t *testing.T) {
			test.T(t, tt.a.Sub(tt.b), tt.r)
		})
	}
}

func TestAmountMul(t *testing.T) {
	tests := []struct {
		a Amount
		b Amount
		r Amount
	}{
		{Amount{105, 3}, Amount{50000, 4}, Amount{525, 3}},
		{Amount{105, 3}, Amount{-5, 1}, Amount{-52, 3}},
		{Amount{-105, 3}, Amount{5, 0}, Amount{-525, 3}},
		{Amount{-105, 3}, Amount{-5, 2}, Amount{5, 3}},
	}
	for _, tt := range tests {
		t.Run(tt.a.String(), func(t *testing.T) {
			test.T(t, tt.a.Mul(tt.b), tt.r)
		})
	}
}

func TestAmountMuli(t *testing.T) {
	tests := []struct {
		a Amount
		b int
		r Amount
	}{
		{Amount{105, 3}, 5, Amount{525, 3}},
		{Amount{105, 3}, -5, Amount{-525, 3}},
		{Amount{-105, 3}, 5, Amount{-525, 3}},
		{Amount{-105, 3}, -5, Amount{525, 3}},
	}
	for _, tt := range tests {
		t.Run(tt.a.String(), func(t *testing.T) {
			test.T(t, tt.a.Muli(tt.b), tt.r)
		})
	}
}

func TestAmountMulf(t *testing.T) {
	tests := []struct {
		a Amount
		b float64
		r Amount
	}{
		{Amount{105, 3}, 5.0, Amount{525, 3}},
		{Amount{105, 3}, -5.0, Amount{-525, 3}},
		{Amount{-105, 3}, 5.0, Amount{-525, 3}},
		{Amount{-105, 3}, -5.0, Amount{525, 3}},
	}
	for _, tt := range tests {
		t.Run(tt.a.String(), func(t *testing.T) {
			test.T(t, tt.a.Mulf(tt.b), tt.r)
		})
	}
}

func TestAmountDiv(t *testing.T) {
	tests := []struct {
		a Amount
		b Amount
		r Amount
	}{
		{Amount{105, 3}, Amount{50000, 4}, Amount{21, 3}},
		{Amount{105, 3}, Amount{-5, 1}, Amount{-210, 3}},
		{Amount{-105, 3}, Amount{5, 0}, Amount{-21, 3}},
		{Amount{-105, 3}, Amount{-5, 2}, Amount{2100, 3}},
	}
	for _, tt := range tests {
		t.Run(tt.a.String(), func(t *testing.T) {
			test.T(t, tt.a.Div(tt.b), tt.r)
		})
	}
}

func TestAmountDivi(t *testing.T) {
	tests := []struct {
		a Amount
		b int
		r Amount
	}{
		{Amount{105, 3}, 5, Amount{21, 3}},
		{Amount{105, 3}, -5, Amount{-21, 3}},
		{Amount{-105, 3}, 5, Amount{-21, 3}},
		{Amount{-105, 3}, -5, Amount{21, 3}},
	}
	for _, tt := range tests {
		t.Run(tt.a.String(), func(t *testing.T) {
			test.T(t, tt.a.Divi(tt.b), tt.r)
		})
	}
}

func TestAmountDivf(t *testing.T) {
	tests := []struct {
		a Amount
		b float64
		r Amount
	}{
		{Amount{105, 3}, 5.0, Amount{21, 3}},
		{Amount{105, 3}, -5.0, Amount{-21, 3}},
		{Amount{-105, 3}, 5.0, Amount{-21, 3}},
		{Amount{-105, 3}, -5.0, Amount{21, 3}},
	}
	for _, tt := range tests {
		t.Run(tt.a.String(), func(t *testing.T) {
			test.T(t, tt.a.Divf(tt.b), tt.r)
		})
	}
}

func TestAmountOperations(t *testing.T) {
	tests := []struct {
		a Amount
		r Amount
	}{
		{Amount{1000, 3}.Muli(2).Divi(3), Amount{667, 3}},
		{Amount{333333, 3}.Mulf(1.19).ShiftTo(2), Amount{39667, 2}},
	}
	for _, tt := range tests {
		t.Run(tt.a.String(), func(t *testing.T) {
			test.T(t, tt.a, tt.r)
		})
	}
}
