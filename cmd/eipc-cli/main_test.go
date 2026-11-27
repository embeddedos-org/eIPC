package main

import (
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
	"testing"
)

// TestDocCommentListsRealCommands verifies that every command name in the
// package doc comment corresponds to a case in the main switch.
func TestDocCommentListsRealCommands(t *testing.T) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "main.go", nil, parser.ParseComments)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	// Extract command names from the doc comment ("eipc-cli <cmd> ...").
	var docCmds []string
	if f.Doc != nil {
		for _, c := range f.Doc.List {
			// Only the indented usage block ("//\teipc-cli <cmd> ..."); prose lines
			// such as "eipc-cli is a debugging ..." are not commands.
			if !strings.HasPrefix(c.Text, "//\t") {
				continue
			}
			line := strings.TrimSpace(strings.TrimPrefix(c.Text, "//"))
			if strings.HasPrefix(line, "eipc-cli ") {
				parts := strings.Fields(line)
				if len(parts) >= 2 {
					docCmds = append(docCmds, parts[1])
				}
			}
		}
	}
	if len(docCmds) == 0 {
		t.Fatal("no eipc-cli commands found in doc comment")
	}

	// Collect the case labels from the switch in main().
	switchCmds := map[string]bool{}
	ast.Inspect(f, func(n ast.Node) bool {
		sw, ok := n.(*ast.SwitchStmt)
		if !ok {
			return true
		}
		for _, stmt := range sw.Body.List {
			cc, ok := stmt.(*ast.CaseClause)
			if !ok || cc.List == nil {
				continue
			}
			for _, expr := range cc.List {
				bl, ok := expr.(*ast.BasicLit)
				if ok {
					switchCmds[strings.Trim(bl.Value, `"`)] = true
				}
			}
		}
		return true
	})

	for _, cmd := range docCmds {
		if !switchCmds[cmd] {
			t.Errorf("doc comment lists %q but main() has no case for it", cmd)
		}
	}
}
