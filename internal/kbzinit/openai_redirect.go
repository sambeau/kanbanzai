package kbzinit

import (
	_ "embed"
	"fmt"
)

//go:embed OPENAI.md
var embeddedOpenAiRedirect []byte

// installOpenAiRedirect writes OPENAI.md to baseDir using the Manifest's
// OpenAiRedirect artifact and compareManaged decision logic.
func (i *Initializer) installOpenAiRedirect(baseDir string) error {
	a := manifestByKind(OpenAiRedirect)
	if a == nil {
		return fmt.Errorf("OPENAI.md not found in Manifest")
	}
	return installArtifact(*a, embeddedOpenAiRedirect, i.stdout, baseDir)
}
