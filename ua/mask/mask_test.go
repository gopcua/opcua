package mask

import "testing"

func TestMask(t *testing.T) {

	if got, want := Set(0, 2), byte(2); got != want {
		t.Fatalf("got %x want %x", got, want)
	}
	if got, want := Set(2, 2), byte(2); got != want {
		t.Fatalf("got %x want %x", got, want)
	}
	if got, want := Set(2, 1), byte(3); got != want {
		t.Fatalf("got %x want %x", got, want)
	}
	if got, want := Has(2, 1), false; got != want {
		t.Fatalf("got %v want %v", got, want)
	}
	if got, want := Has(2, 2), true; got != want {
		t.Fatalf("got %v want %v", got, want)
	}
	if got, want := Has(3, 2), true; got != want {
		t.Fatalf("got %v want %v", got, want)
	}
	if got, want := Clear(3, 2), byte(1); got != want {
		t.Fatalf("got %v want %v", got, want)
	}
	if got, want := Clear(3, 4), byte(3); got != want {
		t.Fatalf("got %v want %v", got, want)
	}
}
