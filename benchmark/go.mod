module github.com/xgo-dev/llgo/benchmark

go 1.27.0

require github.com/goplus/lib v0.7.1

replace github.com/xgo-dev/llgo => ..

replace github.com/xgo-dev/llgo/cmd => ../cmd

replace github.com/xgo-dev/llgo/runtime => ../runtime

// v0.51.0 plus the generic-method wrapper fix; remove when upstream includes it.
replace golang.org/x/tools => github.com/xgo-dev/tools v0.51.1
