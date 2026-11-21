package provider

// Arcane v3 expands every role and API key permission with the permissions it
// requires (for example "projects:read" also grants "projects:list"), so the
// server reports a superset of what was configured.
//
// impliedSuperset reports whether actual still contains every configured
// permission. When it does, the configured set is kept in state: the extras are
// the server-added dependencies and must not show up as a diff. When a
// configured permission is missing, the server value is used so the drift is
// visible and the next apply restores it.
func impliedSuperset[T comparable](configured, actual []T) bool {
	if len(configured) == 0 {
		return false
	}
	have := make(map[T]bool, len(actual))
	for _, a := range actual {
		have[a] = true
	}
	for _, c := range configured {
		if !have[c] {
			return false
		}
	}
	return true
}
