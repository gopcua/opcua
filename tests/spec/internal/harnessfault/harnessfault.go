// Package harnessfault holds the prefix that marks a spec failure as a
// fault in the spec harness rather than in the client under test.
package harnessfault

// Prefix starts every failure message the spec harness raises for its
// own fault, so the message reads as a fault in the harness, not in
// the client under test.
const Prefix = "spectest: "
