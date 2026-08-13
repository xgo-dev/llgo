// LITTEST
package main

import (
	"go/ast"
	"iter"
	"math"
)

func main() {
	c := &Cursor{in: &Inspector{}}
	_ = c
}

type Cursor struct {
	in    *Inspector
	index int32 // index of push node; -1 for virtual root node
}

func (c Cursor) Node() ast.Node {
	if c.index < 0 {
		return nil
	}
	return c.in.events[c.index].node
}

// CHECK-LABEL: define { %main.Cursor, i1 } @main.Cursor.FindNode(%main.Cursor %0, %"{{.*}}/runtime/internal/runtime.iface" %1){{.*}} {
// CHECK: store %main.Cursor %0, ptr [[FN_CURSOR_SLOT:%[0-9]+]]
// CHECK: store %"{{.*}}/runtime/internal/runtime.iface" %1, ptr [[FN_NODE_SLOT:%[0-9]+]]
// CHECK: br i1 false,
// CHECK: [[FN_MASK:%[0-9]+]] = call i64 @main.maskOf(%"{{.*}}/runtime/internal/runtime.Slice" %{{[0-9]+}})
// CHECK: [[FN_EVENTS:%[0-9]+]] = load %"{{.*}}/runtime/internal/runtime.Slice", ptr %{{[0-9]+}}
// CHECK: [[FN_CURSOR_VALUE:%[0-9]+]] = load %main.Cursor, ptr [[FN_CURSOR_SLOT]]
// CHECK-NEXT: [[FN_RANGE:%[0-9]+]] = call { i32, i32 } @main.Cursor.indices(%main.Cursor [[FN_CURSOR_VALUE]])
// CHECK: [[FN_START:%[0-9]+]] = extractvalue { i32, i32 } [[FN_RANGE]], 0
// CHECK: [[FN_LIMIT:%[0-9]+]] = extractvalue { i32, i32 } [[FN_RANGE]], 1
// CHECK: [[FN_I:%[0-9]+]] = phi i32 [ [[FN_START]],
// CHECK: [[FN_MORE:%[0-9]+]] = icmp slt i32 [[FN_I]], [[FN_LIMIT]]
// CHECK: [[FN_EVENT_PTR:%[0-9]+]] = getelementptr inbounds %main.event, ptr %{{[0-9]+}}, i64 %{{[0-9]+}}
// CHECK: [[FN_EVENT:%[0-9]+]] = load %main.event, ptr [[FN_EVENT_PTR]]
// CHECK: [[FN_PAIR_INDEX:%[0-9]+]] = load i32, ptr %{{[0-9]+}}
// CHECK: [[FN_PUSH:%[0-9]+]] = icmp sgt i32 [[FN_PAIR_INDEX]], [[FN_I]]
// CHECK: [[FN_EVENT_TYPE:%[0-9]+]] = load i64, ptr %{{[0-9]+}}
// CHECK: [[FN_TYPE_MATCH:%[0-9]+]] = and i64 [[FN_EVENT_TYPE]], [[FN_MASK]]
// CHECK: [[FN_MAY_MATCH:%[0-9]+]] = icmp ne i64 [[FN_TYPE_MATCH]], 0
// CHECK: store i32 [[FN_I]], ptr %{{[0-9]+}}
// CHECK: store i1 true, ptr %{{[0-9]+}}
// CHECK: [[FN_POP:%[0-9]+]] = load i32, ptr %{{[0-9]+}}
// CHECK: [[FN_POP_TYPE:%[0-9]+]] = load i64, ptr %{{[0-9]+}}
// CHECK: [[FN_POP_MASK:%[0-9]+]] = and i64 [[FN_POP_TYPE]], [[FN_MASK]]
// CHECK: [[FN_SKIP:%[0-9]+]] = icmp eq i64 [[FN_POP_MASK]], 0
// CHECK: [[FN_NODE_EQUAL:%[0-9]+]] = call i1 @"{{.*}}/runtime/internal/runtime.EfaceEqual"

func (c Cursor) FindNode(n ast.Node) (Cursor, bool) {

	// FindNode is equivalent to this code,
	// but more convenient and 15-20% faster:
	if false {
		for candidate := range c.Preorder(n) {
			if candidate.Node() == n {
				return candidate, true
			}
		}
		return Cursor{}, false
	}

	// TODO(adonovan): opt: should we assume Node.Pos is accurate
	// and combine type-based filtering with position filtering
	// like FindByPos?

	mask := maskOf([]ast.Node{n})
	events := c.in.events

	for i, limit := c.indices(); i < limit; i++ {
		ev := events[i]
		if ev.index > i { // push?
			if ev.typ&mask != 0 && ev.node == n {
				return Cursor{c.in, i}, true
			}
			pop := ev.index
			if events[pop].typ&mask == 0 {
				// Subtree does not contain type of n: skip.
				i = pop
			}
		}
	}
	return Cursor{}, false
}

type event struct {
	node   ast.Node
	typ    uint64 // typeOf(node) on push event, or union of typ strictly between push and pop events on pop events
	index  int32  // index of corresponding push or pop event
	parent int32  // index of parent's push node (push nodes only), or packed edge kind/index (pop nodes only)
}

type Inspector struct {
	events []event
}

func maskOf(nodes []ast.Node) uint64 {
	if len(nodes) == 0 {
		return math.MaxUint64 // match all node types
	}
	var mask uint64
	for _, n := range nodes {
		mask |= typeOf(n)
	}
	return mask
}

// indices return the [start, end) half-open interval of event indices.
func (c Cursor) indices() (int32, int32) {
	if c.index < 0 {
		return 0, int32(len(c.in.events)) // root: all events
	} else {
		return c.index, c.in.events[c.index].index + 1 // just one subtree
	}
}

func (c Cursor) Preorder(types ...ast.Node) iter.Seq[Cursor] {
	mask := maskOf(types)

	return func(yield func(Cursor) bool) {
		events := c.in.events

		for i, limit := c.indices(); i < limit; {
			ev := events[i]
			if ev.index > i { // push?
				if ev.typ&mask != 0 && !yield(Cursor{c.in, i}) {
					break
				}
				pop := ev.index
				if events[pop].typ&mask == 0 {
					// Subtree does not contain types: skip.
					i = pop + 1
					continue
				}
			}
			i++
		}
	}
}

func typeOf(n ast.Node) uint64 {
	// Fast path: nearly half of all nodes are identifiers.
	if _, ok := n.(*ast.Ident); ok {
		return 1 << nIdent
	}

	// These cases include all nodes encountered by ast.Inspect.
	switch n.(type) {
	case *ast.ArrayType:
		return 1 << nArrayType
	case *ast.AssignStmt:
		return 1 << nAssignStmt
	case *ast.BadDecl:
		return 1 << nBadDecl
	case *ast.BadExpr:
		return 1 << nBadExpr
	case *ast.BadStmt:
		return 1 << nBadStmt
	case *ast.BasicLit:
		return 1 << nBasicLit
	case *ast.BinaryExpr:
		return 1 << nBinaryExpr
	case *ast.BlockStmt:
		return 1 << nBlockStmt
	case *ast.BranchStmt:
		return 1 << nBranchStmt
	case *ast.CallExpr:
		return 1 << nCallExpr
	case *ast.CaseClause:
		return 1 << nCaseClause
	case *ast.ChanType:
		return 1 << nChanType
	case *ast.CommClause:
		return 1 << nCommClause
	case *ast.Comment:
		return 1 << nComment
	case *ast.CommentGroup:
		return 1 << nCommentGroup
	case *ast.CompositeLit:
		return 1 << nCompositeLit
	case *ast.DeclStmt:
		return 1 << nDeclStmt
	case *ast.DeferStmt:
		return 1 << nDeferStmt
	case *ast.Ellipsis:
		return 1 << nEllipsis
	case *ast.EmptyStmt:
		return 1 << nEmptyStmt
	case *ast.ExprStmt:
		return 1 << nExprStmt
	case *ast.Field:
		return 1 << nField
	case *ast.FieldList:
		return 1 << nFieldList
	case *ast.File:
		return 1 << nFile
	case *ast.ForStmt:
		return 1 << nForStmt
	case *ast.FuncDecl:
		return 1 << nFuncDecl
	case *ast.FuncLit:
		return 1 << nFuncLit
	case *ast.FuncType:
		return 1 << nFuncType
	case *ast.GenDecl:
		return 1 << nGenDecl
	case *ast.GoStmt:
		return 1 << nGoStmt
	case *ast.Ident:
		return 1 << nIdent
	case *ast.IfStmt:
		return 1 << nIfStmt
	case *ast.ImportSpec:
		return 1 << nImportSpec
	case *ast.IncDecStmt:
		return 1 << nIncDecStmt
	case *ast.IndexExpr:
		return 1 << nIndexExpr
	case *ast.IndexListExpr:
		return 1 << nIndexListExpr
	case *ast.InterfaceType:
		return 1 << nInterfaceType
	case *ast.KeyValueExpr:
		return 1 << nKeyValueExpr
	case *ast.LabeledStmt:
		return 1 << nLabeledStmt
	case *ast.MapType:
		return 1 << nMapType
	case *ast.Package:
		return 1 << nPackage
	case *ast.ParenExpr:
		return 1 << nParenExpr
	case *ast.RangeStmt:
		return 1 << nRangeStmt
	case *ast.ReturnStmt:
		return 1 << nReturnStmt
	case *ast.SelectStmt:
		return 1 << nSelectStmt
	case *ast.SelectorExpr:
		return 1 << nSelectorExpr
	case *ast.SendStmt:
		return 1 << nSendStmt
	case *ast.SliceExpr:
		return 1 << nSliceExpr
	case *ast.StarExpr:
		return 1 << nStarExpr
	case *ast.StructType:
		return 1 << nStructType
	case *ast.SwitchStmt:
		return 1 << nSwitchStmt
	case *ast.TypeAssertExpr:
		return 1 << nTypeAssertExpr
	case *ast.TypeSpec:
		return 1 << nTypeSpec
	case *ast.TypeSwitchStmt:
		return 1 << nTypeSwitchStmt
	case *ast.UnaryExpr:
		return 1 << nUnaryExpr
	case *ast.ValueSpec:
		return 1 << nValueSpec
	}
	return 0
}

const (
	nArrayType = iota
	nAssignStmt
	nBadDecl
	nBadExpr
	nBadStmt
	nBasicLit
	nBinaryExpr
	nBlockStmt
	nBranchStmt
	nCallExpr
	nCaseClause
	nChanType
	nCommClause
	nComment
	nCommentGroup
	nCompositeLit
	nDeclStmt
	nDeferStmt
	nEllipsis
	nEmptyStmt
	nExprStmt
	nField
	nFieldList
	nFile
	nForStmt
	nFuncDecl
	nFuncLit
	nFuncType
	nGenDecl
	nGoStmt
	nIdent
	nIfStmt
	nImportSpec
	nIncDecStmt
	nIndexExpr
	nIndexListExpr
	nInterfaceType
	nKeyValueExpr
	nLabeledStmt
	nMapType
	nPackage
	nParenExpr
	nRangeStmt
	nReturnStmt
	nSelectStmt
	nSelectorExpr
	nSendStmt
	nSliceExpr
	nStarExpr
	nStructType
	nSwitchStmt
	nTypeAssertExpr
	nTypeSpec
	nTypeSwitchStmt
	nUnaryExpr
	nValueSpec
)

// CHECK-LABEL: define %"iter.Seq[main.Cursor]" @main.Cursor.Preorder(%main.Cursor %0, %"{{.*}}/runtime/internal/runtime.Slice" %1){{.*}} {
// CHECK: store %main.Cursor %0, ptr [[PRE_CURSOR_SLOT:%[0-9]+]]
// CHECK: [[PRE_MASK:%[0-9]+]] = call i64 @main.maskOf(%"{{.*}}/runtime/internal/runtime.Slice" %1)
// CHECK: store i64 [[PRE_MASK]], ptr %{{[0-9]+}}
// CHECK: store ptr %{{[0-9]+}}, ptr %{{[0-9]+}}
// CHECK: store ptr %{{[0-9]+}}, ptr %{{[0-9]+}}
// CHECK: [[PRE_CLOSURE:%[0-9]+]] = insertvalue { ptr, ptr } { ptr @"main.Cursor.Preorder$1", ptr undef }, ptr %{{[0-9]+}}, 1
// CHECK: ret %"iter.Seq[main.Cursor]" %{{[0-9]+}}

// CHECK-LABEL: define void @"main.Cursor.Preorder$1"(ptr {{(nest|swiftself)}} %0, { ptr, ptr } %1){{.*}} {
// CHECK: [[PRE_ENV:%[0-9]+]] = load { ptr, ptr }, ptr %0
// CHECK: [[PRE_EVENTS:%[0-9]+]] = load %"{{.*}}/runtime/internal/runtime.Slice", ptr %{{[0-9]+}}
// CHECK: [[PRE_CURSOR_PTR:%[0-9]+]] = extractvalue { ptr, ptr } [[PRE_ENV]], 0
// CHECK: [[PRE_CURSOR:%[0-9]+]] = load %main.Cursor, ptr %{{[0-9]+}}
// CHECK-NEXT: [[PRE_RANGE:%[0-9]+]] = call { i32, i32 } @main.Cursor.indices(%main.Cursor [[PRE_CURSOR]])
// CHECK: [[PRE_START:%[0-9]+]] = extractvalue { i32, i32 } [[PRE_RANGE]], 0
// CHECK: [[PRE_LIMIT:%[0-9]+]] = extractvalue { i32, i32 } [[PRE_RANGE]], 1
// CHECK: [[PRE_I:%[0-9]+]] = phi i32 [ [[PRE_START]],
// CHECK: [[PRE_MORE:%[0-9]+]] = icmp slt i32 [[PRE_I]], [[PRE_LIMIT]]
// CHECK: [[PRE_EVENT:%[0-9]+]] = load %main.event, ptr %{{[0-9]+}}
// CHECK: [[PRE_POP:%[0-9]+]] = load i32, ptr %{{[0-9]+}}
// CHECK: [[PRE_PUSH:%[0-9]+]] = icmp sgt i32 [[PRE_POP]], [[PRE_I]]
// CHECK: [[PRE_EVENT_TYPE:%[0-9]+]] = load i64, ptr %{{[0-9]+}}
// CHECK: [[PRE_SAVED_MASK:%[0-9]+]] = load i64, ptr %{{[0-9]+}}
// CHECK: [[PRE_MATCH_BITS:%[0-9]+]] = and i64 [[PRE_EVENT_TYPE]], [[PRE_SAVED_MASK]]
// CHECK: [[PRE_MATCH:%[0-9]+]] = icmp ne i64 [[PRE_MATCH_BITS]], 0
// CHECK: [[PRE_SUBTREE_TYPE:%[0-9]+]] = load i64, ptr %{{[0-9]+}}
// CHECK: [[PRE_SUBTREE_MASK:%[0-9]+]] = load i64, ptr %{{[0-9]+}}
// CHECK: [[PRE_SUBTREE_BITS:%[0-9]+]] = and i64 [[PRE_SUBTREE_TYPE]], [[PRE_SUBTREE_MASK]]
// CHECK: [[PRE_SKIP:%[0-9]+]] = icmp eq i64 [[PRE_SUBTREE_BITS]], 0
// CHECK: store i32 [[PRE_I]], ptr %{{[0-9]+}}
// CHECK: [[PRE_YIELD_ENV:%[0-9]+]] = extractvalue { ptr, ptr } %1, 1
// CHECK-NEXT: [[PRE_YIELD_CODE:%[0-9]+]] = extractvalue { ptr, ptr } %1, 0
// CHECK: [[PRE_YIELD:%[0-9]+]] = call i1 %{{[^ ]+}}(ptr {{(nest|swiftself)}} [[PRE_YIELD_ENV]], %main.Cursor %{{[0-9]+}})
// CHECK: br i1 [[PRE_YIELD]],
// CHECK: [[PRE_AFTER_POP:%[0-9]+]] = add i32 [[PRE_SKIP_POP:%[0-9]+]], 1

// A virtual-root Cursor exposes all events; a real Cursor exposes its push/pop subtree.
// CHECK-LABEL: define { i32, i32 } @main.Cursor.indices(%main.Cursor %0){{.*}} {
// CHECK: store %main.Cursor %0, ptr [[IDX_CURSOR:%[0-9]+]]
// CHECK: [[IDX_INDEX_FIELD:%[0-9]+]] = getelementptr inbounds %main.Cursor, ptr [[IDX_CURSOR]], i32 0, i32 1
// CHECK-NEXT: [[IDX_INDEX:%[0-9]+]] = load i32, ptr [[IDX_INDEX_FIELD]]
// CHECK: [[IDX_ROOT:%[0-9]+]] = icmp slt i32 [[IDX_INDEX]], 0
// CHECK: [[IDX_ALL_LEN:%[0-9]+]] = extractvalue %"{{.*}}/runtime/internal/runtime.Slice" %{{[0-9]+}}, 1
// CHECK: [[IDX_ALL_END:%[0-9]+]] = trunc i64 [[IDX_ALL_LEN]] to i32
// CHECK: insertvalue { i32, i32 } { i32 0, i32 undef }, i32 [[IDX_ALL_END]], 1
// CHECK: [[IDX_PUSH:%[0-9]+]] = load i32, ptr %{{[0-9]+}}
// CHECK: [[IDX_EVENT:%[0-9]+]] = getelementptr inbounds %main.event, ptr %{{[0-9]+}}, i64 %{{[0-9]+}}
// CHECK: [[IDX_POP_PTR:%[0-9]+]] = getelementptr inbounds %main.event, ptr [[IDX_EVENT]], i32 0, i32 2
// CHECK: [[IDX_POP:%[0-9]+]] = load i32, ptr [[IDX_POP_PTR]]
// CHECK: [[IDX_END:%[0-9]+]] = add i32 [[IDX_POP]], 1
// CHECK: [[IDX_PAIR:%[0-9]+]] = insertvalue { i32, i32 } undef, i32 [[IDX_PUSH]], 0
// CHECK: insertvalue { i32, i32 } [[IDX_PAIR]], i32 [[IDX_END]], 1

// CHECK-LABEL: define i64 @main.maskOf(%"{{.*}}/runtime/internal/runtime.Slice" %0){{.*}} {
// CHECK: [[MASK_LEN:%[0-9]+]] = extractvalue %"{{.*}}/runtime/internal/runtime.Slice" %0, 1
// CHECK: [[MASK_ALL:%[0-9]+]] = icmp eq i64 [[MASK_LEN]], 0
// CHECK: ret i64 -1
// CHECK: [[MASK_ACC:%[0-9]+]] = phi i64 [ 0,
// CHECK: [[MASK_NODE:%[0-9]+]] = load %"{{.*}}/runtime/internal/runtime.iface", ptr %{{[0-9]+}}
// CHECK: [[MASK_ONE:%[0-9]+]] = call i64 @main.typeOf(%"{{.*}}/runtime/internal/runtime.iface" [[MASK_NODE]])
// CHECK: [[MASK_NEXT:%[0-9]+]] = or i64 [[MASK_ACC]], [[MASK_ONE]]
// CHECK: ret i64 [[MASK_ACC]]

// CHECK-LABEL: define i64 @main.typeOf(%"{{.*}}/runtime/internal/runtime.iface" %0){{.*}} {
// Ident has the dedicated fast path; the switch covers every declared node type
// and returns zero only for an unknown implementation.
// CHECK: [[TYPE_IDENT:%[0-9]+]] = call ptr @"{{.*}}/runtime/internal/runtime.IfaceType"(%"{{.*}}/runtime/internal/runtime.iface" %0)
// CHECK: [[IS_IDENT:%[0-9]+]] = icmp eq ptr [[TYPE_IDENT]], @"*_llgo_go/ast.Ident"
// CHECK: ret i64 1073741824
// CHECK: [[TYPE_ARRAY:%[0-9]+]] = call ptr @"{{.*}}/runtime/internal/runtime.IfaceType"(%"{{.*}}/runtime/internal/runtime.iface" %0)
// CHECK: [[IS_ARRAY:%[0-9]+]] = icmp eq ptr [[TYPE_ARRAY]], @"*_llgo_go/ast.ArrayType"
// CHECK: ret i64 1
// CHECK-COUNT-55: call ptr @"{{.*}}/runtime/internal/runtime.IfaceType"(%"{{.*}}/runtime/internal/runtime.iface" %0)
// CHECK: icmp eq ptr %{{[0-9]+}}, @"*_llgo_go/ast.ValueSpec"
// CHECK: ret i64 36028797018963968
// CHECK: ret i64 0
