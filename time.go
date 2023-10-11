package locale

import (
	"fmt"
	"time"

	"golang.org/x/text/message"
)

type TimeFormatter struct {
	time.Time
	layout string
}

func (f TimeFormatter) Format(state fmt.State, verb rune) {
	var b []byte
	if languager, ok := state.(Languager); ok {
		p := message.NewPrinter(languager.Language())
		layout := p.Sprintf(f.layout)
		b = []byte(f.Time.Format(layout))
		for j := 2; j < len(layout); j++ {
			// TODO: advance pos in b when layout is 1 or 2 without 0 prefix and b has two numbers
			i := len(layout) - j
			n := 0
			if layout[i:i+2] == "am" || layout[i:i+2] == "AM" || layout[i:i+2] == "pm" || layout[i:i+2] == "PM" {
				n = 2
			} else if i+3 <= len(layout) && (layout[i:i+3] == "Jan" || layout[i:i+3] == "Mon") {
				n = 3
			} else if i+6 <= len(layout) && layout[i:i+6] == "Monday" {
				n = 6
			} else if i+7 <= len(layout) && layout[i:i+7] == "January" {
				n = 7
			}

			if n != 0 {
				i := len(b) - j
				replacement := []byte(p.Sprintf(string(b[i : i+n])))
				b = append(b[:i], append(replacement, b[i+n:]...)...)
			}
		}

	} else {
		b = []byte(f.Time.Format(f.layout))
	}
	state.Write(b)
}
