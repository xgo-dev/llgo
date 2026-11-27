package main

import (
	"os"
	"testing"
)

func TestZoo(t *testing.T) {
	if Zoo() != 3 {
		t.Fatal("Zoo() != 3")
	}
}

func TestFalse(t *testing.T) {
	// t.Fatal("false")
}

func TestSwitchWd(t *testing.T) {
	_, err := os.Stat("./foo")
	if err != nil {
		t.Fatal(err)
	}
}
