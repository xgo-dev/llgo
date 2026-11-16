package reflectnames

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/xgo-dev/llvm"
)

const callbackIR = `
target datalayout = "e-p:64:64-i64:64-n32:64-S128"
%String = type {ptr, i64}
%Slice = type {ptr, i64, i64}
@a = private constant [12 x i8] c"BeforeCreate"
@b = private constant [9 x i8] c"AfterFind"
@backing = global [2 x %String] [%String {ptr @a, i64 12}, %String {ptr @b, i64 9}]
@names = global %Slice {ptr @backing, i64 2, i64 2}
declare void @lookup(%String)
declare void @escape(ptr)
define void @probe(i64 %index) {
entry:
  %base = load ptr, ptr @names
  %slot = getelementptr inbounds %String, ptr %base, i64 %index
  %ptr = load ptr, ptr %slot
  %len.addr = getelementptr inbounds i8, ptr %slot, i64 8
  %len = load i64, ptr %len.addr
  %s0 = insertvalue %String poison, ptr %ptr, 0
  %s1 = insertvalue %String %s0, i64 %len, 1
  call void @lookup(%String "llgo.reflect.methodbyname.name" %s1)
  ret void
}
`

func parse(t *testing.T, text string) llvm.Module {
	t.Helper()
	ctx := llvm.NewContext()
	t.Cleanup(ctx.Dispose)
	path := filepath.Join(t.TempDir(), "input.ll")
	if err := os.WriteFile(path, []byte(text), 0o600); err != nil {
		t.Fatal(err)
	}
	buffer, err := llvm.NewMemoryBufferFromFile(path)
	if err != nil {
		t.Fatal(err)
	}
	mod, err := ctx.ParseIR(buffer)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(mod.Dispose)
	if err := llvm.VerifyModule(mod, llvm.ReturnStatusAction); err != nil {
		t.Fatal(err)
	}
	return mod
}

func lookupCall(t *testing.T, mod llvm.Module) llvm.Value {
	t.Helper()
	fn := mod.NamedFunction("probe")
	for bb := fn.FirstBasicBlock(); !bb.IsNil(); bb = llvm.NextBasicBlock(bb) {
		for v := bb.FirstInstruction(); !v.IsNil(); v = llvm.NextInstruction(v) {
			if opcode(v) == llvm.Call && v.CalledValue().Name() == "lookup" {
				return v
			}
		}
	}
	t.Fatal("missing lookup call")
	return llvm.Value{}
}

func TestReadonlyCallbackArray(t *testing.T) {
	for _, tc := range []struct {
		name, text   string
		closed, want bool
	}{
		{"closed mutable package globals", callbackIR, true, true},
		{"external ownership unknown", callbackIR, false, false},
		{"internal globals", strings.ReplaceAll(callbackIR, " = global ", " = internal global "), false, true},
		{"mutated slice header", strings.Replace(callbackIR, "  ret void", "  store ptr null, ptr @names\n  ret void", 1), true, false},
		{"mutated backing through header", strings.Replace(callbackIR, "  ret void", "  store ptr null, ptr %slot\n  ret void", 1), true, false},
		{"escaping backing pointer", strings.Replace(callbackIR, "  ret void", "  call void @escape(ptr %base)\n  ret void", 1), true, false},
		{"escaping header pointer", strings.Replace(callbackIR, "  ret void", "  call void @escape(ptr @names)\n  ret void", 1), true, false},
		{"non-inbounds dynamic access", strings.Replace(callbackIR, "getelementptr inbounds %String", "getelementptr %String", 1), true, false},
		{"volatile pointer load", strings.Replace(callbackIR, "%ptr = load ptr", "%ptr = load volatile ptr", 1), true, false},
		{"volatile header load", strings.Replace(callbackIR, "%base = load ptr", "%base = load volatile ptr", 1), true, false},
		{"uncorrelated length", strings.Replace(callbackIR, "i8, ptr %slot, i64 8", "i8, ptr %base, i64 8", 1), true, false},
		{"length beyond literal", strings.Replace(callbackIR, "ptr @a, i64 12", "ptr @a, i64 13", 1), true, false},
		{"no call marker", strings.Replace(callbackIR, ` "llgo.reflect.methodbyname.name"`, "", 1), true, false},
		{"externally initialized", strings.Replace(callbackIR, "@names = global", "@names = externally_initialized global", 1), true, false},
		{"externally initialized quoted name", strings.ReplaceAll(strings.Replace(callbackIR, "@names = global", "@names = externally_initialized global", 1), "@names", `@" = global fake"`), true, false},
		{"global alias", strings.Replace(callbackIR, "declare void @lookup", "@alias = alias [2 x %String], ptr @backing\ndeclare void @lookup", 1), true, false},
		{"nested address holder mutation", strings.Replace(strings.Replace(callbackIR, "declare void @lookup", "@holder = global ptr @names\ndeclare void @lookup", 1), "  ret void", "  %header = load ptr, ptr @holder\n  %array = load ptr, ptr %header\n  store ptr null, ptr %array\n  ret void", 1), true, false},
		{"integer alias mutation", strings.Replace(callbackIR, "  ret void", "  %bits = load i64, ptr @names\n  %alias = inttoptr i64 %bits to ptr\n  store ptr null, ptr %alias\n  ret void", 1), true, false},
		{"mutable string data", strings.Replace(strings.Replace(callbackIR, "@a = private constant", "@a = private global", 1), "  ret void", "  store i8 0, ptr %ptr\n  ret void", 1), true, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			mod := parse(t, tc.text)
			var closed func(llvm.Value) bool
			if tc.closed {
				closed = func(llvm.Value) bool { return true }
			}
			names, ok := ResolveCall(mod, lookupCall(t, mod), closed)
			if ok != tc.want {
				t.Fatalf("ResolveCall = %v, %v; want success %v", names, ok, tc.want)
			}
			if ok && !reflect.DeepEqual(names, []string{"AfterFind", "BeforeCreate"}) {
				t.Fatalf("names = %q", names)
			}
		})
	}
}

func TestStringPairs(t *testing.T) {
	const prefix = `target datalayout = "e-p:64:64-i64:64-n32:64-S128"
@a = private constant [4 x i8] c"Keep"
@b = private constant [4 x i8] c"Drop"
declare void @lookup(ptr, i64)
define void @probe(i1 %which) {
entry:
`
	for _, tc := range []struct {
		name, body string
		want       []string
	}{
		{"constant", `call void @lookup(ptr "llgo.reflect.methodbyname.name" @a, i64 4)`, []string{"Keep"}},
		{"substring", `call void @lookup(ptr "llgo.reflect.methodbyname.name" getelementptr (i8, ptr @a, i64 1), i64 3)`, []string{"eep"}},
		{"paired select", `%p = select i1 %which, ptr @a, ptr @b
%n = select i1 %which, i64 4, i64 2
call void @lookup(ptr "llgo.reflect.methodbyname.name" %p, i64 %n)`, []string{"Dr", "Keep"}},
		{"negative length", `call void @lookup(ptr "llgo.reflect.methodbyname.name" @a, i64 -1)`, nil},
		{"beyond literal", `call void @lookup(ptr "llgo.reflect.methodbyname.name" @a, i64 5)`, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			mod := parse(t, prefix+tc.body+"\nret void\n}\n")
			got, ok := ResolveCall(mod, lookupCall(t, mod), nil)
			if ok != (tc.want != nil) || !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("got %q, %v; want %q", got, ok, tc.want)
			}
		})
	}
}

func TestRejectDynamicSubarray(t *testing.T) {
	text := strings.Replace(callbackIR, "@backing = global [2 x %String] [%String {ptr @a, i64 12}, %String {ptr @b, i64 9}]", "@backing = global [1 x [2 x %String]] [[2 x %String] [%String {ptr @a, i64 12}, %String {ptr @b, i64 9}]]", 1)
	mod := parse(t, text)
	if names, ok := ResolveCall(mod, lookupCall(t, mod), func(llvm.Value) bool { return true }); ok {
		t.Fatalf("accepted unproved subarray: %q", names)
	}
}

func TestChoiceLimit(t *testing.T) {
	values := make([]string, maxChoices+1)
	for i := range values {
		values[i] = "%String {ptr @a, i64 12}"
	}
	text := strings.Replace(callbackIR, "[2 x %String] [%String {ptr @a, i64 12}, %String {ptr @b, i64 9}]", "[33 x %String] ["+strings.Join(values, ", ")+"]", 1)
	mod := parse(t, text)
	if names, ok := ResolveCall(mod, lookupCall(t, mod), func(llvm.Value) bool { return true }); ok {
		t.Fatalf("accepted unbounded choice count: %q", names)
	}
}

func TestInBoundsModifierCannotComeFromSymbolName(t *testing.T) {
	text := strings.Replace(callbackIR, "getelementptr inbounds %String", "getelementptr %String", 1)
	text = strings.ReplaceAll(text, "@backing", `@"getelementptr inbounds backing"`)
	text = strings.ReplaceAll(text, "%slot", `%" = getelementptr inbounds fake"`)
	mod := parse(t, text)
	if names, ok := ResolveCall(mod, lookupCall(t, mod), func(llvm.Value) bool { return true }); ok {
		t.Fatalf("read inbounds from a symbol name: %q", names)
	}
}
