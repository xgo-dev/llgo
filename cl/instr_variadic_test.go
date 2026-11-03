package cl

import (
	"debug/dwarf"
	"regexp"
	"strings"
	"testing"

	llssa "github.com/xgo-dev/llgo/ssa"
)

func TestIndirectCVariadicArguments(t *testing.T) {
	const source = `package foo
//llgo:cdecl
type Variadic func(marker int32, __llgo_va_list ...any) int32
//llgo:cdecl
type Fixed func(marker int32, values ...any) int32
func NativeFixed(function Fixed, marker int32, values []any) int32 {
	return function(marker, values...)
}
func Call(function Variadic, marker int32, integer int32, floating float64) int32 {
	return function(marker, integer, floating)
}
func Empty(function Variadic, marker int32) int32 { return function(marker) }
func Ordinary(function func(marker int32, values ...any) int32, marker int32, integer int32) int32 {
	return function(marker, integer)
}
`
	for _, order := range []string{"c-first", "go-first"} {
		orderedSource := source
		names := map[string]string{"Call": "Call", "Empty": "Empty", "Ordinary": "Ordinary", "NativeFixed": "NativeFixed"}
		if order == "go-first" {
			// Package lowering sorts member names. Make the ordinary Go callback
			// populate the shared type cache before either native callback role.
			orderedSource = strings.NewReplacer("func Call(", "func ZCall(",
				"func Empty(", "func ZEmpty(", "func Ordinary(", "func AOrdinary(",
				"func NativeFixed(", "func YNativeFixed(").Replace(source)
			names = map[string]string{"Call": "ZCall", "Empty": "ZEmpty", "Ordinary": "AOrdinary", "NativeFixed": "YNativeFixed"}
		}
		ssaPkg, _, files := buildGoSSAPkg(t, orderedSource)
		for _, target := range []llssa.Target{
			{GOOS: "linux", GOARCH: "amd64"},
			{GOOS: "darwin", GOARCH: "arm64"},
		} {
			t.Run(order+"/"+target.GOOS+"/"+target.GOARCH, func(t *testing.T) {
				prog := newLLSSAProgForTarget(t, &target)
				defer prog.Dispose()
				pkg, _, err := newPackageEx(prog, nil, nil, nil, ssaPkg, files, nil, false, Options{Debug: true, DebugSymbols: true})
				if err != nil {
					t.Fatal(err)
				}
				// C ellipsis arguments must be concrete scalar LLVM values, rather than a
				// Go slice header. Ordinary Go variadic function values keep their slice ABI.
				// Ignore target-specific attributes on the closure environment argument.
				for _, test := range []struct {
					name    string
					pattern string
				}{
					{"Call", `call i32 \(i32, \.\.\.\) %[^ (]+\(i32 %[^,)\n]+, i32 %[^,)\n]+, double %[^,)\n]+\)`},
					{"Empty", `call i32 \(i32, \.\.\.\) %[^ (]+\(i32 %[^,)\n]+\)`},
					{"Ordinary", `call i32 %[^ (]+\(ptr [^,)\n]+, i32 %[^,)\n]+, %"[^"]*runtime\.Slice" %[^,)\n]+\)`},
					{"NativeFixed", `call i32 %[^ (]+\(i32 %[^,)\n]+, %"[^"]*runtime\.Slice" %[^,)\n]+\)`},
				} {
					t.Run(test.name, func(t *testing.T) {
						text := mustNamedFunction(t, pkg.Module(), "foo."+names[test.name]).String()
						if !regexp.MustCompile(test.pattern).MatchString(text) {
							t.Fatalf("missing expected call ABI:\n%s", text)
						}
					})
				}
				data := readDebugObject(t, prog, pkg.Module(), target.GOOS)
				for _, test := range []struct {
					name     string
					variadic bool
				}{
					{"foo." + names["Call"], true},
					{"foo." + names["Ordinary"], false},
					{"foo." + names["NativeFixed"], false},
				} {
					reader := data.Reader()
					function, found := "", false
					for {
						entry, err := reader.Next()
						if err != nil {
							t.Fatal(err)
						}
						if entry == nil {
							break
						}
						if entry.Tag == dwarf.TagSubprogram {
							function, _ = entry.Val(dwarf.AttrName).(string)
						}
						if function != test.name || entry.Tag != dwarf.TagFormalParameter || entry.Val(dwarf.AttrName) != "function" {
							continue
						}
						offset, ok := entry.Val(dwarf.AttrType).(dwarf.Offset)
						if !ok {
							t.Fatal("callback parameter has no type")
						}
						typ, err := data.Type(offset)
						if err != nil {
							t.Fatal(err)
						}
					unwrap:
						for {
							switch current := typ.(type) {
							case *dwarf.TypedefType:
								typ = current.Type
							case *dwarf.PtrType:
								typ = current.Type
							case *dwarf.StructType:
								// A Go function value points to its code/environment
								// descriptor; native C callbacks already point to code.
								var code dwarf.Type
								for _, field := range current.Field {
									if field.Name == "code" {
										code = field.Type
									}
								}
								if code == nil {
									t.Fatalf("callback %s has no code pointer: %s", test.name, typ)
								}
								typ = code
							default:
								break unwrap
							}
						}
						signature, ok := typ.(*dwarf.FuncType)
						if !ok || len(signature.ParamType) != 2 {
							t.Fatalf("callback %s has unexpected signature %s", test.name, typ)
						}
						_, variadic := signature.ParamType[1].(*dwarf.DotDotDotType)
						if variadic != test.variadic {
							t.Errorf("callback %s variadic = %v, want %v: %s", test.name, variadic, test.variadic, signature)
						}
						found = true
					}
					if !found {
						t.Errorf("missing callback parameter for %s", test.name)
					}
				}
			})
		}
	}
}
