//go:build !linux

package pclnpost

func lockExecutableWrite() func() { return func() {} }
