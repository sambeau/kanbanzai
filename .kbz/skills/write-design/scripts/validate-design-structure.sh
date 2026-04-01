#!/bin/sh
# Usage: validate-design-structure.sh <path-to-document>
# Exit 0: all required sections present
# Exit 1: one or more sections missing (names printed to stdout)
# Required sections: "Problem and Motivation", "Design", "Alternatives Considered", "Decisions"
# Dependencies: POSIX shell utilities only (grep)
# Runtime: < 5 seconds on files up to 2000 lines

if [ -z "$1" ]; then
  echo "Usage: validate-design-structure.sh <path-to-document>" >&2
  exit 2
fi

if [ ! -f "$1" ]; then
  echo "Error: file not found: $1" >&2
  exit 2
fi

missing=0

# Check for "Problem and Motivation" section
if ! grep -qE '^#{1,3}[[:space:]]+Problem and Motivation[[:space:]]*$' "$1"; then
  echo "Missing: Problem and Motivation"
  missing=1
fi

# Check for "Design" section (exact heading, not a longer heading like "Design Decisions")
if ! grep -qE '^#{1,3}[[:space:]]+Design[[:space:]]*$' "$1"; then
  echo "Missing: Design"
  missing=1
fi

# Check for "Alternatives Considered" section
if ! grep -qE '^#{1,3}[[:space:]]+Alternatives Considered[[:space:]]*$' "$1"; then
  echo "Missing: Alternatives Considered"
  missing=1
fi

# Check for "Decisions" section
if ! grep -qE '^#{1,3}[[:space:]]+Decisions[[:space:]]*$' "$1"; then
  echo "Missing: Decisions"
  missing=1
fi

if [ "$missing" -ne 0 ]; then
  exit 1
fi

echo "All required sections present."
exit 0
