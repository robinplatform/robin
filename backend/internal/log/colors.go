package log

import (
	"strconv"
)

var ansiTextColorReset = ansi(39)

func ansi(code int) string {
	return "\033[" + strconv.FormatInt(int64(code), 10) + "m"
}

var levelStrings = map[string]string{
	"error": ansi(31) + "ERR" + ansiTextColorReset,

	"debug": ansi(33) + "DBG" + ansiTextColorReset,
	"warn":  ansi(33) + "WRN" + ansiTextColorReset,

	"print": ansi(34) + "INF" + ansiTextColorReset,
	"info":  ansi(34) + "INF" + ansiTextColorReset,
}

func hash(str string) int {
	var hash int = 5381
	for _, c := range str {
		hash = ((hash << 5) + hash) + int(c)
	}
	return hash
}

// randColor returns a random ANSI color code
func randColor(seed string) int {
	return hash(seed)%6 + 31
}

func color(clr int, text string) string {
	return ansi(clr) + text + ansiTextColorReset
}
