package context

import "fmt"

// ResolveRoleChain returns the inheritance chain from root ancestor to leaf (leaf is last).
// Returns an error if any inherits reference does not resolve or a cycle is detected.
func ResolveRoleChain(store *RoleStore, id string) ([]*Role, error) {
	var chain []*Role
	visited := make(map[string]bool)

	currentID := id
	for currentID != "" {
		if visited[currentID] {
			return nil, fmt.Errorf("cycle detected in role inheritance chain at %q", currentID)
		}
		visited[currentID] = true

		r, err := store.Load(currentID)
		if err != nil {
			return nil, fmt.Errorf("resolve role chain for %q: %w", id, err)
		}
		chain = append(chain, r)
		currentID = r.Inherits
	}

	// chain is currently leaf→root; reverse to root→leaf.
	for i, j := 0, len(chain)-1; i < j; i, j = i+1, j-1 {
		chain[i], chain[j] = chain[j], chain[i]
	}
	return chain, nil
}

// ResolveRole walks the inheritance chain and returns the fully resolved role.
// Merge semantics per FR-010:
//   - vocabulary: parent ++ child (concatenation)
//   - anti_patterns: parent ++ child (concatenation)
//   - tools: union (no duplicates, order preserved: parent first, then child additions)
//   - identity: leaf only (not inherited)
//   - id: leaf only
func ResolveRole(store *RoleStore, id string) (*ResolvedRole, error) {
	chain, err := ResolveRoleChain(store, id)
	if err != nil {
		return nil, err
	}

	if len(chain) == 0 {
		return nil, fmt.Errorf("empty inheritance chain for role %q", id)
	}

	resolved := &ResolvedRole{}

	// Walk from root to leaf, accumulating vocabulary and anti-patterns
	// via concatenation, and tools via union.
	for _, r := range chain {
		// Vocabulary: concatenate parent ++ child.
		resolved.Vocabulary = append(resolved.Vocabulary, r.Vocabulary...)

		// Anti-patterns: concatenate parent ++ child.
		resolved.AntiPatterns = append(resolved.AntiPatterns, r.AntiPatterns...)

		// Tools: union, preserving order (parent entries first).
		resolved.Tools = mergeToolsUnion(resolved.Tools, r.Tools)
	}

	// id, identity, and last_verified always come from the leaf (last element in chain).
	leaf := chain[len(chain)-1]
	resolved.ID = leaf.ID
	resolved.Identity = leaf.Identity
	resolved.LastVerified = leaf.LastVerified

	return resolved, nil
}

// mergeToolsUnion appends entries from additions to base, skipping duplicates.
// Order is preserved: existing entries in base keep their position, new entries
// from additions are appended in their original order.
func mergeToolsUnion(base, additions []string) []string {
	seen := make(map[string]bool, len(base))
	for _, t := range base {
		seen[t] = true
	}

	result := make([]string, len(base))
	copy(result, base)

	for _, t := range additions {
		if !seen[t] {
			seen[t] = true
			result = append(result, t)
		}
	}
	return result
}
