package console

import (
	"fmt"
)

type ANSIMod string

var ResetMod = ToANSICode(Reset)

const (
	Reset = iota
	Bold
	Faint
	Italic
	Underline
	CrossedOut = 9
)

const (
	Black = iota + 30
	Red
	Green
	Yellow
	Blue
	Magenta
	Cyan
	Gray
)

const (
	BrightBlack = iota + 90
	BrightRed
	BrightGreen
	BrightYellow
	BrightBlue
	BrightMagenta
	BrightCyan
	White
)

func (c ANSIMod) String() string {
	return string(c)
}

func ToANSICode(modes ...int) ANSIMod {
	if len(modes) == 0 {
		return ""
	}

	var s string
	for i, m := range modes {
		if i > 0 {
			s += ";"
		}
		s += fmt.Sprintf("%d", m)
	}
	return ANSIMod("\x1b[" + s + "m")
}

type Theme struct {
	Name           string
	Timestamp      ANSIMod
	Header         ANSIMod
	Source         ANSIMod
	Message        ANSIMod
	MessageDebug   ANSIMod
	AttrKey        ANSIMod
	AttrValue      ANSIMod
	AttrValueError ANSIMod
	LevelError     ANSIMod
	LevelWarn      ANSIMod
	LevelInfo      ANSIMod
	LevelDebug     ANSIMod
}

func NewDefaultTheme() Theme {
	return Theme{
		Name:           "Default",
		Timestamp:      ToANSICode(Faint),
		Header:         ToANSICode(Faint, Bold),
		Source:         ToANSICode(BrightBlack, Italic),
		Message:        ToANSICode(Bold),
		MessageDebug:   ToANSICode(Bold),
		AttrKey:        ToANSICode(Faint, Green),
		AttrValue:      ToANSICode(),
		AttrValueError: ToANSICode(Bold, Red),
		LevelError:     ToANSICode(Red),
		LevelWarn:      ToANSICode(Yellow),
		LevelInfo:      ToANSICode(Cyan),
		LevelDebug:     ToANSICode(BrightMagenta),
	}
}

func NewBrightTheme() Theme {
	return Theme{
		Name:           "Bright",
		Timestamp:      ToANSICode(Gray),
		Header:         ToANSICode(Bold, Gray),
		Source:         ToANSICode(Gray, Bold, Italic),
		Message:        ToANSICode(Bold, White),
		MessageDebug:   ToANSICode(),
		AttrKey:        ToANSICode(BrightCyan),
		AttrValue:      ToANSICode(),
		AttrValueError: ToANSICode(Bold, BrightRed),
		LevelError:     ToANSICode(BrightRed),
		LevelWarn:      ToANSICode(BrightYellow),
		LevelInfo:      ToANSICode(BrightGreen),
		LevelDebug:     ToANSICode(),
	}
}
