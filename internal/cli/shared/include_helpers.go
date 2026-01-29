package shared

// HasInclude returns true when include is present in values.
func HasInclude(values []string, include string) bool {
	for _, value := range values {
		if value == include {
			return true
		}
	}
	return false
}
