package stack

import "fmt"

// ANSI colour / style codes (SPEC §19).
const (
	Magenta = "\033[95m"
	Blue    = "\033[94m"
	Green   = "\033[92m"
	Red     = "\033[91m"
	Bold    = "\033[1m"
	Reset   = "\033[0m"
)

// hyperlink wraps text in an OSC 8 terminal hyperlink sequence.
func hyperlink(url, text string) string {
	return fmt.Sprintf("\033]8;;%s\033\\%s\033]8;;\033\\", url, text)
}

// Style helpers.
func colorf(code, format string, a ...any) string { return code + fmt.Sprintf(format, a...) + Reset }
func Redf(format string, a ...any) string         { return colorf(Red, format, a...) }
func Greenf(format string, a ...any) string       { return colorf(Green, format, a...) }
func Bluef(format string, a ...any) string        { return colorf(Blue, format, a...) }
func Headerf(format string, a ...any) string      { return colorf(Magenta, format, a...) }
