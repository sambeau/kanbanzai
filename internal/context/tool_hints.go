package context

// ResolveToolHint returns the effective tool hint for the given role ID.
// Resolution order: exact match in hints map, then walk the role's inherits
// chain via the RoleStore. Returns "" if no hint resolves.
func ResolveToolHint(hints map[string]string, roleID string, store *RoleStore) string {
	if len(hints) == 0 {
		return ""
	}
	// 1. Exact match.
	if hint, ok := hints[roleID]; ok {
		return hint
	}
	// 2. Walk inheritance chain.
	current := roleID
	visited := make(map[string]bool)
	for {
		visited[current] = true
		r, err := store.Load(current)
		if err != nil || r.Inherits == "" {
			break
		}
		parent := r.Inherits
		if visited[parent] {
			break // cycle protection
		}
		if hint, ok := hints[parent]; ok {
			return hint
		}
		current = parent
	}
	return ""
}
