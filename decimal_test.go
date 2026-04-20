package locale

import (
	"fmt"
	"testing"

	"github.com/tdewolff/test"
)

func TestDecimalFormatter(t *testing.T) {
	tests := []struct {
		p   *Printer
		fmt string
		f   float64
		s   string
	}{
		{en, "%v", 1234.5678901, "1234.5678901"},
		{en, "%f", 1234.5678901, "1234.567890"},
		{en, "%.3f", 1234.5678901, "1234.568"},
		{en, "%g", 1234.5678900, "1234.56789"},
		{en, "%e", 1234.5678900, "1.234568\u00A0×\u00A010³"},

		{es, "%v", 1234.5678901, "1234,5678901"},
		{es, "%f", 1234.5678901, "1234,567890"},
		{es, "%.3f", 1234.5678901, "1234,568"},
		{es, "%g", 1234.5678900, "1234,56789"},
		{es, "%e", 1234.5678900, "1,234568\u00A0×\u00A010³"},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprint(tt.p.LanguageTag, "_", tt.s), func(t *testing.T) {
			test.T(t, tt.p.T(tt.fmt, tt.f), tt.s)
		})
	}
}
