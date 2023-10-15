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
