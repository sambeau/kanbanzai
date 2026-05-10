package kbzinit

import (
	_ "embed"
	"fmt"
	"os"
)

//go:embed cursor_rules/kanbanzai.mdc
var embeddedCursorRule []byte

// installCursorRule installs .cursor/rules/kanbanzai.mdc when at least one of
// these conditions is true (REQ-005/006):
//   (a) .cursor/ directory already exists at install time, or
//   (b) the user passed --enable-cursor.
//
// When neither condition holds, installation is skipped silently (no warning).
// When either condition holds, .cursor/rules/ is created if absent and
// kanbanzai.mdc is written via installArtifact with the standard MarkerSpec
// comparator. Both conditions being true simultaneously is idempotent.
func (i *Initializer) installCursorRule(baseDir string, enableCursor bool) error {
	a := manifestByKind(CursorRule)
	if a == nil {
		return fmt.Errorf("kanbanzai.mdc not found in Manifest")
	}

	cursorDir := baseDir + "/.cursor"
	cursorExists := false
	if info, err := os.Stat(cursorDir); err == nil && info.IsDir() {
		cursorExists = true
	}

	if !cursorExists && !enableCursor {
		return nil // silently skip
	}

	return installArtifact(*a, embeddedCursorRule, i.stdout, baseDir)
}
