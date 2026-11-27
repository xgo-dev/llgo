package lib

import _ "unsafe"

const LLGoPackage = "decl"

type Text struct{ Bytes [6]byte }
type Value = Text
type String = *Value
type Chain = String

// llgo:link String.Compare C.strcmp
func (String) Compare(other *byte) int32 { return -1 }

//llgo:link (Chain).CompareChain C.strcmp
func (Chain) CompareChain(other *byte) int32 { return -1 }

type Number int32
type Int = Number

// llgo:link Int.Abs C.abs
func (Int) Abs() int32 { return -1 }
