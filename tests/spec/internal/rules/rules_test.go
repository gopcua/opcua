package rules

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"regexp"
	"strings"
	"testing"
)

func TestRuleMetadata(t *testing.T) {
	seen := make(map[string]bool)
	for _, rule := range All() {
		if rule.Name == "" {
			t.Errorf("a rule has an empty Name: %+v", rule)
		}
		if seen[rule.Name] {
			t.Errorf("two rules share the name %s", rule.Name)
		}
		seen[rule.Name] = true
		if !regexp.MustCompile(`^P4-[0-9.]+$`).MatchString(rule.Clause) {
			t.Errorf("%s carries the clause %q, want a P4 label", rule.Name, rule.Clause)
		}
		if rule.Keyword != "shall" && rule.Keyword != "should" {
			t.Errorf("%s carries the keyword %q, want shall or should", rule.Name, rule.Keyword)
		}
		if rule.Check == nil {
			t.Errorf("%s has no Check", rule.Name)
		}
	}
}

func TestAllListsEveryRule(t *testing.T) {
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("read package dir: %v", err)
	}
	literals := 0
	fset := token.NewFileSet()
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		file, err := parser.ParseFile(fset, name, nil, 0)
		if err != nil {
			t.Fatalf("parse %s: %v", name, err)
		}
		ast.Inspect(file, func(node ast.Node) bool {
			literal, isComposite := node.(*ast.CompositeLit)
			if !isComposite {
				return true
			}
			if selector, isSelector := literal.Type.(*ast.SelectorExpr); isSelector && selector.Sel.Name == "Rule" {
				literals++
			}
			return true
		})
	}
	if literals != len(All()) {
		t.Fatalf("the package declares %d Rule literals but All() lists %d", literals, len(All()))
	}
}
