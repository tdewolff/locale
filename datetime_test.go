package locale

import (
	"testing"
	"time"

	"github.com/tdewolff/test"
)

func TestScanTime(t *testing.T) {
	tests := []struct {
		s string
		t time.Time
	}{
		{"2012-10-20", time.Date(2012, 10, 20, 0, 0, 0, 0, time.UTC)},
		{"2012-10-20 12:30", time.Date(2012, 10, 20, 12, 30, 0, 0, time.UTC)},
		{"2012-10-20T12:30", time.Date(2012, 10, 20, 12, 30, 0, 0, time.UTC)},
		{"2012-10-20 12:30:05", time.Date(2012, 10, 20, 12, 30, 5, 0, time.UTC)},
		{"2012-10-20 12:30:05.1", time.Date(2012, 10, 20, 12, 30, 5, 1e8, time.UTC)},
		{"2012-10-20 12:30:05.001", time.Date(2012, 10, 20, 12, 30, 5, 1e6, time.UTC)},
		{"12:30", time.Date(1, 1, 1, 12, 30, 0, 0, time.UTC)},
		{"12:30:05", time.Date(1, 1, 1, 12, 30, 5, 0, time.UTC)},
		{"12:30:05.1", time.Date(1, 1, 1, 12, 30, 5, 1e8, time.UTC)},
		{"5000.1", time.Unix(5000, 1e8)},
	}
	for _, tt := range tests {
		t.Run(tt.s, func(t *testing.T) {
			var dst time.Time
			err := scanTime(&dst, tt.s)
			test.Error(t, err)
			test.T(t, dst, tt.t)
		})
	}
}

func TestTimeFormatter(t *testing.T) {
	tests := []struct {
		p      *Printer
		layout string
		t      time.Time
		str    string
	}{
		{en, "Monday, January 2, 2006 15:04:05 Mountain Standard Time", time.Date(2025, 1, 2, 12, 30, 0, 0, tzPST), "Thursday, January 2, 2025 at 12:30:00\u202FPM Pacific Standard Time"},
		{en, "January 2, 2006 15:04:05 MST", time.Date(2025, 1, 2, 12, 30, 0, 0, tzPST), "January 2, 2025 at 12:30:00\u202FPM PST"},
		{en, "Jan. 2, 2006 15:04:05", time.Date(2025, 1, 2, 12, 30, 0, 0, tzPST), "Jan 2, 2025, 12:30:00\u202FPM"},
		{en, "1/2/06 15:04", time.Date(2025, 1, 2, 12, 30, 0, 0, tzPST), "1/2/25, 12:30\u202FPM"},

		{es, "Monday, January 2, 2006 15:04:05 Mountain Standard Time", time.Date(2025, 1, 2, 12, 30, 0, 0, tzCET), "jueves, 2 de enero de 2025, 12:30:00 (hora estándar de Europa central)"},
		{es, "January 2, 2006 15:04:05 MST", time.Date(2025, 1, 2, 12, 30, 0, 0, tzCET), "2 de enero de 2025, 12:30:00 CET"},
		{es, "Jan. 2, 2006 15:04:05", time.Date(2025, 1, 2, 12, 30, 0, 0, tzCET), "2 ene 2025, 12:30:00"},
		{es, "1/2/06 15:04", time.Date(2025, 1, 2, 12, 30, 0, 0, tzCET), "2/1/25, 12:30"},
	}
	for _, tt := range tests {
		t.Run(tt.str, func(t *testing.T) {
			test.T(t, tt.p.T(tt.t, tt.layout), tt.str)
		})
	}
}

func TestIntervalFormatter(t *testing.T) {
	tests := []struct {
		p          *Printer
		layout     string
		start, end time.Time
		str        string
	}{
		{en, "Monday, January 2, 2006 15:04:05 Mountain Standard Time", time.Date(2025, 1, 2, 12, 30, 0, 0, tzPST), time.Date(2025, 1, 2, 12, 34, 0, 0, tzPST), "Thursday, January 2, 2025 at 12:30:00\u202FPM Pacific Standard Time\u2009–\u200912:34:00\u202FPM Pacific Standard Time"},
		{en, "Monday, January 2, 2006 15:04 Mountain Standard Time", time.Date(2025, 1, 2, 12, 30, 0, 0, tzPST), time.Date(2025, 1, 2, 12, 34, 0, 0, tzPST), "Thursday, January 2, 2025 at 12:30\u2009–\u200912:34\u202FPM Pacific Standard Time"},
		{en, "January 2, 2006 15:04:05 MST", time.Date(2025, 1, 2, 12, 30, 0, 0, tzPST), time.Date(2025, 1, 2, 12, 34, 0, 0, tzPST), "January 2, 2025 at 12:30:00\u202FPM PST\u2009–\u200912:34:00\u202FPM PST"},
		{en, "January 2, 2006 15:04 MST", time.Date(2025, 1, 2, 12, 30, 0, 0, tzPST), time.Date(2025, 1, 2, 12, 34, 0, 0, tzPST), "January 2, 2025 at 12:30\u2009–\u200912:34\u202FPM PST"},
		{en, "Jan. 2, 2006 15:04:05", time.Date(2025, 1, 2, 12, 30, 0, 0, tzPST), time.Date(2025, 1, 2, 12, 34, 0, 0, tzPST), "Jan 2, 2025, 12:30:00\u202FPM\u2009–\u200912:34:00\u202FPM"},
		{en, "Jan. 2, 2006 15:04", time.Date(2025, 1, 2, 12, 30, 0, 0, tzPST), time.Date(2025, 1, 2, 12, 34, 0, 0, tzPST), "Jan 2, 2025, 12:30\u2009–\u200912:34\u202FPM"},
		{en, "Jan. 2, 2006, 15:04:05", time.Date(2025, 1, 2, 12, 30, 0, 0, tzPST), time.Date(2025, 1, 3, 12, 30, 0, 0, tzPST), "Jan 2, 2025, 12:30:00\u202FPM\u2009–\u2009Jan 3, 2025, 12:30:00\u202FPM"},
		{en, "1/2/06 15:04", time.Date(2025, 1, 2, 12, 30, 0, 0, tzPST), time.Date(2025, 1, 2, 12, 34, 0, 0, tzPST), "1/2/25, 12:30\u2009–\u200912:34\u202FPM"},
		{en, "1/2/06 15:04", time.Date(2025, 1, 2, 11, 30, 0, 0, tzPST), time.Date(2025, 1, 2, 12, 34, 0, 0, tzPST), "1/2/25, 11:30\u202FAM\u2009–\u200912:34\u202FPM"},

		{es, "Jan. 2, 2006 15:04:05", time.Date(2025, 1, 2, 12, 30, 0, 0, tzCET), time.Date(2025, 1, 2, 12, 34, 0, 0, tzCET), "2 ene 2025, 12:30:00\u2009–\u200912:34:00"},
	}
	for _, tt := range tests {
		t.Run(tt.str, func(t *testing.T) {
			test.T(t, tt.p.T(tt.start, tt.end, tt.layout), tt.str)
		})
	}
}
