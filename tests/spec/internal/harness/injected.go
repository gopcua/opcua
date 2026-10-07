package harness

// Injected is one armed fault the catalogue's Inject returned; Fired
// reports whether the harness observed the fault happening.
type Injected struct {
	check func() bool
}

// NewInjected builds an Injected whose Fired reports what check says.
// A nil check never fires.
func NewInjected(check func() bool) *Injected {
	return &Injected{check: check}
}

// Fired reports whether the harness observed the armed fault
// happening.
func (i *Injected) Fired() bool {
	if i == nil || i.check == nil {
		return false
	}
	return i.check()
}
