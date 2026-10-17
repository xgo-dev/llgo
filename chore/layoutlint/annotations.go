package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"golang.org/x/tools/go/packages"
)

// writeAnnotations renders observed facts after a failure. It never changes
// the input or the generated source that was compiled into the executable.
func writeAnnotations(directory, source string, pkg *packages.Package, checks []fieldCheck, results, nested []offsetResult, metadata []metadataResult, private []privateLayout) error {
	type note struct {
		subject, text string
	}
	notes := make(map[string]map[int][]note)
	for _, file := range pkg.Syntax {
		name := pkg.Fset.PositionFor(file.Pos(), false).Filename
		notes[name] = make(map[int][]note)
	}
	notes[source] = make(map[int][]note)
	for _, plan := range private {
		for _, file := range plan.pkg.Syntax {
			name := plan.pkg.Fset.PositionFor(file.Pos(), false).Filename
			if notes[name] == nil {
				notes[name] = make(map[int][]note)
			}
		}
		nested = append(slices.Clone(nested), plan.Fields...)
		metadata = append(slices.Clone(metadata), plan.Metadata)
	}
	add := func(file string, line int, subject, text string) {
		lines, ok := notes[file]
		if !ok {
			// A referenced dependency's declarations are not part of the input
			// package's source copy. Its observations remain in offsets.json.
			return
		}
		entry := note{subject: subject, text: text}
		if !slices.ContainsFunc(lines[line], func(previous note) bool {
			return previous.text == text && (file != source || previous.subject == subject)
		}) {
			lines[line] = append(lines[line], entry)
		}
	}
	for _, observations := range [][]offsetResult{results, nested} {
		for _, result := range observations {
			actual, width := "unobserved", "unobserved"
			var reasons []string
			if result.Observed {
				actual = fmt.Sprint(result.Actual)
				if result.Actual != result.Expected {
					reasons = append(reasons, "offset mismatch")
				}
			}
			if result.Size.Observed {
				width = fmt.Sprint(result.Size.Actual)
				if result.Size.Actual != result.Size.Expected {
					reasons = append(reasons, "field size mismatch")
				}
			}
			if alignment := result.Alignment; alignment != nil {
				if !alignment.ObjectMatches {
					reasons = append(reasons, fmt.Sprintf("parent address %s is not aligned to %d", alignment.ObjectAddress, alignment.ObjectRequired))
				}
				if !alignment.FieldMatches {
					reasons = append(reasons, fmt.Sprintf("field address %s is not aligned to %d", alignment.FieldAddress, alignment.FieldRequired))
				}
			}
			text := fmt.Sprintf("GoOffset: %d GoSize: %d LLGoOffset: %s LLGoSize: %s", result.Expected, result.Size.Expected, actual, width)
			missingMetadata := false
			if result.Operation != "array-element" {
				value := "unobserved"
				if result.MetadataOffset != nil {
					value = fmt.Sprint(*result.MetadataOffset)
					if *result.MetadataOffset != uint64(result.Expected) {
						reasons = append(reasons, "metadata offset differs from Go")
					}
					if result.Observed && *result.MetadataOffset != uint64(result.Actual) {
						reasons = append(reasons, "metadata offset differs from actual address")
					}
				} else {
					missingMetadata = true
				}
				text += " MetadataOffset: " + value
			}
			if len(reasons) != 0 {
				text += "; FAIL: " + strings.Join(reasons, "; ")
			}
			if !result.Observed || !result.Size.Observed || missingMetadata {
				text += "; NOT OBSERVED: check did not complete"
			}
			add(result.File, result.Line, result.Type+"."+result.Field, text)
		}
	}
	for _, result := range metadata {
		actual := "unobserved"
		var reasons []string
		if stride := result.Stride; stride != nil {
			actual = fmt.Sprint(stride.Bytes)
			if !stride.GoMatches {
				reasons = append(reasons, "struct size/array stride mismatch")
			}
			if !stride.MetadataMatches {
				reasons = append(reasons, fmt.Sprintf("metadata.Size_: %d disagrees with actual stride %d", result.TypeHeader.Size, stride.Bytes))
			}
		}
		for _, difference := range result.Differences {
			reasons = append(reasons, fmt.Sprintf("metadata.%s.%s: Go: %d LLGo: %d", difference.Descriptor, difference.Field, difference.Expected, difference.Actual))
		}
		text := fmt.Sprintf("GoSize: %d LLGoSize: %s", result.Go.Layout.Size, actual)
		if len(reasons) != 0 {
			text += "; FAIL: " + strings.Join(reasons, "; ")
		}
		if !result.Observed || result.Stride == nil {
			text += "; NOT OBSERVED: metadata/stride check did not complete"
		}
		add(result.File, result.Line, result.Type, text)
	}

	// Find the actual dereference statement in the compiled wrapper. The
	// original program's accesses were not executed by this checker.
	fset := token.NewFileSet()
	generated, err := parser.ParseFile(fset, source, nil, 0)
	if err != nil {
		return err
	}
	accessLines := make(map[string]int)
	for _, declaration := range generated.Decls {
		fn, ok := declaration.(*ast.FuncDecl)
		if !ok || fn.Body == nil {
			continue
		}
		for _, statement := range fn.Body.List {
			assignment, ok := statement.(*ast.AssignStmt)
			if !ok {
				continue
			}
			for _, expression := range append(slices.Clone(assignment.Lhs), assignment.Rhs...) {
				pointer, ok := expression.(*ast.StarExpr)
				if !ok {
					continue
				}
				name, ok := pointer.X.(*ast.Ident)
				if ok && name.Name == "field" {
					accessLines[fn.Name.Name] = fset.PositionFor(statement.End(), false).Line
				}
			}
		}
	}
	for _, result := range results {
		check := checks[result.checkIndex]
		if check.kind == types.Invalid {
			continue
		}
		operation, name := "load", check.wrapper
		if result.Operation == "write" {
			operation = "store"
		}
		var reasons []string
		access := result.Access
		if access == nil || access.ActualValue == "" {
			reasons = append(reasons, "NOT OBSERVED: access check did not complete")
		} else {
			if !access.ValueMatches {
				reasons = append(reasons, fmt.Sprintf("FAIL: bytes want %s, got %s", access.ExpectedValue, access.ActualValue))
			}
			if !access.MemoryMatches {
				reasons = append(reasons, fmt.Sprintf("FAIL: unexpected memory changes at object offsets %v", access.ChangedBytes))
			}
		}
		if len(reasons) == 0 {
			continue
		}
		line, ok := accessLines[name]
		if !ok {
			return fmt.Errorf("cannot locate access statement in %s", name)
		}
		add(source, line, fmt.Sprintf("%s address case %d", operation, result.AddressCase), strings.Join(reasons, "; "))
	}

	files := make([]string, 0, len(notes))
	for file, lines := range notes {
		if len(lines) != 0 {
			files = append(files, file)
		}
	}
	slices.Sort(files)
	for _, file := range files {
		data, err := os.ReadFile(file)
		if err != nil {
			return err
		}
		lines := strings.Split(string(data), "\n")
		for line, entries := range notes[file] {
			if line <= 0 || line > len(lines) {
				return fmt.Errorf("invalid annotation location %s:%d", file, line)
			}
			var comments []string
			for _, entry := range entries {
				text := entry.text
				if len(entries) > 1 || file == source {
					text = entry.subject + ": " + text
				}
				comments = append(comments, text)
			}
			// Append without reformatting: original line numbers and comments
			// remain intact, including fields sharing one declaration line.
			ending := ""
			if strings.HasSuffix(lines[line-1], "\r") {
				ending = "\r"
			}
			lines[line-1] = strings.TrimRight(lines[line-1], "\r\t ") + " // " + strings.Join(comments, " | ") + ending
		}
		output := filepath.Join(directory, "annotated", filepath.Base(file))
		if err := os.MkdirAll(filepath.Dir(output), 0755); err != nil {
			return err
		}
		if err := os.WriteFile(output, []byte(strings.Join(lines, "\n")), 0644); err != nil {
			return err
		}
	}
	return nil
}
