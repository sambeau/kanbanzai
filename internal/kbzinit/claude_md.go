package kbzinit

import (
	_ "embed"
	"fmt"
)

//go:embed CLAUDE.md
var embeddedClaudeMd []byte

// installClaudeMd writes CLAUDE.md to baseDir using the Manifest's ClaudeMd
// artifact and compareManaged decision logic.
func (i *Initializer) installClaudeMd(baseDir string) error {
	a := manifestByKind(ClaudeMd)
	if a == nil {
		return fmt.Errorf("CLAUDE.md not found in Manifest")
	}
	return installArtifact(*a, embeddedClaudeMd, i.stdout, baseDir)
}
