package cl

import (
	"debug/dwarf"
	"regexp"
	"testing"

	llssa "github.com/xgo-dev/llgo/ssa"
)

func TestIndirectCVariadicArguments(t *testing.T) {
	const source = `package foo
//llgo:cdecl
type Variadic func(marker int32, __llgo_va_list ...any) int32
func Call(function Variadic, marker int32, integer int32, floating float64) int32 {
	return function(marker, integer, floating)
}
func Empty(function Variadic, marker int32) int32 { return function(marker) }
func Ordinary(function func(marker int32, values ...any) int32, marker int32, integer int32) int32 {
	return function(marker, integer)
}
`
	ssaPkg, _, files := buildGoSSAPkg(t, source)
	for _, target := range []llssa.Target{
		{GOOS: "linux", GOARCH: "amd64"},
		{GOOS: "darwin", GOARCH: "arm64"},
	} {
		t.Run(target.GOOS+"/"+target.GOARCH, func(t *testing.T) {
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
				{"Ordinary", `call i32 \(ptr, i32, \.\.\.\) %[^ (]+\(ptr [^,)\n]+, i32 %[^,)\n]+, %"[^"]*runtime\.Slice" %[^,)\n]+\)`},
			} {
				t.Run(test.name, func(t *testing.T) {
					text := mustNamedFunction(t, pkg.Module(), "foo."+test.name).String()
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
				{"foo.Call", true},
				{"foo.Ordinary", false},
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
							// Ordinary Go function values store their code pointer
							// in $f; native C callbacks are already pointers.
							var code dwarf.Type
							for _, field := range current.Field {
								if field.Name == "$f" {
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
