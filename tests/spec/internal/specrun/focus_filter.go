package specrun

// FocusFilter resolves the spec focus list: an explicitly passed flag wins,
// otherwise the environment value as a single entry, otherwise no focus.
func FocusFilter(flagPassed bool, given []string, env string) []string {
	if flagPassed {
		return given
	}
	if env != "" {
		return []string{env}
	}
	return nil
}
