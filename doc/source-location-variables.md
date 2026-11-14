# Source location variables

LLGo supports compiler-provided variables for the C `__FILE__` and `__LINE__`
macros. Bind any package-level variable with the corresponding link name:

```go
package location

import _ "unsafe"

//go:linkname File llgo.file
var File *int8

//go:linkname Line llgo.line
var Line int32
```

Aliases and named types with the same underlying representation are accepted,
including `*c.Char` and `c.Int`. The Go names are arbitrary: the compiler uses the
link names. The current `github.com/goplus/lib/c` package already declares a
`FILE` type, so it cannot also declare a variable named `FILE` without an API
change in that library.

Each read produces the location of that expression. `Line` has C `int` width
(`int32`), including on 64-bit targets. `File` points to a private, read-only,
NUL-terminated constant; identical filenames share storage within each LLVM
module. Neither declaration emits a global variable or an external symbol.

The location honors `//line` directives. A generic function reports the position
in its body; a wrapper reports the position inside the wrapper. Package-level
initializers and closures are supported. Without a `//line` override, the
filename is the path supplied by the loader, normally absolute.

Assignments, compound assignments, increment/decrement, range assignment, and
taking a compiler variable's address are errors, including in unreachable code
and uninstantiated generic functions. Copying its value into another variable
is allowed. Ordinary Go constant-expression rules apply: these values cannot
be used in `const` declarations or array lengths. Expression switch cases are
ordinary expressions in Go and may read them.

Unknown `llgo.*` variable links are rejected. The proposed Phase 2 bindings
`llgo.func`, `llgo.date`, and `llgo.time` are not implemented. LLGo's command line
still rejects `-trimpath`; this feature does not add artifact-wide path trimming.

## Runnable examples

The [C logging example](../_demo/c/sourcelocation/main.go) declares bindings using
`*c.Char` and `c.Int`, passes them to a logging helper at the call site, and checks
both ordinary and `//line`-adjusted locations. Run it from the repository root:

```sh
(cd _demo/c && llgo run ./sourcelocation)
llgo test -v ./test/llgoext/sourcelocation
llgo test -O0 -v ./test/llgoext/sourcelocation
```

The [runtime tests](../test/llgoext/sourcelocation/location_test.go) import reusable
bindings from another package and assert individual read positions, filename
pooling, C string compatibility, initializers, wrapper bodies, generic bodies,
and closures. They run in the existing native `llgo test ./test/...` CI suite;
the logging example runs in the regular demo suite.

## Compiler boundary

`internal/directive` collects plain variable declaration records and validates
checked uses before typed rewriting can erase illegal address-taking. Records
are scoped to a `types.Package` instance; an unannotated declaration in another
source view does not inherit a compatibility link. Effective patch records keep
unreplaced original declarations; each replacement supplies its own record,
including the absence of a link. Checked-use validation runs only for packages
that declare compiler variables or directly import a package that does.

`cl` queries the prepared kind and passes the read instruction's adjusted
`token.Position` to `ssa`. `ssa` produces the typed constant and reuses its
module's string pool. Neither lowering layer parses AST for this feature.
