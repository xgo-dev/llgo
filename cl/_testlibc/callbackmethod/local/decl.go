package local

import (
	"structs"
	"unsafe"
)

const (
	LLGoFiles   = "_wrap/visit.c"
	LLGoPackage = "link"
)

type Node struct {
	_    structs.HostLayout
	Kind int32
	Data int32
}

// llgo:cdecl
type Visitor = func(Node, unsafe.Pointer) int32
type VisitorAlias = Visitor

//go:linkname VisitNode C.llgo_visit_node
func VisitNode(node Node, visitor Visitor, data unsafe.Pointer) int32

// llgo:link Node.Visit C.llgo_visit_node
func (n Node) Visit(visitor VisitorAlias, data unsafe.Pointer) int32 { return -1 }
