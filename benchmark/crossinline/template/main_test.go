package main

import (
	"bytes"
	"fmt"
	"testing"
)

var outputSize int

func BenchmarkTemplateExecute(b *testing.B) {
	tmpl := parseTemplate()
	records := makeRecords()
	var output bytes.Buffer
	// Parse and grow the reusable output buffer before timing execution.
	if err := tmpl.Execute(&output, records); err != nil {
		b.Fatal(err)
	}
	b.SetBytes(int64(output.Len()))
	b.ResetTimer()
	for range b.N {
		output.Reset()
		if err := tmpl.Execute(&output, records); err != nil {
			b.Fatal(err)
		}
		outputSize = output.Len()
	}
}

func TestTemplateOutput(t *testing.T) {
	records := makeRecords()
	var output, want bytes.Buffer
	if err := parseTemplate().Execute(&output, records); err != nil {
		t.Fatal(err)
	}
	for _, r := range records {
		state := "inactive"
		if r.Active {
			state = "active"
		}
		fmt.Fprintf(&want, "%d|%s|%d|%s\n", r.ID, r.Name, r.Amount, state)
	}
	if !bytes.Equal(output.Bytes(), want.Bytes()) {
		t.Fatalf("template output mismatch:\ngot: %q\nwant: %q", output.String(), want.String())
	}
}
