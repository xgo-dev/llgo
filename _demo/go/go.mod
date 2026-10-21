module github.com/xgo-dev/llgo/_demo/go

go 1.23

require github.com/goplus/lib v0.5.1

// Explicit native variable bindings from goplus/lib#53; remove after its release.
replace github.com/goplus/lib => github.com/cpunion/lib v0.0.0-20261021003434-95e56e9fa51c
