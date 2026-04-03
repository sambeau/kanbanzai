package config

// MergeToolHints returns the effective tool hints map. Local hints override
// project hints on a per-key basis. Either or both inputs may be nil.
func MergeToolHints(project, local map[string]string) map[string]string {
	if len(project) == 0 && len(local) == 0 {
		return nil
	}
	merged := make(map[string]string, len(project)+len(local))
	for k, v := range project {
		merged[k] = v
	}
	for k, v := range local {
		merged[k] = v // local wins
	}
	return merged
}
