package kbzinit

import (
	"bufio"
	"bytes"
	"strings"
)

const managedMarker = "# kanbanzai-managed:"
const versionMarker = "# kanbanzai-version:"

// transformSkillContent ensures the managed-marker and version lines are
// present in src. If they already exist they are replaced in-place (so the
// canonical long-form text and the current version are always used). If either
// is absent it is injected at the very top of the output so the result is
// never marker-less regardless of what the embedded source contains.
//
// The function is a pure transformation — no I/O, no side effects.
func transformSkillContent(src []byte, version string) ([]byte, error) {
	var body bytes.Buffer
	scanner := bufio.NewScanner(bytes.NewReader(src))
	foundManaged := false
	foundVersion := false

	for scanner.Scan() {
		line := scanner.Text()
		switch {
		case strings.HasPrefix(line, managedMarker):
			foundManaged = true
			body.WriteString("# kanbanzai-managed: do not edit. Regenerate with: kanbanzai init --update-skills\n")
		case strings.HasPrefix(line, versionMarker):
			foundVersion = true
			body.WriteString("# kanbanzai-version: " + version + "\n")
		default:
			body.WriteString(line)
			body.WriteByte('\n')
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}

	// Both markers were already present — return the rewritten body as-is.
	if foundManaged && foundVersion {
		return body.Bytes(), nil
	}

	// One or both markers were absent — prepend the missing ones so the output
	// is never marker-less.
	var result bytes.Buffer
	if !foundManaged {
		result.WriteString("# kanbanzai-managed: do not edit. Regenerate with: kanbanzai init --update-skills\n")
	}
	if !foundVersion {
		result.WriteString("# kanbanzai-version: " + version + "\n")
	}
	result.Write(body.Bytes())
	return result.Bytes(), nil
}

// hasLine reports whether data contains a line that starts with prefix.
func hasLine(data []byte, prefix string) bool {
	scanner := bufio.NewScanner(bytes.NewReader(data))
	for scanner.Scan() {
		if strings.HasPrefix(scanner.Text(), prefix) {
			return true
		}
	}
	return false
}

// extractVersion returns the version string from the first line that starts
// with "# kanbanzai-version:". Returns empty string if not found.
func extractVersion(data []byte) string {
	scanner := bufio.NewScanner(bytes.NewReader(data))
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, versionMarker) {
			v := strings.TrimPrefix(line, versionMarker)
			return strings.TrimSpace(v)
		}
	}
	return ""
}
