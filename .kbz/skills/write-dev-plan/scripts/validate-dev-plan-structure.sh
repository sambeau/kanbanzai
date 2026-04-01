#!/bin/sh
# Usage: validate-dev-plan-structure.sh <path-to-document>
# Exit 0: all required sections and cross-references present
# Exit 1: missing sections or cross-references (names printed to stdout)
# Required sections: "Scope", "Task Breakdown", "Dependency Graph",
#                    "Risk Assessment", "Verification Approach"
# Cross-reference check: Scope must contain a reference to a specification
# Dependencies: POSIX shell utilities only (grep, sed)
# Runtime: < 5 seconds on files up to 2000 lines

if [ -z "$1" ]; then
  echo "Usage: validate-dev-plan-structure.sh <path-to-document>" >&2
  exit 2
fi

if [ ! -f "$1" ]; then
  echo "Error: file not found: $1" >&2
  exit 2
fi

missing=0

# Check for each required section heading (## or ### level)
for section in "Scope" "Task Breakdown" "Dependency Graph" "Risk Assessment" "Verification Approach"; do
  if ! grep -qE "^#{1,3}[[:space:]]+${section}[[:space:]]*$" "$1"; then
    echo "Missing section: ${section}"
    missing=1
  fi
done

# Cross-reference check: Scope section must reference a specification.
# Extract content between the Scope heading and the next heading of equal or higher level.
scope_content=$(sed -n '/^##*[[:space:]]*Scope[[:space:]]*$/,/^##*[[:space:]]/{/^##*[[:space:]]*Scope[[:space:]]*$/d;/^##*[[:space:]]/d;p;}' "$1")

if [ -z "$scope_content" ]; then
  echo "Missing: Scope section is empty or could not be parsed"
  missing=1
else
  if ! printf '%s\n' "$scope_content" | grep -qiE '(spec|specification|DOC-|/spec/)'; then
    echo "Missing: Scope does not reference a specification"
    missing=1
  fi
fi

if [ "$missing" -ne 0 ]; then
  exit 1
fi

echo "All required sections and cross-references present."
exit 0
