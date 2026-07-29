package safepointplan

import (
	"go/ast"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"testing"

	"golang.org/x/tools/go/ssa"
	"golang.org/x/tools/go/ssa/ssautil"
)

func TestBackedges(t *testing.T) {
	tests := []struct {
		name string
		src  string
		want int
	}{
		{
			name: "straight line",
			src:  `package p; func f(n int) int { return n + 1 }`,
		},
		{
			name: "loop",
			src:  `package p; func f(n int) { for n > 0 { n-- } }`,
			want: 1,
		},
		{
			name: "nested loops",
			src:  `package p; func f(n int) { for i := 0; i < n; i++ { for j := 0; j < n; j++ {} } }`,
			want: 2,
		},
		{
			name: "irreducible loop",
			src: `package p
func f(n int) {
	if n > 0 { goto left }
right:
	n--
	if n > 0 { goto left }
	return
left:
	n--
	if n > 0 { goto right }
}`,
			want: 1,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			fn := buildFunction(t, test.src)
			polls := Backedges(fn)
			if len(polls) != test.want {
				t.Fatalf("Backedges returned %d polls, want %d", len(polls), test.want)
			}
			for instr := range polls {
				switch instr.(type) {
				case *ssa.If, *ssa.Jump:
				default:
					t.Errorf("poll instruction is %T, want a block terminator", instr)
				}
			}
		})
	}
}

func TestBackedgesNil(t *testing.T) {
	if got := Backedges(nil); got != nil {
		t.Fatalf("Backedges(nil) = %v, want nil", got)
	}
}

func buildFunction(t *testing.T, src string) *ssa.Function {
	t.Helper()
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "p.go", src, 0)
	if err != nil {
		t.Fatal(err)
	}
	pkg, _, err := ssautil.BuildPackage(
		&types.Config{Importer: importer.Default()},
		fset,
		types.NewPackage("p", "p"),
		[]*ast.File{file},
		ssa.InstantiateGenerics,
	)
	if err != nil {
		t.Fatal(err)
	}
	return pkg.Func("f")
}
