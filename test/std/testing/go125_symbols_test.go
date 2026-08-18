//go:build go1.25

package testing_test

import (
	"fmt"
	"testing"
)

func TestOutputAndAttr(t *testing.T) {
	t.Attr("go-version", "1.25+")
	if _, err := fmt.Fprintln(t.Output(), "test output"); err != nil {
		t.Fatal(err)
	}
}

func TestBenchmarkOutputAndAttr(t *testing.T) {
	result := testing.Benchmark(func(b *testing.B) {
		b.Attr("go-version", "1.25+")
		fmt.Fprintln(b.Output(), "benchmark output")
		for range b.N {
		}
	})
	if result.N <= 0 {
		t.Fatalf("Benchmark ran %d iterations", result.N)
	}
}

func FuzzOutputAndAttr(f *testing.F) {
	f.Attr("go-version", "1.25+")
	fmt.Fprintln(f.Output(), "fuzz output")
	f.Add("seed")
	f.Fuzz(func(t *testing.T, input string) {
		if input == "" {
			t.Skip()
		}
	})
}
