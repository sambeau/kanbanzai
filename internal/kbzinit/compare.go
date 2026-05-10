package kbzinit

import (
	"bufio"
	"bytes"
	"strconv"
	"strings"
)

// compareManaged examines the on-disk content of a managed file and returns
// the Decision that installArtifact should take.
//
// Rules (REQ-004):
//   - nil or empty existing     → Create      (file absent)
//   - no line matching spec.Comment → WarnSkip (user-authored, preserve it)
//   - marker found, version unparseable → WarnSkip (do not silently overwrite)
//   - marker found, version older       → Overwrite
//   - marker found, version equal/newer → NoOp
//
// The function is pure: no I/O side effects, no global state reads.
func compareManaged(existing []byte, spec MarkerSpec) Decision {
	if len(existing) == 0 {
		return Create
	}

	line, found := findMarkerLine(existing, spec.Comment)
	if !found {
		return WarnSkip
	}

	// Extract the version from the marker line: strip the comment prefix,
	// drop any trailing HTML comment close ("-->") or surrounding quotes
	// (to handle YAML role files where version is stored as `version: "x.y.z"`),
	// then trim whitespace.
	raw := strings.TrimPrefix(line, spec.Comment)
	raw = strings.TrimSuffix(strings.TrimSpace(raw), "-->")
	raw = strings.TrimSpace(raw)
	raw = strings.Trim(raw, `"`)

	switch spec.VersionKind {
	case IntCounter:
		return compareIntCounter(raw, spec.CurrentValue)
	case Semver:
		return compareSemverVersions(raw, spec.CurrentValue)
	default:
		return WarnSkip
	}
}

// findMarkerLine scans data line-by-line and returns the first line whose
// text starts with prefix. The second return value is false when no match
// is found.
func findMarkerLine(data []byte, prefix string) (string, bool) {
	scanner := bufio.NewScanner(bytes.NewReader(data))
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, prefix) {
			return line, true
		}
	}
	return "", false
}

// compareIntCounter parses both existing and current as plain integers and
// returns Overwrite when existing < current, NoOp otherwise.
// Returns WarnSkip when existing is not a valid integer.
func compareIntCounter(existing, current string) Decision {
	existingN, err := strconv.Atoi(existing)
	if err != nil {
		return WarnSkip
	}
	currentN, err := strconv.Atoi(current)
	if err != nil {
		// Binary current value should always be valid; fail safe to NoOp.
		return NoOp
	}
	if existingN < currentN {
		return Overwrite
	}
	return NoOp
}

// compareSemverVersions compares two "vMAJOR.MINOR.PATCH" strings and returns
// Overwrite when existing < current, NoOp otherwise.
// Returns WarnSkip when existing cannot be parsed.
func compareSemverVersions(existing, current string) Decision {
	existingParts, err := parseSemverParts(existing)
	if err != nil {
		return WarnSkip
	}
	currentParts, err := parseSemverParts(current)
	if err != nil {
		// Binary current value should always be valid; fail safe to NoOp.
		return NoOp
	}
	for i := range existingParts {
		if existingParts[i] < currentParts[i] {
			return Overwrite
		}
		if existingParts[i] > currentParts[i] {
			return NoOp
		}
	}
	return NoOp // equal
}

// parseSemverParts splits a "vMAJOR.MINOR.PATCH" string into its three
// numeric components. The leading "v" is optional.
func parseSemverParts(v string) ([3]int, error) {
	v = strings.TrimPrefix(v, "v")
	parts := strings.SplitN(v, ".", 3)
	if len(parts) != 3 {
		return [3]int{}, strconv.ErrSyntax
	}
	var result [3]int
	for i, p := range parts {
		n, err := strconv.Atoi(p)
		if err != nil {
			return [3]int{}, err
		}
		result[i] = n
	}
	return result, nil
}
