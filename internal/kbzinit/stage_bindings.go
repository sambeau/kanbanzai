package kbzinit

import (
	"bufio"
	"bytes"
	_ "embed"
	"fmt"
	"os"
	"strings"
)

//go:embed stage-bindings.yaml
var embeddedStageBindings []byte

// installStageBindings installs the canonical stage-bindings.yaml from the
// Manifest into <kbzDir>/stage-bindings.yaml.
func (i *Initializer) installStageBindings(kbzDir string) error {
	a := manifestByKind(StageBindings)
	if a == nil {
		return fmt.Errorf("stage-bindings.yaml not found in Manifest")
	}

	// Transform: replace version placeholders with the binary version.
	content := transformStageBindingsContent(embeddedStageBindings, i.version)

	// Ensure the .kbz/ directory exists.
	if err := os.MkdirAll(kbzDir, 0o755); err != nil {
		return fmt.Errorf("create .kbz/: %w", err)
	}

	a.Marker.CurrentValue = i.version
	return installArtifact(*a, content, i.stdout, kbzDir)
}

// transformStageBindingsContent replaces the version placeholder in the
// embedded stage-bindings.yaml with the actual binary version.
func transformStageBindingsContent(src []byte, version string) []byte {
	var buf bytes.Buffer
	scanner := bufio.NewScanner(bytes.NewReader(src))
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "# kanbanzai-managed:") {
			buf.WriteString("# kanbanzai-managed: true\n")
			continue
		}
		if strings.HasPrefix(line, "# kanbanzai-version:") {
			buf.WriteString("# kanbanzai-version: " + version + "\n")
			continue
		}
		buf.WriteString(line)
		buf.WriteByte('\n')
	}
	return buf.Bytes()
}
