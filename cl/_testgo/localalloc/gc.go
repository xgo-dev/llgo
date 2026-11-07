//go:build !baremetal

package main

import "runtime"

func collectLocalAlloc() { runtime.GC() }
