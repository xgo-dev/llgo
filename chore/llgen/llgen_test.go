package main

import (
	"log"
	"os"
	"path/filepath"
	"strings"
	"testing"
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
