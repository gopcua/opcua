package matrix

// Rule is one Part 4 behaviour the specs assert: a stable Name the
// case's checks and the clause file's check Its are keyed by, and
// the Check that asserts it against the facts the caller observed.
// The workload returns the facts as an untyped value, and the clause
// package's Check asserts them to its own type, so a new rule never
// edits internal/.
type Rule struct {
	Name  string
	Check func(facts any)
}
