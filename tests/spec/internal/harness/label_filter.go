package harness

// LabelFilter resolves the spec label filter: an explicitly passed flag wins
// (even when empty), otherwise the environment value, otherwise the default
// "!known-defect". The default is a Ginkgo label-filter expression, where
// "!label" excludes every spec carrying that label
// (https://onsi.github.io/ginkgo/#spec-labels). It excludes specs labelled
// "known-defect": specs measured to fail on `main` because the implementation
// under test violates the behaviour they assert.
//
// An explicit empty filter runs every spec. Only the go test form forwards
// it to this binary; ginkgo v2.33.0's GenerateFlagArgs (types/flags.go,
// the empty-string check next to flag.AlwaysExport) drops a string flag
// whose value is empty, so the CLI never passes an empty filter through.
// To run every spec through the ginkgo CLI, pass an expression that
// matches everything, for example `!known-defect || known-defect`.
func LabelFilter(flagPassed bool, given string, env string) string {
	if flagPassed {
		return given
	}
	if env != "" {
		return env
	}
	return "!known-defect"
}

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
