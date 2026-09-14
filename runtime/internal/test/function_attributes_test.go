package test

import "testing"

//go:noinline
//llgo:cold
func rareAttributeValue() int { return 42 }

//go:noinline
//llgo:cold
//llgo:noreturn
func fatalAttributeValue() { panic("attribute panic") }

func recoverAttributePanic() (value any) {
	defer func() { value = recover() }()
	fatalAttributeValue()
	return "unexpected normal return"
}

func TestFunctionAttributes(t *testing.T) {
	if got := rareAttributeValue(); got != 42 {
		t.Fatalf("cold function returned %d", got)
	}
	if got := recoverAttributePanic(); got != "attribute panic" {
		t.Fatalf("noreturn panic/recover: %v", got)
	}
}
