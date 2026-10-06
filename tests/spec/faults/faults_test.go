package faults

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"slices"
	"strings"
	"testing"
)

func TestAllFaultsSize(t *testing.T) {
	if len(AllFaults) != 89 {
		t.Fatalf("len(AllFaults) = %d, want 89", len(AllFaults))
	}
}

func TestAllFaultsNamesUnique(t *testing.T) {
	seen := make(map[string]bool)
	for _, f := range AllFaults {
		name := f.Name()
		if seen[name] {
			t.Errorf("duplicate fault name %s", name)
		}
		seen[name] = true
	}
}

func TestCatalogueNames(t *testing.T) {
	byName := faultsByName()
	present := []string{
		"RequestLost/ActivateSession",
		"ResponseLost/Publish",
		"CutAfterResponse/HEL",
		"DelayBelowTimeout/Read",
		"DelayAboveTimeout/Publish",
		"Overload/Publish/Bad_TooManyPublishRequests",
		"Overload/Read/Bad_ResourceUnavailable",
		"Link/ClosedOnAccept",
		"Link/ListenerClosed",
		"Link/Stall",
		"Link/HELUnanswered",
		"Server/Pause",
		"Server/DuplicateSequence",
		"Server/SkippedSequence",
		"Consumer/Slow",
		"RequestLost/CloseSecureChannel",
	}
	for _, want := range present {
		if _, ok := byName[want]; !ok {
			t.Errorf("AllFaults is missing %s", want)
		}
	}
	absent := []string{
		"DelayAboveTimeout/HEL",
		"Overload/HEL/Bad_TooManyOperations",
		"ResponseLost/CloseSecureChannel",
		"CutAfterResponse/CloseSecureChannel",
		"DelayBelowTimeout/CloseSecureChannel",
		"DelayAboveTimeout/CloseSecureChannel",
	}
	for _, name := range absent {
		if _, ok := byName[name]; ok {
			t.Errorf("AllFaults contains %s, want it absent", name)
		}
	}
}

func TestMessageFaultsCoverEveryMember(t *testing.T) {
	members := messageMembersFromSource(t)
	if len(members) != 13 {
		t.Fatalf("parsed %d Message members from source, want 13: %v", len(members), members)
	}
	byName := faultsByName()
	for _, m := range members {
		var kinds []string
		switch m {
		case "HEL":
			kinds = []string{"RequestLost", "ResponseLost", "CutAfterResponse", "DelayBelowTimeout"}
		case "CloseSecureChannel":
			kinds = []string{"RequestLost"}
		default:
			kinds = []string{"RequestLost", "ResponseLost", "CutAfterResponse", "DelayBelowTimeout", "DelayAboveTimeout"}
		}
		for _, k := range kinds {
			want := k + "/" + m
			if _, ok := byName[want]; !ok {
				t.Errorf("AllFaults is missing %s", want)
			}
		}
	}
}

func TestAvailable(t *testing.T) {
	cases := []struct {
		fault   string
		sends   []Message
		wantNil bool
	}{
		{"RequestLost/HEL", []Message{Publish}, false},
		{"RequestLost/HEL", []Message{HEL, Publish}, true},
		{"Overload/Publish/Bad_TooManyPublishRequests", []Message{Read}, false},
		{"Overload/Publish/Bad_TooManyPublishRequests", []Message{Publish}, true},
		{"Link/ClosedOnAccept", []Message{Publish}, false},
		{"Link/ListenerClosed", []Message{Publish}, false},
		{"Link/HELUnanswered", []Message{Publish}, false},
		{"Link/ClosedOnAccept", []Message{HEL}, true},
		{"Link/ListenerClosed", []Message{HEL}, true},
		{"Link/HELUnanswered", []Message{HEL}, true},
		{"Link/Stall", []Message{}, true},
		{"Server/Pause", []Message{}, true},
		{"Consumer/Slow", []Message{}, true},
		{"Server/DuplicateSequence", []Message{Read}, false},
		{"Server/SkippedSequence", []Message{Read}, false},
		{"Server/DuplicateSequence", []Message{Publish}, true},
		{"Server/SkippedSequence", []Message{Publish}, true},
	}
	byName := faultsByName()
	for _, c := range cases {
		f, ok := byName[c.fault]
		if !ok {
			t.Errorf("no fault named %s", c.fault)
			continue
		}
		reason := f.Available(c.sends)
		if c.wantNil {
			if reason != nil {
				t.Errorf("%s.Available(%v) = %q, want nil", c.fault, c.sends, reason.Text)
			}
			continue
		}
		if reason == nil {
			t.Errorf("%s.Available(%v) = nil, want a reason", c.fault, c.sends)
		}
	}
}

func TestEveryReasonHasText(t *testing.T) {
	sendSets := [][]Message{{}, {Publish}, {HEL}, {Read, Publish}}
	for _, f := range AllFaults {
		for _, sends := range sendSets {
			if r := f.Available(sends); r != nil && r.Text == "" {
				t.Errorf("%s.Available(%v) returned a reason with empty Text", f.Name(), sends)
			}
		}
	}
}

func faultsByName() map[string]Fault {
	byName := make(map[string]Fault)
	for _, f := range AllFaults {
		byName[f.Name()] = f
	}
	return byName
}

func messageMembersFromSource(t *testing.T) []string {
	t.Helper()
	fset := token.NewFileSet()
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("read package dir: %v", err)
	}
	var members []string
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		file, err := parser.ParseFile(fset, name, nil, 0)
		if err != nil {
			t.Fatalf("parse %s: %v", name, err)
		}
		for _, decl := range file.Decls {
			gen, ok := decl.(*ast.GenDecl)
			if !ok || gen.Tok != token.CONST {
				continue
			}
			typ := ""
			for _, spec := range gen.Specs {
				vs, ok := spec.(*ast.ValueSpec)
				if !ok {
					continue
				}
				if id, ok := vs.Type.(*ast.Ident); ok {
					typ = id.Name
				} else if vs.Type != nil {
					typ = ""
				}
				if typ != "Message" {
					continue
				}
				for _, name := range vs.Names {
					if name.Name != "messageInvalid" && !slices.Contains(members, name.Name) {
						members = append(members, name.Name)
					}
				}
			}
		}
	}
	return members
}
