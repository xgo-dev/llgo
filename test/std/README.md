# Standard Library Compatibility Tests

This directory contains compatibility tests for the Go standard library on llgo. The tests run with real Go 1.20 through Go 1.26 toolchains and matching module versions, so release tags and available standard-library APIs reflect the version being checked. Go 1.25 and Go 1.26 receive full coverage on Linux and macOS; each earlier version runs a representative, resource-bounded package set on one platform.

## Directory Structure

```
test/std/
├── README.md           # This file
├── math/
│   ├── math_test.go         # Core math tests (no build tags)
│   ├── math_go124_test.go   # Go 1.24+ specific tests
│   ├── math_bench_test.go   # Performance benchmarks
│   └── testdata/            # Test fixtures (if needed)
└── ... other packages
```

## Test Organization

### Package Structure

Each standard library package should have its own subdirectory under `test/std/`. For example:
- `test/std/math/` - tests for the `math` package
- `test/std/strings/` - tests for the `strings` package
- `test/std/io/` - tests for the `io` package

### File Naming Conventions

- **`<package>_test.go`**: Core tests that run on both `go test` and `llgo test`. These files should NOT have `//go:build llgo` tags so they validate compatibility with standard Go.

- **`<package>_go1XX_test.go`**: Version-specific tests for Go 1.XX+ APIs. Use build tags like `//go:build go1.24` to gate features only available in newer Go versions.

- **`<package>_bench_test.go`**: Benchmarks for performance-sensitive functions. At least one benchmark is recommended per package.

- **`testdata/`**: Directory for test fixtures, golden files, or other test data.

## Build Tags

### Standard Tests (no tags)

Core functionality tests should NOT use `//go:build llgo` tags:

```go
package math_test

import (
	"math"
	"testing"
)

func TestSqrt(t *testing.T) {
	result := math.Sqrt(4.0)
	if result != 2.0 {
		t.Errorf("Sqrt(4.0) = %v, want 2.0", result)
	}
}
```

This ensures both `go test` and `llgo test` exercise the same test suite.

### Version-Specific Tests

For APIs introduced in Go 1.24 or later:

```go
//go:build go1.24
// +build go1.24

package math_test

import "testing"

func TestNewGo124API(t *testing.T) {
	// Test Go 1.24+ specific functionality
}
```

### llgo-Specific Tests

Tests that require llgo-specific runtime features belong in `test/c/` (and other language-specific directories as they are added), not here. Those tests use `//go:build llgo` tags.

## Documenting Unsupported Features

If llgo doesn't yet support a feature, use `t.Skip()` with a TODO identifier:

```go
func TestUnsupportedFeature(t *testing.T) {
	t.Skip("TODO(issue-1234): implement feature X in llgo runtime")
	// Test code here
}
```

This allows:
1. Tests to be written before full implementation
2. Clear tracking of what's missing
3. Easy identification of work items for contributors

## Adding Benchmarks

Performance-sensitive packages should include at least one benchmark:

```go
func BenchmarkSqrt(b *testing.B) {
	x := 2.0
	for i := 0; i < b.N; i++ {
		math.Sqrt(x)
	}
}
```

Run benchmarks with:
```bash
go test -bench=. ./test/std/math/
# or with llgo
llgo test -bench=. ./test/std/math/
```

## Running Tests

### Standard Go

```bash
# Run all stdlib compatibility tests
go test ./test/std/...

# Run specific package tests
go test ./test/std/math/

# Run tests in short mode
go test -short ./test/std/...

# Run benchmarks
go test -bench=. ./test/std/math/
```

### llgo

```bash
# Run all tests (requires LLGO_ROOT environment variable)
./dev/llgo.sh test ./test/...

# Or with installed llgo
llgo test ./test/...

# Run specific package
llgo test ./test/std/math/

# Run with an exact older Go toolchain and matching release tags
dev/test_go_version.sh 1.20 ./test/std/math
```

## Contributing New Package Tests

1. **Create package directory**: `mkdir test/std/<package>`

2. **Write core tests**: Create `<package>_test.go` with functional test cases

3. **Add benchmarks**: Create `<package>_bench_test.go` with at least one benchmark

4. **Handle version differences**: If needed, create `<package>_go1XX_test.go` files

5. **Document gaps**: Use `t.Skip("TODO: ...")` for unsupported features

6. **Update tracking**: Add entry to `test/std/TODO.md`

7. **Validate**: Ensure tests pass with both `go test` and `llgo test`

## Test Coverage Goals

Focus on:
- **Core functionality**: Cover main API surface area
- **Edge cases**: Boundary conditions, special values (NaN, Inf, zero)
- **Error conditions**: Invalid inputs, error paths
- **Performance**: Key hot-path functions
- **Compatibility**: Behavior matches standard Go across versions

## Related Directories

- **`test/c/`**: llgo-specific C interop tests (uses `//go:build llgo`)
- **`_cmptest/`**: Comparison tests ensuring Go/llgo output equivalence

The `test/std/` suite focuses on validating standard library API conformance, while `_cmptest/` validates behavioral equivalence through end-to-end comparison.
