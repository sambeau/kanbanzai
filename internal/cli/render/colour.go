package render

import "fmt"

// ANSI escape codes for terminal colours.
const (
	ansiGreen  = "\033[32m"
	ansiYellow = "\033[33m"
	ansiRed    = "\033[31m"
	ansiReset  = "\033[0m"
)

// Green wraps s in ANSI green codes when tty is true.
// When tty is false, s is returned unchanged.
func Green(s string, tty bool) string {
	if !tty {
		return s
	}
	return fmt.Sprintf("%s%s%s", ansiGreen, s, ansiReset)
}

// Yellow wraps s in ANSI yellow codes when tty is true.
// When tty is false, s is returned unchanged.
func Yellow(s string, tty bool) string {
	if !tty {
		return s
	}
	return fmt.Sprintf("%s%s%s", ansiYellow, s, ansiReset)
}

// Red wraps s in ANSI red codes when tty is true.
// When tty is false, s is returned unchanged.
func Red(s string, tty bool) string {
	if !tty {
		return s
	}
	return fmt.Sprintf("%s%s%s", ansiRed, s, ansiReset)
}

// Default returns s unchanged — it represents the default/neutral colour.
func Default(s string) string {
	return s
}
