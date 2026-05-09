package registry

import (
	"fmt"
	"strings"
)

const (
	beginPrefix  = "<!-- registry-gen:begin:"
	endPrefix    = "<!-- registry-gen:end:"
	markerSuffix = " -->"
)

// Region holds the parsed position of a named generated region within file content.
// ContentStart is the byte offset of the first byte after the begin-marker line's
// newline. ContentEnd is the byte offset of the first byte of the end-marker line.
// Content is the raw bytes between those offsets.
type Region struct {
	Name         string
	Source       string
	ContentStart int
	ContentEnd   int
	Content      string
}

// ParseRegions parses all named generated regions from content.
// filePath is used only in error messages.
//
// It returns an error (naming the file and marker) for any of:
//   - A begin marker with no matching end marker.
//   - An end marker with no matching begin marker.
//   - A duplicated begin marker for the same region name.
//   - A nested begin marker found before the open region is closed.
func ParseRegions(filePath, content string) ([]Region, error) {
	type pending struct {
		name         string
		source       string
		contentStart int
	}

	var regions []Region
	var open *pending
	seen := make(map[string]bool)

	pos := 0
	for pos <= len(content) {
		nl := strings.Index(content[pos:], "\n")
		var line string
		var nextPos int
		if nl == -1 {
			// Last line (no trailing newline).
			line = content[pos:]
			nextPos = len(content) + 1 // sentinel to exit loop after this iteration
		} else {
			line = content[pos : pos+nl]
			nextPos = pos + nl + 1
		}

		trimmed := strings.TrimSpace(line)

		if strings.HasPrefix(trimmed, beginPrefix) {
			name, source, ok := parseBeginMarker(trimmed)
			if ok {
				if open != nil {
					if open.name == name {
						return nil, fmt.Errorf("%s: duplicated begin marker for region %q", filePath, name)
					}
					return nil, fmt.Errorf("%s: nested region %q found inside open region %q", filePath, name, open.name)
				}
				if seen[name] {
					return nil, fmt.Errorf("%s: duplicated begin marker for region %q", filePath, name)
				}
				open = &pending{
					name:         name,
					source:       source,
					contentStart: nextPos,
				}
			}
		} else if strings.HasPrefix(trimmed, endPrefix) {
			name, ok := parseEndMarker(trimmed)
			if ok {
				if open == nil {
					return nil, fmt.Errorf("%s: end marker for region %q has no matching begin marker", filePath, name)
				}
				if open.name != name {
					return nil, fmt.Errorf("%s: end marker for region %q does not match open region %q", filePath, name, open.name)
				}
				seen[name] = true
				regions = append(regions, Region{
					Name:         open.name,
					Source:       open.source,
					ContentStart: open.contentStart,
					ContentEnd:   pos,
					Content:      content[open.contentStart:pos],
				})
				open = nil
			}
		}

		if nextPos > len(content) {
			break
		}
		pos = nextPos
	}

	if open != nil {
		return nil, fmt.Errorf("%s: begin marker for region %q has no matching end marker", filePath, open.name)
	}

	return regions, nil
}

// CheckRegion compares the current content of a named region with candidate.
// It returns (stale, regionName, filePath, error).
// stale is true when the region's current content differs from candidate.
func CheckRegion(filePath, content, regionName, candidate string) (stale bool, region string, file string, err error) {
	regions, parseErr := ParseRegions(filePath, content)
	if parseErr != nil {
		return false, regionName, filePath, parseErr
	}
	for _, r := range regions {
		if r.Name == regionName {
			return r.Content != candidate, regionName, filePath, nil
		}
	}
	return false, regionName, filePath, fmt.Errorf("%s: region %q not found", filePath, regionName)
}

// SyncRegion replaces the content of a named region with candidate.
// All bytes before the opening marker and after the closing marker are
// preserved byte-for-byte. Running SyncRegion twice with the same candidate
// on an unchanged file produces identical bytes on the second run.
func SyncRegion(filePath, content, regionName, candidate string) (string, error) {
	regions, err := ParseRegions(filePath, content)
	if err != nil {
		return "", err
	}
	for _, r := range regions {
		if r.Name == regionName {
			var sb strings.Builder
			sb.WriteString(content[:r.ContentStart])
			sb.WriteString(candidate)
			sb.WriteString(content[r.ContentEnd:])
			return sb.String(), nil
		}
	}
	return "", fmt.Errorf("%s: region %q not found", filePath, regionName)
}

// parseBeginMarker extracts the region name and source attribute from a
// trimmed begin-marker line. Returns (name, source, ok).
func parseBeginMarker(trimmed string) (name, source string, ok bool) {
	inner := trimmed[len(beginPrefix):]
	if !strings.HasSuffix(inner, "-->") {
		return "", "", false
	}
	inner = strings.TrimSuffix(inner, "-->")
	inner = strings.TrimSpace(inner)

	// inner is now "NAME" or "NAME source=..." or "NAME source=... attr=..."
	spaceIdx := strings.IndexByte(inner, ' ')
	if spaceIdx == -1 {
		name = inner
	} else {
		name = inner[:spaceIdx]
		attrs := strings.TrimSpace(inner[spaceIdx+1:])
		for _, attr := range strings.Fields(attrs) {
			if strings.HasPrefix(attr, "source=") {
				source = strings.TrimPrefix(attr, "source=")
			}
		}
	}

	if name == "" {
		return "", "", false
	}
	return name, source, true
}

// parseEndMarker extracts the region name from a trimmed end-marker line.
// Returns (name, ok).
func parseEndMarker(trimmed string) (name string, ok bool) {
	inner := trimmed[len(endPrefix):]
	if !strings.HasSuffix(inner, "-->") {
		return "", false
	}
	inner = strings.TrimSuffix(inner, "-->")
	name = strings.TrimSpace(inner)
	if name == "" {
		return "", false
	}
	return name, true
}
