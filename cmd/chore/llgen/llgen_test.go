package main

import (
	"log"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xgo-dev/llgo/cmd/compiler/llgen"
)

func TestMain(t *testing.T) {
	// Create test package in current module
	testPkg := filepath.Join(".testdata_dont_commit", "hello")
	err := os.MkdirAll(testPkg, 0755)
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(filepath.Join(".testdata_dont_commit"))

	helloFile := filepath.Join(testPkg, "hello.go")
	err = os.WriteFile(helloFile, []byte(`package hello

func Hello() string {
	return "Hello, World!"
}
`), 0644)
	if err != nil {
		t.Fatal(err)
	}

	// Save original args and restore them after test
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	// Get absolute path to test package
	absTestPkg, err := filepath.Abs(testPkg)
	if err != nil {
		t.Fatal(err)
	}

	// Set test arguments
	os.Args = []string{"llgen", absTestPkg}

	// Run main
	main()

	// Check if the output file exists
	outputFile := filepath.Join(testPkg, "llgo_autogen.ll")
	log.Printf("Generated file: %s", filepath.Join(absTestPkg, "llgo_autogen.ll"))
	if _, err = os.Stat(outputFile); err != nil {
		t.Fatalf("Generated file should exist: %v", err)
	}

	// Read and verify file content
	content, err := os.ReadFile(outputFile)
	if err != nil {
		t.Fatalf("Should be able to read generated file: %v", err)
	}
	if !strings.Contains(string(content), "define") {
		t.Error("Generated file should contain LLVM IR code")
	}
}

func TestSelectPhase(t *testing.T) {
	tests := []struct {
		name     string
		phase    string
		phaseSet bool
		abi      int
		abiSet   bool
		want     llgen.Phase
		wantErr  string
	}{
		{name: "default", phase: "pre-abi", want: llgen.PhasePreABI},
		{name: "phase pre ABI", phase: "pre-abi", phaseSet: true, want: llgen.PhasePreABI},
		{name: "phase post ABI", phase: "post-abi", phaseSet: true, want: llgen.PhasePostABI},
		{name: "legacy ABI zero", phase: "pre-abi", abi: 0, abiSet: true, want: llgen.PhasePreABI},
		{name: "legacy ABI two", phase: "pre-abi", abi: 2, abiSet: true, want: llgen.PhasePostABI},
		{name: "removed ABI one", phase: "pre-abi", abi: 1, abiSet: true, wantErr: "invalid -abi=1"},
		{name: "invalid phase", phase: "middle", phaseSet: true, wantErr: "invalid -phase"},
		{name: "conflict", phase: "post-abi", phaseSet: true, abi: 2, abiSet: true, wantErr: "cannot be used together"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := selectPhase(test.phase, test.phaseSet, test.abi, test.abiSet)
			if test.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), test.wantErr) {
					t.Fatalf("selectPhase error = %v, want %q", err, test.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if got != test.want {
				t.Fatalf("selectPhase = %q, want %q", got, test.want)
			}
		})
	}
}
