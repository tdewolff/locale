package locale

import (
	"testing"
	"time"

	"github.com/tdewolff/test"
)

func TestDurationFormatter(t *testing.T) {
	tests := []struct {
		p      *Printer
		layout string
		d      time.Duration
		str    string
	}{
		{en, "second", 5*time.Hour + 2*time.Minute, "5 hours 2 minutes"},
		{en, "sec", 5*time.Hour + 2*time.Minute, "5 hr 2 min"},
		{en, "s", 5*time.Hour + 2*time.Minute, "5h 2m"},
		{es, "second", 5*time.Hour + 2*time.Minute, "5 horas 2 minutos"},
		{es, "sec", 5*time.Hour + 2*time.Minute, "5 h 2 min"},
		{es, "s", 5*time.Hour + 2*time.Minute, "5h 2min"},

		{en, "second", 36 * time.Hour, "36 hours"},
		{en, "≈second", 5*time.Hour + 30*time.Minute, "6 hours"},
		{en, "≈sec", 5*time.Hour + 30*time.Minute, "6 hr"},
		{en, "≈s", 5*time.Hour + 30*time.Minute, "6h"},
	}
	for _, tt := range tests {
		t.Run(tt.str, func(t *testing.T) {
			test.T(t, tt.p.T(tt.d, tt.layout), tt.str)
		})
	}
}

func TestDurationIntervalFormatter(t *testing.T) {
	tests := []struct {
		p      *Printer
		layout string
		t      time.Time
		d      time.Duration
		str    string
	}{
		{en, "second", time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC), 36 * time.Hour, "1 day 12 hours"},
		{en, "second", time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC), 31 * 24 * time.Hour, "1 month"},
		{en, "second", time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC), 31*24*time.Hour - time.Second, "4 weeks 2 days 23 hours 59 minutes 59 seconds"},
		{en, "≈second", time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC), 36 * time.Hour, "2 days"},
		{en, "≈second", time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC), 31*24*time.Hour - time.Second, "1 month"},

		{es, "second", time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC), 36 * time.Hour, "1 día 12 horas"},
		{es, "second", time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC), 31 * 24 * time.Hour, "1 mes"},
		{es, "second", time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC), 31*24*time.Hour - time.Second, "4 semanas 2 días 23 horas 59 minutos 59 segundos"},
	}
	for _, tt := range tests {
		t.Run(tt.str, func(t *testing.T) {
			test.T(t, tt.p.T(tt.t, tt.d, tt.layout), tt.str)
		})
	}
}
