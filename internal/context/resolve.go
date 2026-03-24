package context

import "fmt"

// ResolveChain returns the inheritance chain from root ancestor to leaf (leaf is last).
// Returns an error if any inherits reference does not resolve or a cycle is detected.
func ResolveChain(store *ProfileStore, id string) ([]*Profile, error) {
	var chain []*Profile
	visited := make(map[string]bool)

	currentID := id
	for currentID != "" {
		if visited[currentID] {
			return nil, fmt.Errorf("cycle detected in inheritance chain at profile %q", currentID)
		}
		visited[currentID] = true

		p, err := store.Load(currentID)
		if err != nil {
			return nil, fmt.Errorf("resolve chain for %q: %w", id, err)
		}
		chain = append(chain, p)
		currentID = p.Inherits
	}

	// chain is currently leaf→root; reverse to root→leaf.
	for i, j := 0, len(chain)-1; i < j; i, j = i+1, j-1 {
		chain[i], chain[j] = chain[j], chain[i]
	}
	return chain, nil
}

// ResolveProfile walks the inheritance chain and returns the fully resolved profile
// using leaf-level replace semantics (P2-DEC-002):
//   - Scalars: leaf (child) wins over ancestor (parent).
//   - Slices: leaf's slice replaces parent's slice entirely; no concatenation.
//   - Maps (*Architecture): leaf's map replaces parent's map entirely.
//   - Absent fields (nil slice, nil pointer, empty string): inherited from parent unchanged.
//   - id is always taken from the leaf; inherits is never inherited.
func ResolveProfile(store *ProfileStore, id string) (*ResolvedProfile, error) {
	chain, err := ResolveChain(store, id)
	if err != nil {
		return nil, err
	}

	resolved := &ResolvedProfile{}

	for _, p := range chain {
		if p.Description != "" {
			resolved.Description = p.Description
		}
		if p.Packages != nil {
			resolved.Packages = p.Packages
		}
		if p.Conventions != nil {
			resolved.Conventions = p.Conventions
		}
		if p.Architecture != nil {
			resolved.Architecture = p.Architecture
		}
	}

	// id always comes from the leaf (last element in chain).
	if len(chain) > 0 {
		resolved.ID = chain[len(chain)-1].ID
	}

	return resolved, nil
}
