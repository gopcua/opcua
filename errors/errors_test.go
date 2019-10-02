package errors

import "testing"

func TestErrors(t *testing.T) {
	err := Errorf("hello %s", "world")
	if err.Error() != "opcua: hello world" {
		t.Fatalf("got %s, wanted %s", err.Error(), "opcua: hello world")
	}

	err = New("hello")
	if err.Error() != "opcua: hello" {
		t.Fatalf("got %s, wanted %s", err.Error(), "opcua: hello")
	}

	err = New("hello %s")
	if err.Error() != "opcua: hello %s" {
		t.Fatalf("got %s, wanted %s", err.Error(), "opcua: %s")
	}
}
