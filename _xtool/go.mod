module github.com/xgo-dev/llgo/_xtool

go 1.23

require (
	github.com/goplus/lib v0.5.3
	github.com/llarhub/clang-c v0.6.1
)

// Explicit native bindings from goplus/lib#53; remove after its release.
replace github.com/goplus/lib => github.com/cpunion/lib v0.0.0-20261021012937-cbe6d613b5f0
