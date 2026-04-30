package render

// Symbol returns the display symbol for the given name, choosing between
// Unicode (TTY) and ASCII (non-TTY) representations.
//
// Mapping table:
//
//	TTY  | ASCII      | Meaning
//	-----|------------|---------
//	✓    | [ok]       | present/approved
//	✗    | [missing]  | missing
//	⚠    | [warn]     | attention
//	●    | [*]        | active/in-progress
//	○    | [ ]        | ready/queued
//	·    | -          | separator in counts
func Symbol(name string, tty bool) string {
	if tty {
		return ttySymbols[name]
	}
	return asciiSymbols[name]
}

var ttySymbols = map[string]string{
	"ok":       "✓",
	"missing":  "✗",
	"warn":     "⚠",
	"active":   "●",
	"ready":    "○",
	"separator": "·",
}

var asciiSymbols = map[string]string{
	"ok":        "[ok]",
	"missing":   "[missing]",
	"warn":      "[warn]",
	"active":    "[*]",
	"ready":     "[ ]",
	"separator": "-",
}
