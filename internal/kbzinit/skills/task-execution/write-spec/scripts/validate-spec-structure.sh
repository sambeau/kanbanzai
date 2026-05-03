#!/bin/sh
# Usage: validate-spec-structure.sh <path-to-document>
# Exit 0: all required sections and cross-references present
# Exit 1: missing sections or cross-references (names printed to stdout)
# Required sections: "Overview", "Scope", "Functional Requirements",
#                    "Non-Functional Requirements", "Acceptance Criteria",
#                    "Verification Plan"
# Cross-reference check: Overview must contain a reference to a
#                        design document (path or document ID)
# Dependencies: POSIX shell utilities only (grep, sed)
# Runtime: < 5 seconds on files up to 2000 lines

if [ -z "$1" ]; then
  echo "Usage: validate-spec-structure.sh <path-to-document>" >&2
  exit 2
fi

if [ ! -f "$1" ]; then
  echo "Error: file not found: $1" >&2
  exit 2
fi

missing=0

# Check each required section heading (## level only for top-level sections)
for section in "Overview" "Scope" "Functional Requirements" "Non-Functional Requirements" "Acceptance Criteria" "Verification Plan"; do
  if ! grep -qE "^##[[:space:]]+${section}[[:space:]]*$" "$1"; then
    echo "Missing section: ${section}"
    missing=1
  fi
done

# Cross-reference check: Overview must reference a design document.
# Extract content between Overview heading and the next heading.
overview_content=$(sed -n '/^##[[:space:]]*Overview[[:space:]]*$/,/^#/{/^#/d;p;}' "$1")

if [ -z "$overview_content" ]; then
  echo "Missing: Overview section is empty"
  missing=1
else
  if ! printf '%s\n' "$overview_content" | grep -qiE '(design|DOC-|/design/)'; then
    echo "Missing: Overview does not reference a design document"
    missing=1
  fi
fi

if [ "$missing" -ne 0 ]; then
  exit 1
fi

echo "All required sections and cross-references present."
exit 0
