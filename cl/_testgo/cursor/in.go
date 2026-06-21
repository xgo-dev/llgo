// LITTEST
package main

import (
	"go/ast"
	"iter"
	"math"
)

// CHECK: {{^}}@0 = private unnamed_addr constant [36 x i8] c"iterator call did not preserve panic", align 1{{$}}
// CHECK: {{^}}@2 = private unnamed_addr constant [43 x i8] c"yield function called after range loop exit", align 1{{$}}
// CHECK: {{^}}@3 = private unnamed_addr constant [47 x i8] c"{{.*}}/cl/_testgo/cursor.Cursor", align 1{{$}}
// CHECK: {{^}}@4 = private unnamed_addr constant [8 x i8] c"FindNode", align 1{{$}}
// CHECK: {{^}}@5 = private unnamed_addr constant [4 x i8] c"Node", align 1{{$}}
// CHECK: {{^}}@6 = private unnamed_addr constant [8 x i8] c"Preorder", align 1{{$}}
// CHECK: {{^}}@7 = private unnamed_addr constant [7 x i8] c"indices", align 1{{$}}

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

// CHECK-LABEL: define { %"{{.*}}/cl/_testgo/cursor.Cursor", i1 } @"{{.*}}/cl/_testgo/cursor.Cursor.FindNode"(%"{{.*}}/cl/_testgo/cursor.Cursor" %0, %"{{.*}}/runtime/internal/runtime.iface" %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = alloca %"{{.*}}/cl/_testgo/cursor.Cursor", align 8
// CHECK-NEXT:   call void @llvm.memset(ptr %2, i8 0, i64 16, i1 false)
// CHECK-NEXT:   store %"{{.*}}/cl/_testgo/cursor.Cursor" %0, ptr %2, align 8
// CHECK-NEXT:   %3 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.iface" %1, ptr %3, align 8
// CHECK-NEXT:   %4 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 16)
// CHECK-NEXT:   %5 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 1)
// CHECK-NEXT:   br i1 false, label %_llgo_1, label %_llgo_2
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_0
// CHECK-NEXT:   %6 = load %"{{.*}}/cl/_testgo/cursor.Cursor", ptr %2, align 8
// CHECK-NEXT:   %7 = load %"{{.*}}/runtime/internal/runtime.iface", ptr %3, align 8
// CHECK-NEXT:   %8 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 16)
// CHECK-NEXT:   %9 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.iface", ptr %8, i64 0
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.iface" %7, ptr %9, align 8
// CHECK-NEXT:   %10 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" undef, ptr %8, 0
// CHECK-NEXT:   %11 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" %10, i64 1, 1
// CHECK-NEXT:   %12 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" %11, i64 1, 2
// CHECK-NEXT:   %13 = call %"iter.Seq[{{.*}}/cl/_testgo/cursor.Cursor]" @"{{.*}}/cl/_testgo/cursor.Cursor.Preorder"(%"{{.*}}/cl/_testgo/cursor.Cursor" %6, %"{{.*}}/runtime/internal/runtime.Slice" %12)
// CHECK-NEXT:   %14 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 8)
// CHECK-NEXT:   %15 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 32)
// CHECK-NEXT:   %16 = getelementptr inbounds { ptr, ptr, ptr, ptr }, ptr %15, i32 0, i32 0
// CHECK-NEXT:   store ptr %14, ptr %16, align 8
// CHECK-NEXT:   %17 = getelementptr inbounds { ptr, ptr, ptr, ptr }, ptr %15, i32 0, i32 1
// CHECK-NEXT:   store ptr %3, ptr %17, align 8
// CHECK-NEXT:   %18 = getelementptr inbounds { ptr, ptr, ptr, ptr }, ptr %15, i32 0, i32 2
// CHECK-NEXT:   store ptr %4, ptr %18, align 8
// CHECK-NEXT:   %19 = getelementptr inbounds { ptr, ptr, ptr, ptr }, ptr %15, i32 0, i32 3
// CHECK-NEXT:   store ptr %5, ptr %19, align 8
// CHECK-NEXT:   %20 = insertvalue { ptr, ptr } { ptr @"{{.*}}/cl/_testgo/cursor.Cursor.FindNode$1", ptr undef }, ptr %15, 1
// CHECK-NEXT:   %21 = extractvalue %"iter.Seq[{{.*}}/cl/_testgo/cursor.Cursor]" %13, 1
// CHECK-NEXT:   %22 = extractvalue %"iter.Seq[{{.*}}/cl/_testgo/cursor.Cursor]" %13, 0
// CHECK-NEXT:   call void %22(ptr %21, { ptr, ptr } %20)
// CHECK-NEXT:   %23 = load i64, ptr %14, align 8
// CHECK-NEXT:   %24 = icmp eq i64 %23, -1
// CHECK-NEXT:   br i1 %24, label %_llgo_4, label %_llgo_5
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_0
// CHECK-NEXT:   %25 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 16)
// CHECK-NEXT:   %26 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.iface", ptr %25, i64 0
// CHECK-NEXT:   %27 = load %"{{.*}}/runtime/internal/runtime.iface", ptr %3, align 8
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.iface" %27, ptr %26, align 8
// CHECK-NEXT:   %28 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" undef, ptr %25, 0
// CHECK-NEXT:   %29 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" %28, i64 1, 1
// CHECK-NEXT:   %30 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" %29, i64 1, 2
// CHECK-NEXT:   %31 = call i64 @"{{.*}}/cl/_testgo/cursor.maskOf"(%"{{.*}}/runtime/internal/runtime.Slice" %30)
// CHECK-NEXT:   %32 = getelementptr inbounds %"{{.*}}/cl/_testgo/cursor.Cursor", ptr %2, i32 0, i32 0
// CHECK-NEXT:   %33 = load ptr, ptr %32, align 8
// CHECK-NEXT:   %34 = getelementptr inbounds %"{{.*}}/cl/_testgo/cursor.Inspector", ptr %33, i32 0, i32 0
// CHECK-NEXT:   %35 = load %"{{.*}}/runtime/internal/runtime.Slice", ptr %34, align 8
// CHECK-NEXT:   %36 = load %"{{.*}}/cl/_testgo/cursor.Cursor", ptr %2, align 8
// CHECK-NEXT:   %37 = call { i32, i32 } @"{{.*}}/cl/_testgo/cursor.Cursor.indices"(%"{{.*}}/cl/_testgo/cursor.Cursor" %36)
// CHECK-NEXT:   %38 = extractvalue { i32, i32 } %37, 0
// CHECK-NEXT:   %39 = extractvalue { i32, i32 } %37, 1
// CHECK-NEXT:   br label %_llgo_9
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_3:                                          ; preds = %_llgo_7, %_llgo_6
// CHECK-NEXT:   store %"{{.*}}/cl/_testgo/cursor.Cursor" zeroinitializer, ptr %4, align 8
// CHECK-NEXT:   store i1 false, ptr %5, align 1
// CHECK-NEXT:   %40 = load %"{{.*}}/cl/_testgo/cursor.Cursor", ptr %4, align 8
// CHECK-NEXT:   %41 = load i1, ptr %5, align 1
// CHECK-NEXT:   %42 = insertvalue { %"{{.*}}/cl/_testgo/cursor.Cursor", i1 } undef, %"{{.*}}/cl/_testgo/cursor.Cursor" %40, 0
// CHECK-NEXT:   %43 = insertvalue { %"{{.*}}/cl/_testgo/cursor.Cursor", i1 } %42, i1 %41, 1
// CHECK-NEXT:   ret { %"{{.*}}/cl/_testgo/cursor.Cursor", i1 } %43
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_4:                                          ; preds = %_llgo_1
// CHECK-NEXT:   %44 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @0, i64 36 }, ptr %44, align 8
// CHECK-NEXT:   %45 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_string, ptr undef }, ptr %44, 1
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.Panic"(%"{{.*}}/runtime/internal/runtime.eface" %45)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_5:                                          ; preds = %_llgo_1
// CHECK-NEXT:   %46 = icmp eq i64 %23, 0
// CHECK-NEXT:   br i1 %46, label %_llgo_6, label %_llgo_7
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_6:                                          ; preds = %_llgo_5
// CHECK-NEXT:   store i64 -2, ptr %14, align 8
// CHECK-NEXT:   br label %_llgo_3
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_7:                                          ; preds = %_llgo_5
// CHECK-NEXT:   %47 = icmp eq i64 %23, 1
// CHECK-NEXT:   br i1 %47, label %_llgo_8, label %_llgo_3
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_8:                                          ; preds = %_llgo_7
// CHECK-NEXT:   %48 = load %"{{.*}}/cl/_testgo/cursor.Cursor", ptr %4, align 8
// CHECK-NEXT:   %49 = load i1, ptr %5, align 1
// CHECK-NEXT:   %50 = insertvalue { %"{{.*}}/cl/_testgo/cursor.Cursor", i1 } undef, %"{{.*}}/cl/_testgo/cursor.Cursor" %48, 0
// CHECK-NEXT:   %51 = insertvalue { %"{{.*}}/cl/_testgo/cursor.Cursor", i1 } %50, i1 %49, 1
// CHECK-NEXT:   ret { %"{{.*}}/cl/_testgo/cursor.Cursor", i1 } %51
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_9:                                          ; preds = %_llgo_13, %_llgo_2
// CHECK-NEXT:   %52 = phi i32 [ %38, %_llgo_2 ], [ %75, %_llgo_13 ]
// CHECK-NEXT:   %53 = icmp slt i32 %52, %39
// CHECK-NEXT:   br i1 %53, label %_llgo_10, label %_llgo_11
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_10:                                         ; preds = %_llgo_9
// CHECK-NEXT:   %54 = alloca %"{{.*}}/cl/_testgo/cursor.event", align 8
// CHECK-NEXT:   call void @llvm.memset(ptr %54, i8 0, i64 32, i1 false)
// CHECK-NEXT:   %55 = extractvalue %"{{.*}}/runtime/internal/runtime.Slice" %35, 0
// CHECK-NEXT:   %56 = extractvalue %"{{.*}}/runtime/internal/runtime.Slice" %35, 1
// CHECK-NEXT:   %57 = sext i32 %52 to i64
// CHECK-NEXT:   %58 = icmp slt i64 %57, 0
// CHECK-NEXT:   %59 = icmp uge i64 %57, %56
// CHECK-NEXT:   %60 = or i1 %59, %58
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.CheckIndexRange"(i1 %60, {{.*}})
// CHECK-NEXT:   %61 = getelementptr inbounds %"{{.*}}/cl/_testgo/cursor.event", ptr %55, i64 %57
// CHECK-NEXT:   %62 = load %"{{.*}}/cl/_testgo/cursor.event", ptr %61, align 8
// CHECK-NEXT:   store %"{{.*}}/cl/_testgo/cursor.event" %62, ptr %54, align 8
// CHECK-NEXT:   %63 = getelementptr inbounds %"{{.*}}/cl/_testgo/cursor.event", ptr %54, i32 0, i32 2
// CHECK-NEXT:   %64 = load i32, ptr %63, align 4
// CHECK-NEXT:   %65 = icmp sgt i32 %64, %52
// CHECK-NEXT:   br i1 %65, label %_llgo_12, label %_llgo_13
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_11:                                         ; preds = %_llgo_9
// CHECK-NEXT:   store %"{{.*}}/cl/_testgo/cursor.Cursor" zeroinitializer, ptr %4, align 8
// CHECK-NEXT:   store i1 false, ptr %5, align 1
// CHECK-NEXT:   %66 = load %"{{.*}}/cl/_testgo/cursor.Cursor", ptr %4, align 8
// CHECK-NEXT:   %67 = load i1, ptr %5, align 1
// CHECK-NEXT:   %68 = insertvalue { %"{{.*}}/cl/_testgo/cursor.Cursor", i1 } undef, %"{{.*}}/cl/_testgo/cursor.Cursor" %66, 0
// CHECK-NEXT:   %69 = insertvalue { %"{{.*}}/cl/_testgo/cursor.Cursor", i1 } %68, i1 %67, 1
// CHECK-NEXT:   ret { %"{{.*}}/cl/_testgo/cursor.Cursor", i1 } %69
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_12:                                         ; preds = %_llgo_10
// CHECK-NEXT:   %70 = getelementptr inbounds %"{{.*}}/cl/_testgo/cursor.event", ptr %54, i32 0, i32 1
// CHECK-NEXT:   %71 = load i64, ptr %70, align 8
// CHECK-NEXT:   %72 = and i64 %71, %31
// CHECK-NEXT:   %73 = icmp ne i64 %72, 0
// CHECK-NEXT:   br i1 %73, label %_llgo_16, label %_llgo_15
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_13:                                         ; preds = %_llgo_17, %_llgo_15, %_llgo_10
// CHECK-NEXT:   %74 = phi i32 [ %52, %_llgo_10 ], [ %52, %_llgo_15 ], [ %87, %_llgo_17 ]
// CHECK-NEXT:   %75 = add i32 %74, 1
// CHECK-NEXT:   br label %_llgo_9
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_14:                                         ; preds = %_llgo_16
// CHECK-NEXT:   %76 = alloca %"{{.*}}/cl/_testgo/cursor.Cursor", align 8
// CHECK-NEXT:   call void @llvm.memset(ptr %76, i8 0, i64 16, i1 false)
// CHECK-NEXT:   %77 = getelementptr inbounds %"{{.*}}/cl/_testgo/cursor.Cursor", ptr %76, i32 0, i32 0
// CHECK-NEXT:   %78 = getelementptr inbounds %"{{.*}}/cl/_testgo/cursor.Cursor", ptr %2, i32 0, i32 0
// CHECK-NEXT:   %79 = load ptr, ptr %78, align 8
// CHECK-NEXT:   %80 = getelementptr inbounds %"{{.*}}/cl/_testgo/cursor.Cursor", ptr %76, i32 0, i32 1
// CHECK-NEXT:   store ptr %79, ptr %77, align 8
// CHECK-NEXT:   store i32 %52, ptr %80, align 4
// CHECK-NEXT:   %81 = load %"{{.*}}/cl/_testgo/cursor.Cursor", ptr %76, align 8
// CHECK-NEXT:   store %"{{.*}}/cl/_testgo/cursor.Cursor" %81, ptr %4, align 8
// CHECK-NEXT:   store i1 true, ptr %5, align 1
// CHECK-NEXT:   %82 = load %"{{.*}}/cl/_testgo/cursor.Cursor", ptr %4, align 8
// CHECK-NEXT:   %83 = load i1, ptr %5, align 1
// CHECK-NEXT:   %84 = insertvalue { %"{{.*}}/cl/_testgo/cursor.Cursor", i1 } undef, %"{{.*}}/cl/_testgo/cursor.Cursor" %82, 0
// CHECK-NEXT:   %85 = insertvalue { %"{{.*}}/cl/_testgo/cursor.Cursor", i1 } %84, i1 %83, 1
// CHECK-NEXT:   ret { %"{{.*}}/cl/_testgo/cursor.Cursor", i1 } %85
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_15:                                         ; preds = %_llgo_16, %_llgo_12
// CHECK-NEXT:   %86 = getelementptr inbounds %"{{.*}}/cl/_testgo/cursor.event", ptr %54, i32 0, i32 2
// CHECK-NEXT:   %87 = load i32, ptr %86, align 4
// CHECK-NEXT:   %88 = extractvalue %"{{.*}}/runtime/internal/runtime.Slice" %35, 0
// CHECK-NEXT:   %89 = extractvalue %"{{.*}}/runtime/internal/runtime.Slice" %35, 1
// CHECK-NEXT:   %90 = sext i32 %87 to i64
// CHECK-NEXT:   %91 = icmp slt i64 %90, 0
// CHECK-NEXT:   %92 = icmp uge i64 %90, %89
// CHECK-NEXT:   %93 = or i1 %92, %91
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.CheckIndexRange"(i1 %93, {{.*}})
// CHECK-NEXT:   %94 = getelementptr inbounds %"{{.*}}/cl/_testgo/cursor.event", ptr %88, i64 %90
// CHECK-NEXT:   %95 = getelementptr inbounds %"{{.*}}/cl/_testgo/cursor.event", ptr %94, i32 0, i32 1
// CHECK-NEXT:   %96 = load i64, ptr %95, align 8
// CHECK-NEXT:   %97 = and i64 %96, %31
// CHECK-NEXT:   %98 = icmp eq i64 %97, 0
// CHECK-NEXT:   br i1 %98, label %_llgo_17, label %_llgo_13
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_16:                                         ; preds = %_llgo_12
// CHECK-NEXT:   %99 = getelementptr inbounds %"{{.*}}/cl/_testgo/cursor.event", ptr %54, i32 0, i32 0
// CHECK-NEXT:   %100 = load %"{{.*}}/runtime/internal/runtime.iface", ptr %99, align 8
// CHECK-NEXT:   %101 = load %"{{.*}}/runtime/internal/runtime.iface", ptr %3, align 8
// CHECK-NEXT:   %102 = call ptr @"{{.*}}/runtime/internal/runtime.IfaceType"(%"{{.*}}/runtime/internal/runtime.iface" %100)
// CHECK-NEXT:   %103 = extractvalue %"{{.*}}/runtime/internal/runtime.iface" %100, 1
// CHECK-NEXT:   %104 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" undef, ptr %102, 0
// CHECK-NEXT:   %105 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" %104, ptr %103, 1
// CHECK-NEXT:   %106 = call ptr @"{{.*}}/runtime/internal/runtime.IfaceType"(%"{{.*}}/runtime/internal/runtime.iface" %101)
// CHECK-NEXT:   %107 = extractvalue %"{{.*}}/runtime/internal/runtime.iface" %101, 1
// CHECK-NEXT:   %108 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" undef, ptr %106, 0
// CHECK-NEXT:   %109 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" %108, ptr %107, 1
// CHECK-NEXT:   %110 = call i1 @"{{.*}}/runtime/internal/runtime.EfaceEqual"(%"{{.*}}/runtime/internal/runtime.eface" %105, %"{{.*}}/runtime/internal/runtime.eface" %109)
// CHECK-NEXT:   br i1 %110, label %_llgo_14, label %_llgo_15
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_17:                                         ; preds = %_llgo_15
// CHECK-NEXT:   br label %_llgo_13
// CHECK-NEXT: }

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

// CHECK-LABEL: define i1 @"{{.*}}/cl/_testgo/cursor.Cursor.FindNode$1"(ptr %0, %"{{.*}}/cl/_testgo/cursor.Cursor" %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = load { ptr, ptr, ptr, ptr }, ptr %0, align 8
// CHECK-NEXT:   %3 = extractvalue { ptr, ptr, ptr, ptr } %2, 0
// CHECK-NEXT:   %4 = load i64, ptr %3, align 8
// CHECK-NEXT:   %5 = icmp eq i64 %4, 0
// CHECK-NEXT:   br i1 %5, label %_llgo_1, label %_llgo_2
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_0
// CHECK-NEXT:   %6 = extractvalue { ptr, ptr, ptr, ptr } %2, 0
// CHECK-NEXT:   store i64 -1, ptr %6, align 8
// CHECK-NEXT:   %7 = call %"{{.*}}/runtime/internal/runtime.iface" @"{{.*}}/cl/_testgo/cursor.Cursor.Node"(%"{{.*}}/cl/_testgo/cursor.Cursor" %1)
// CHECK-NEXT:   %8 = extractvalue { ptr, ptr, ptr, ptr } %2, 1
// CHECK-NEXT:   %9 = load %"{{.*}}/runtime/internal/runtime.iface", ptr %8, align 8
// CHECK-NEXT:   %10 = call ptr @"{{.*}}/runtime/internal/runtime.IfaceType"(%"{{.*}}/runtime/internal/runtime.iface" %7)
// CHECK-NEXT:   %11 = extractvalue %"{{.*}}/runtime/internal/runtime.iface" %7, 1
// CHECK-NEXT:   %12 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" undef, ptr %10, 0
// CHECK-NEXT:   %13 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" %12, ptr %11, 1
// CHECK-NEXT:   %14 = call ptr @"{{.*}}/runtime/internal/runtime.IfaceType"(%"{{.*}}/runtime/internal/runtime.iface" %9)
// CHECK-NEXT:   %15 = extractvalue %"{{.*}}/runtime/internal/runtime.iface" %9, 1
// CHECK-NEXT:   %16 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" undef, ptr %14, 0
// CHECK-NEXT:   %17 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" %16, ptr %15, 1
// CHECK-NEXT:   %18 = call i1 @"{{.*}}/runtime/internal/runtime.EfaceEqual"(%"{{.*}}/runtime/internal/runtime.eface" %13, %"{{.*}}/runtime/internal/runtime.eface" %17)
// CHECK-NEXT:   br i1 %18, label %_llgo_3, label %_llgo_4
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_0
// CHECK-NEXT:   %19 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @2, i64 43 }, ptr %19, align 8
// CHECK-NEXT:   %20 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_string, ptr undef }, ptr %19, 1
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.Panic"(%"{{.*}}/runtime/internal/runtime.eface" %20)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_3:                                          ; preds = %_llgo_1
// CHECK-NEXT:   %21 = extractvalue { ptr, ptr, ptr, ptr } %2, 2
// CHECK-NEXT:   store %"{{.*}}/cl/_testgo/cursor.Cursor" %1, ptr %21, align 8
// CHECK-NEXT:   %22 = extractvalue { ptr, ptr, ptr, ptr } %2, 3
// CHECK-NEXT:   store i1 true, ptr %22, align 1
// CHECK-NEXT:   %23 = extractvalue { ptr, ptr, ptr, ptr } %2, 0
// CHECK-NEXT:   store i64 1, ptr %23, align 8
// CHECK-NEXT:   ret i1 false
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_4:                                          ; preds = %_llgo_1
// CHECK-NEXT:   %24 = extractvalue { ptr, ptr, ptr, ptr } %2, 0
// CHECK-NEXT:   store i64 0, ptr %24, align 8
// CHECK-NEXT:   ret i1 true
// CHECK-NEXT: }

// CHECK-LABEL: define %"{{.*}}/runtime/internal/runtime.iface" @"{{.*}}/cl/_testgo/cursor.Cursor.Node"(%"{{.*}}/cl/_testgo/cursor.Cursor" %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = alloca %"{{.*}}/cl/_testgo/cursor.Cursor", align 8
// CHECK-NEXT:   call void @llvm.memset(ptr %1, i8 0, i64 16, i1 false)
// CHECK-NEXT:   store %"{{.*}}/cl/_testgo/cursor.Cursor" %0, ptr %1, align 8
// CHECK-NEXT:   %2 = getelementptr inbounds %"{{.*}}/cl/_testgo/cursor.Cursor", ptr %1, i32 0, i32 1
// CHECK-NEXT:   %3 = load i32, ptr %2, align 4
// CHECK-NEXT:   %4 = icmp slt i32 %3, 0
// CHECK-NEXT:   br i1 %4, label %_llgo_1, label %_llgo_2
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_0
// CHECK-NEXT:   ret %"{{.*}}/runtime/internal/runtime.iface" zeroinitializer
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_0
// CHECK-NEXT:   %5 = getelementptr inbounds %"{{.*}}/cl/_testgo/cursor.Cursor", ptr %1, i32 0, i32 0
// CHECK-NEXT:   %6 = load ptr, ptr %5, align 8
// CHECK-NEXT:   %7 = getelementptr inbounds %"{{.*}}/cl/_testgo/cursor.Inspector", ptr %6, i32 0, i32 0
// CHECK-NEXT:   %8 = load %"{{.*}}/runtime/internal/runtime.Slice", ptr %7, align 8
// CHECK-NEXT:   %9 = getelementptr inbounds %"{{.*}}/cl/_testgo/cursor.Cursor", ptr %1, i32 0, i32 1
// CHECK-NEXT:   %10 = load i32, ptr %9, align 4
// CHECK-NEXT:   %11 = extractvalue %"{{.*}}/runtime/internal/runtime.Slice" %8, 0
// CHECK-NEXT:   %12 = extractvalue %"{{.*}}/runtime/internal/runtime.Slice" %8, 1
// CHECK-NEXT:   %13 = sext i32 %10 to i64
// CHECK-NEXT:   %14 = icmp slt i64 %13, 0
// CHECK-NEXT:   %15 = icmp uge i64 %13, %12
// CHECK-NEXT:   %16 = or i1 %15, %14
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.CheckIndexRange"(i1 %16, {{.*}})
// CHECK-NEXT:   %17 = getelementptr inbounds %"{{.*}}/cl/_testgo/cursor.event", ptr %11, i64 %13
// CHECK-NEXT:   %18 = getelementptr inbounds %"{{.*}}/cl/_testgo/cursor.event", ptr %17, i32 0, i32 0
// CHECK-NEXT:   %19 = load %"{{.*}}/runtime/internal/runtime.iface", ptr %18, align 8
// CHECK-NEXT:   ret %"{{.*}}/runtime/internal/runtime.iface" %19
// CHECK-NEXT: }

// CHECK-LABEL: define %"iter.Seq[{{.*}}/cl/_testgo/cursor.Cursor]" @"{{.*}}/cl/_testgo/cursor.Cursor.Preorder"(%"{{.*}}/cl/_testgo/cursor.Cursor" %0, %"{{.*}}/runtime/internal/runtime.Slice" %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/cl/_testgo/cursor.Cursor" %0, ptr %2, align 8
// CHECK-NEXT:   %3 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 8)
// CHECK-NEXT:   %4 = call i64 @"{{.*}}/cl/_testgo/cursor.maskOf"(%"{{.*}}/runtime/internal/runtime.Slice" %1)
// CHECK-NEXT:   store i64 %4, ptr %3, align 8
// CHECK-NEXT:   %5 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   %6 = getelementptr inbounds { ptr, ptr }, ptr %5, i32 0, i32 0
// CHECK-NEXT:   store ptr %2, ptr %6, align 8
// CHECK-NEXT:   %7 = getelementptr inbounds { ptr, ptr }, ptr %5, i32 0, i32 1
// CHECK-NEXT:   store ptr %3, ptr %7, align 8
// CHECK-NEXT:   %8 = insertvalue { ptr, ptr } { ptr @"{{.*}}/cl/_testgo/cursor.Cursor.Preorder$1", ptr undef }, ptr %5, 1
// CHECK-NEXT:   %9 = alloca %"iter.Seq[{{.*}}/cl/_testgo/cursor.Cursor]", align 8
// CHECK-NEXT:   store { ptr, ptr } %8, ptr %9, align 8
// CHECK-NEXT:   %10 = load %"iter.Seq[{{.*}}/cl/_testgo/cursor.Cursor]", ptr %9, align 8
// CHECK-NEXT:   ret %"iter.Seq[{{.*}}/cl/_testgo/cursor.Cursor]" %10
// CHECK-NEXT: }

// CHECK-LABEL: define void @"{{.*}}/cl/_testgo/cursor.Cursor.Preorder$1"(ptr %0, { ptr, ptr } %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = load { ptr, ptr }, ptr %0, align 8
// CHECK-NEXT:   %3 = extractvalue { ptr, ptr } %2, 0
// CHECK-NEXT:   %4 = getelementptr inbounds %"{{.*}}/cl/_testgo/cursor.Cursor", ptr %3, i32 0, i32 0
// CHECK-NEXT:   %5 = load ptr, ptr %4, align 8
// CHECK-NEXT:   %6 = getelementptr inbounds %"{{.*}}/cl/_testgo/cursor.Inspector", ptr %5, i32 0, i32 0
// CHECK-NEXT:   %7 = load %"{{.*}}/runtime/internal/runtime.Slice", ptr %6, align 8
// CHECK-NEXT:   %8 = extractvalue { ptr, ptr } %2, 0
// CHECK-NEXT:   %9 = load %"{{.*}}/cl/_testgo/cursor.Cursor", ptr %8, align 8
// CHECK-NEXT:   %10 = call { i32, i32 } @"{{.*}}/cl/_testgo/cursor.Cursor.indices"(%"{{.*}}/cl/_testgo/cursor.Cursor" %9)
// CHECK-NEXT:   %11 = extractvalue { i32, i32 } %10, 0
// CHECK-NEXT:   %12 = extractvalue { i32, i32 } %10, 1
// CHECK-NEXT:   br label %_llgo_1
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_5, %_llgo_8, %_llgo_0
// CHECK-NEXT:   %13 = phi i32 [ %11, %_llgo_0 ], [ %59, %_llgo_8 ], [ %33, %_llgo_5 ]
// CHECK-NEXT:   %14 = icmp slt i32 %13, %12
// CHECK-NEXT:   br i1 %14, label %_llgo_2, label %_llgo_3
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_1
// CHECK-NEXT:   %15 = alloca %"{{.*}}/cl/_testgo/cursor.event", align 8
// CHECK-NEXT:   call void @llvm.memset(ptr %15, i8 0, i64 32, i1 false)
// CHECK-NEXT:   %16 = extractvalue %"{{.*}}/runtime/internal/runtime.Slice" %7, 0
// CHECK-NEXT:   %17 = extractvalue %"{{.*}}/runtime/internal/runtime.Slice" %7, 1
// CHECK-NEXT:   %18 = sext i32 %13 to i64
// CHECK-NEXT:   %19 = icmp slt i64 %18, 0
// CHECK-NEXT:   %20 = icmp uge i64 %18, %17
// CHECK-NEXT:   %21 = or i1 %20, %19
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.CheckIndexRange"(i1 %21, {{.*}})
// CHECK-NEXT:   %22 = getelementptr inbounds %"{{.*}}/cl/_testgo/cursor.event", ptr %16, i64 %18
// CHECK-NEXT:   %23 = load %"{{.*}}/cl/_testgo/cursor.event", ptr %22, align 8
// CHECK-NEXT:   store %"{{.*}}/cl/_testgo/cursor.event" %23, ptr %15, align 8
// CHECK-NEXT:   %24 = getelementptr inbounds %"{{.*}}/cl/_testgo/cursor.event", ptr %15, i32 0, i32 2
// CHECK-NEXT:   %25 = load i32, ptr %24, align 4
// CHECK-NEXT:   %26 = icmp sgt i32 %25, %13
// CHECK-NEXT:   br i1 %26, label %_llgo_4, label %_llgo_5
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_3:                                          ; preds = %_llgo_7, %_llgo_1
// CHECK-NEXT:   ret void
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_4:                                          ; preds = %_llgo_2
// CHECK-NEXT:   %27 = getelementptr inbounds %"{{.*}}/cl/_testgo/cursor.event", ptr %15, i32 0, i32 1
// CHECK-NEXT:   %28 = load i64, ptr %27, align 8
// CHECK-NEXT:   %29 = extractvalue { ptr, ptr } %2, 1
// CHECK-NEXT:   %30 = load i64, ptr %29, align 8
// CHECK-NEXT:   %31 = and i64 %28, %30
// CHECK-NEXT:   %32 = icmp ne i64 %31, 0
// CHECK-NEXT:   br i1 %32, label %_llgo_7, label %_llgo_6
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_5:                                          ; preds = %_llgo_6, %_llgo_2
// CHECK-NEXT:   %33 = add i32 %13, 1
// CHECK-NEXT:   br label %_llgo_1
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_6:                                          ; preds = %_llgo_7, %_llgo_4
// CHECK-NEXT:   %34 = getelementptr inbounds %"{{.*}}/cl/_testgo/cursor.event", ptr %15, i32 0, i32 2
// CHECK-NEXT:   %35 = load i32, ptr %34, align 4
// CHECK-NEXT:   %36 = extractvalue %"{{.*}}/runtime/internal/runtime.Slice" %7, 0
// CHECK-NEXT:   %37 = extractvalue %"{{.*}}/runtime/internal/runtime.Slice" %7, 1
// CHECK-NEXT:   %38 = sext i32 %35 to i64
// CHECK-NEXT:   %39 = icmp slt i64 %38, 0
// CHECK-NEXT:   %40 = icmp uge i64 %38, %37
// CHECK-NEXT:   %41 = or i1 %40, %39
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.CheckIndexRange"(i1 %41, {{.*}})
// CHECK-NEXT:   %42 = getelementptr inbounds %"{{.*}}/cl/_testgo/cursor.event", ptr %36, i64 %38
// CHECK-NEXT:   %43 = getelementptr inbounds %"{{.*}}/cl/_testgo/cursor.event", ptr %42, i32 0, i32 1
// CHECK-NEXT:   %44 = load i64, ptr %43, align 8
// CHECK-NEXT:   %45 = extractvalue { ptr, ptr } %2, 1
// CHECK-NEXT:   %46 = load i64, ptr %45, align 8
// CHECK-NEXT:   %47 = and i64 %44, %46
// CHECK-NEXT:   %48 = icmp eq i64 %47, 0
// CHECK-NEXT:   br i1 %48, label %_llgo_8, label %_llgo_5
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_7:                                          ; preds = %_llgo_4
// CHECK-NEXT:   %49 = alloca %"{{.*}}/cl/_testgo/cursor.Cursor", align 8
// CHECK-NEXT:   call void @llvm.memset(ptr %49, i8 0, i64 16, i1 false)
// CHECK-NEXT:   %50 = getelementptr inbounds %"{{.*}}/cl/_testgo/cursor.Cursor", ptr %49, i32 0, i32 0
// CHECK-NEXT:   %51 = extractvalue { ptr, ptr } %2, 0
// CHECK-NEXT:   %52 = getelementptr inbounds %"{{.*}}/cl/_testgo/cursor.Cursor", ptr %51, i32 0, i32 0
// CHECK-NEXT:   %53 = load ptr, ptr %52, align 8
// CHECK-NEXT:   %54 = getelementptr inbounds %"{{.*}}/cl/_testgo/cursor.Cursor", ptr %49, i32 0, i32 1
// CHECK-NEXT:   store ptr %53, ptr %50, align 8
// CHECK-NEXT:   store i32 %13, ptr %54, align 4
// CHECK-NEXT:   %55 = load %"{{.*}}/cl/_testgo/cursor.Cursor", ptr %49, align 8
// CHECK-NEXT:   %56 = extractvalue { ptr, ptr } %1, 1
// CHECK-NEXT:   %57 = extractvalue { ptr, ptr } %1, 0
// CHECK-NEXT:   %58 = call i1 %57(ptr %56, %"{{.*}}/cl/_testgo/cursor.Cursor" %55)
// CHECK-NEXT:   br i1 %58, label %_llgo_6, label %_llgo_3
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_8:                                          ; preds = %_llgo_6
// CHECK-NEXT:   %59 = add i32 %35, 1
// CHECK-NEXT:   br label %_llgo_1
// CHECK-NEXT: }

// CHECK-LABEL: define { i32, i32 } @"{{.*}}/cl/_testgo/cursor.Cursor.indices"(%"{{.*}}/cl/_testgo/cursor.Cursor" %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = alloca %"{{.*}}/cl/_testgo/cursor.Cursor", align 8
// CHECK-NEXT:   call void @llvm.memset(ptr %1, i8 0, i64 16, i1 false)
// CHECK-NEXT:   store %"{{.*}}/cl/_testgo/cursor.Cursor" %0, ptr %1, align 8
// CHECK-NEXT:   %2 = getelementptr inbounds %"{{.*}}/cl/_testgo/cursor.Cursor", ptr %1, i32 0, i32 1
// CHECK-NEXT:   %3 = load i32, ptr %2, align 4
// CHECK-NEXT:   %4 = icmp slt i32 %3, 0
// CHECK-NEXT:   br i1 %4, label %_llgo_1, label %_llgo_2
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_0
// CHECK-NEXT:   %5 = getelementptr inbounds %"{{.*}}/cl/_testgo/cursor.Cursor", ptr %1, i32 0, i32 0
// CHECK-NEXT:   %6 = load ptr, ptr %5, align 8
// CHECK-NEXT:   %7 = getelementptr inbounds %"{{.*}}/cl/_testgo/cursor.Inspector", ptr %6, i32 0, i32 0
// CHECK-NEXT:   %8 = load %"{{.*}}/runtime/internal/runtime.Slice", ptr %7, align 8
// CHECK-NEXT:   %9 = extractvalue %"{{.*}}/runtime/internal/runtime.Slice" %8, 1
// CHECK-NEXT:   %10 = trunc i64 %9 to i32
// CHECK-NEXT:   %11 = insertvalue { i32, i32 } { i32 0, i32 undef }, i32 %10, 1
// CHECK-NEXT:   ret { i32, i32 } %11
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_0
// CHECK-NEXT:   %12 = getelementptr inbounds %"{{.*}}/cl/_testgo/cursor.Cursor", ptr %1, i32 0, i32 1
// CHECK-NEXT:   %13 = load i32, ptr %12, align 4
// CHECK-NEXT:   %14 = getelementptr inbounds %"{{.*}}/cl/_testgo/cursor.Cursor", ptr %1, i32 0, i32 0
// CHECK-NEXT:   %15 = load ptr, ptr %14, align 8
// CHECK-NEXT:   %16 = getelementptr inbounds %"{{.*}}/cl/_testgo/cursor.Inspector", ptr %15, i32 0, i32 0
// CHECK-NEXT:   %17 = load %"{{.*}}/runtime/internal/runtime.Slice", ptr %16, align 8
// CHECK-NEXT:   %18 = getelementptr inbounds %"{{.*}}/cl/_testgo/cursor.Cursor", ptr %1, i32 0, i32 1
// CHECK-NEXT:   %19 = load i32, ptr %18, align 4
// CHECK-NEXT:   %20 = extractvalue %"{{.*}}/runtime/internal/runtime.Slice" %17, 0
// CHECK-NEXT:   %21 = extractvalue %"{{.*}}/runtime/internal/runtime.Slice" %17, 1
// CHECK-NEXT:   %22 = sext i32 %19 to i64
// CHECK-NEXT:   %23 = icmp slt i64 %22, 0
// CHECK-NEXT:   %24 = icmp uge i64 %22, %21
// CHECK-NEXT:   %25 = or i1 %24, %23
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.CheckIndexRange"(i1 %25, {{.*}})
// CHECK-NEXT:   %26 = getelementptr inbounds %"{{.*}}/cl/_testgo/cursor.event", ptr %20, i64 %22
// CHECK-NEXT:   %27 = getelementptr inbounds %"{{.*}}/cl/_testgo/cursor.event", ptr %26, i32 0, i32 2
// CHECK-NEXT:   %28 = load i32, ptr %27, align 4
// CHECK-NEXT:   %29 = add i32 %28, 1
// CHECK-NEXT:   %30 = insertvalue { i32, i32 } undef, i32 %13, 0
// CHECK-NEXT:   %31 = insertvalue { i32, i32 } %30, i32 %29, 1
// CHECK-NEXT:   ret { i32, i32 } %31
// CHECK-NEXT: }

// CHECK-LABEL: define { %"{{.*}}/cl/_testgo/cursor.Cursor", i1 } @"{{.*}}/cl/_testgo/cursor.(*Cursor).FindNode"(ptr %0, %"{{.*}}/runtime/internal/runtime.iface" %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = icmp eq ptr %0, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PanicWrapNilPointer"(i1 %2, %"{{.*}}/runtime/internal/runtime.String" { ptr @3, i64 47 }, %"{{.*}}/runtime/internal/runtime.String" { ptr @4, i64 8 })
// CHECK-NEXT:   %3 = load %"{{.*}}/cl/_testgo/cursor.Cursor", ptr %0, align 8
// CHECK-NEXT:   %4 = call { %"{{.*}}/cl/_testgo/cursor.Cursor", i1 } @"{{.*}}/cl/_testgo/cursor.Cursor.FindNode"(%"{{.*}}/cl/_testgo/cursor.Cursor" %3, %"{{.*}}/runtime/internal/runtime.iface" %1)
// CHECK-NEXT:   %5 = extractvalue { %"{{.*}}/cl/_testgo/cursor.Cursor", i1 } %4, 0
// CHECK-NEXT:   %6 = extractvalue { %"{{.*}}/cl/_testgo/cursor.Cursor", i1 } %4, 1
// CHECK-NEXT:   %7 = insertvalue { %"{{.*}}/cl/_testgo/cursor.Cursor", i1 } undef, %"{{.*}}/cl/_testgo/cursor.Cursor" %5, 0
// CHECK-NEXT:   %8 = insertvalue { %"{{.*}}/cl/_testgo/cursor.Cursor", i1 } %7, i1 %6, 1
// CHECK-NEXT:   ret { %"{{.*}}/cl/_testgo/cursor.Cursor", i1 } %8
// CHECK-NEXT: }

// CHECK-LABEL: define %"{{.*}}/runtime/internal/runtime.iface" @"{{.*}}/cl/_testgo/cursor.(*Cursor).Node"(ptr %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = icmp eq ptr %0, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PanicWrapNilPointer"(i1 %1, %"{{.*}}/runtime/internal/runtime.String" { ptr @3, i64 47 }, %"{{.*}}/runtime/internal/runtime.String" { ptr @5, i64 4 })
// CHECK-NEXT:   %2 = load %"{{.*}}/cl/_testgo/cursor.Cursor", ptr %0, align 8
// CHECK-NEXT:   %3 = call %"{{.*}}/runtime/internal/runtime.iface" @"{{.*}}/cl/_testgo/cursor.Cursor.Node"(%"{{.*}}/cl/_testgo/cursor.Cursor" %2)
// CHECK-NEXT:   ret %"{{.*}}/runtime/internal/runtime.iface" %3
// CHECK-NEXT: }

// CHECK-LABEL: define %"iter.Seq[{{.*}}/cl/_testgo/cursor.Cursor]" @"{{.*}}/cl/_testgo/cursor.(*Cursor).Preorder"(ptr %0, %"{{.*}}/runtime/internal/runtime.Slice" %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = icmp eq ptr %0, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PanicWrapNilPointer"(i1 %2, %"{{.*}}/runtime/internal/runtime.String" { ptr @3, i64 47 }, %"{{.*}}/runtime/internal/runtime.String" { ptr @6, i64 8 })
// CHECK-NEXT:   %3 = load %"{{.*}}/cl/_testgo/cursor.Cursor", ptr %0, align 8
// CHECK-NEXT:   %4 = call %"iter.Seq[{{.*}}/cl/_testgo/cursor.Cursor]" @"{{.*}}/cl/_testgo/cursor.Cursor.Preorder"(%"{{.*}}/cl/_testgo/cursor.Cursor" %3, %"{{.*}}/runtime/internal/runtime.Slice" %1)
// CHECK-NEXT:   ret %"iter.Seq[{{.*}}/cl/_testgo/cursor.Cursor]" %4
// CHECK-NEXT: }

// CHECK-LABEL: define { i32, i32 } @"{{.*}}/cl/_testgo/cursor.(*Cursor).indices"(ptr %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = icmp eq ptr %0, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PanicWrapNilPointer"(i1 %1, %"{{.*}}/runtime/internal/runtime.String" { ptr @3, i64 47 }, %"{{.*}}/runtime/internal/runtime.String" { ptr @7, i64 7 })
// CHECK-NEXT:   %2 = load %"{{.*}}/cl/_testgo/cursor.Cursor", ptr %0, align 8
// CHECK-NEXT:   %3 = call { i32, i32 } @"{{.*}}/cl/_testgo/cursor.Cursor.indices"(%"{{.*}}/cl/_testgo/cursor.Cursor" %2)
// CHECK-NEXT:   %4 = extractvalue { i32, i32 } %3, 0
// CHECK-NEXT:   %5 = extractvalue { i32, i32 } %3, 1
// CHECK-NEXT:   %6 = insertvalue { i32, i32 } undef, i32 %4, 0
// CHECK-NEXT:   %7 = insertvalue { i32, i32 } %6, i32 %5, 1
// CHECK-NEXT:   ret { i32, i32 } %7
// CHECK-NEXT: }

// CHECK-LABEL: define void @"{{.*}}/cl/_testgo/cursor.init"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = load i1, ptr @"{{.*}}/cl/_testgo/cursor.init$guard", align 1
// CHECK-NEXT:   br i1 %0, label %_llgo_2, label %_llgo_1
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_0
// CHECK-NEXT:   store i1 true, ptr @"{{.*}}/cl/_testgo/cursor.init$guard", align 1
// CHECK-NEXT:   call void @"go/ast.init"()
// CHECK-NEXT:   call void @iter.init()
// CHECK-NEXT:   call void @math.init()
// CHECK-NEXT:   br label %_llgo_2
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_1, %_llgo_0
// CHECK-NEXT:   ret void
// CHECK-NEXT: }

// CHECK-LABEL: define void @"{{.*}}/cl/_testgo/cursor.main"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 16)
// CHECK-NEXT:   %1 = getelementptr inbounds %"{{.*}}/cl/_testgo/cursor.Cursor", ptr %0, i32 0, i32 0
// CHECK-NEXT:   %2 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 24)
// CHECK-NEXT:   store ptr %2, ptr %1, align 8
// CHECK-NEXT:   ret void
// CHECK-NEXT: }

// CHECK-LABEL: define i64 @"{{.*}}/cl/_testgo/cursor.maskOf"(%"{{.*}}/runtime/internal/runtime.Slice" %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = extractvalue %"{{.*}}/runtime/internal/runtime.Slice" %0, 1
// CHECK-NEXT:   %2 = icmp eq i64 %1, 0
// CHECK-NEXT:   br i1 %2, label %_llgo_1, label %_llgo_2
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_0
// CHECK-NEXT:   ret i64 -1
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_0
// CHECK-NEXT:   %3 = extractvalue %"{{.*}}/runtime/internal/runtime.Slice" %0, 1
// CHECK-NEXT:   br label %_llgo_3
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_3:                                          ; preds = %_llgo_4, %_llgo_2
// CHECK-NEXT:   %4 = phi i64 [ 0, %_llgo_2 ], [ %16, %_llgo_4 ]
// CHECK-NEXT:   %5 = phi i64 [ -1, %_llgo_2 ], [ %6, %_llgo_4 ]
// CHECK-NEXT:   %6 = add i64 %5, 1
// CHECK-NEXT:   %7 = icmp slt i64 %6, %3
// CHECK-NEXT:   br i1 %7, label %_llgo_4, label %_llgo_5
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_4:                                          ; preds = %_llgo_3
// CHECK-NEXT:   %8 = extractvalue %"{{.*}}/runtime/internal/runtime.Slice" %0, 0
// CHECK-NEXT:   %9 = extractvalue %"{{.*}}/runtime/internal/runtime.Slice" %0, 1
// CHECK-NEXT:   %10 = icmp slt i64 %6, 0
// CHECK-NEXT:   %11 = icmp uge i64 %6, %9
// CHECK-NEXT:   %12 = or i1 %11, %10
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.CheckIndexRange"(i1 %12, {{.*}})
// CHECK-NEXT:   %13 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.iface", ptr %8, i64 %6
// CHECK-NEXT:   %14 = load %"{{.*}}/runtime/internal/runtime.iface", ptr %13, align 8
// CHECK-NEXT:   %15 = call i64 @"{{.*}}/cl/_testgo/cursor.typeOf"(%"{{.*}}/runtime/internal/runtime.iface" %14)
// CHECK-NEXT:   %16 = or i64 %4, %15
// CHECK-NEXT:   br label %_llgo_3
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_5:                                          ; preds = %_llgo_3
// CHECK-NEXT:   ret i64 %4
// CHECK-NEXT: }

// CHECK-LABEL: define i64 @"{{.*}}/cl/_testgo/cursor.typeOf"(%"{{.*}}/runtime/internal/runtime.iface" %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = call ptr @"{{.*}}/runtime/internal/runtime.IfaceType"(%"{{.*}}/runtime/internal/runtime.iface" %0)
// CHECK-NEXT:   %2 = icmp eq ptr %1, @"*_llgo_go/ast.Ident"
// CHECK-NEXT:   br i1 %2, label %_llgo_115, label %_llgo_116
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_117
// CHECK-NEXT:   ret i64 1073741824
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_117
// CHECK-NEXT:   %3 = call ptr @"{{.*}}/runtime/internal/runtime.IfaceType"(%"{{.*}}/runtime/internal/runtime.iface" %0)
// CHECK-NEXT:   %4 = icmp eq ptr %3, @"*_llgo_go/ast.ArrayType"
// CHECK-NEXT:   br i1 %4, label %_llgo_118, label %_llgo_119
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_3:                                          ; preds = %_llgo_120
// CHECK-NEXT:   ret i64 1
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_4:                                          ; preds = %_llgo_120
// CHECK-NEXT:   %5 = call ptr @"{{.*}}/runtime/internal/runtime.IfaceType"(%"{{.*}}/runtime/internal/runtime.iface" %0)
// CHECK-NEXT:   %6 = icmp eq ptr %5, @"*_llgo_go/ast.AssignStmt"
// CHECK-NEXT:   br i1 %6, label %_llgo_121, label %_llgo_122
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_5:                                          ; preds = %_llgo_123
// CHECK-NEXT:   ret i64 2
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_6:                                          ; preds = %_llgo_123
// CHECK-NEXT:   %7 = call ptr @"{{.*}}/runtime/internal/runtime.IfaceType"(%"{{.*}}/runtime/internal/runtime.iface" %0)
// CHECK-NEXT:   %8 = icmp eq ptr %7, @"*_llgo_go/ast.BadDecl"
// CHECK-NEXT:   br i1 %8, label %_llgo_124, label %_llgo_125
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_7:                                          ; preds = %_llgo_126
// CHECK-NEXT:   ret i64 4
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_8:                                          ; preds = %_llgo_126
// CHECK-NEXT:   %9 = call ptr @"{{.*}}/runtime/internal/runtime.IfaceType"(%"{{.*}}/runtime/internal/runtime.iface" %0)
// CHECK-NEXT:   %10 = icmp eq ptr %9, @"*_llgo_go/ast.BadExpr"
// CHECK-NEXT:   br i1 %10, label %_llgo_127, label %_llgo_128
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_9:                                          ; preds = %_llgo_129
// CHECK-NEXT:   ret i64 8
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_10:                                         ; preds = %_llgo_129
// CHECK-NEXT:   %11 = call ptr @"{{.*}}/runtime/internal/runtime.IfaceType"(%"{{.*}}/runtime/internal/runtime.iface" %0)
// CHECK-NEXT:   %12 = icmp eq ptr %11, @"*_llgo_go/ast.BadStmt"
// CHECK-NEXT:   br i1 %12, label %_llgo_130, label %_llgo_131
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_11:                                         ; preds = %_llgo_132
// CHECK-NEXT:   ret i64 16
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_12:                                         ; preds = %_llgo_132
// CHECK-NEXT:   %13 = call ptr @"{{.*}}/runtime/internal/runtime.IfaceType"(%"{{.*}}/runtime/internal/runtime.iface" %0)
// CHECK-NEXT:   %14 = icmp eq ptr %13, @"*_llgo_go/ast.BasicLit"
// CHECK-NEXT:   br i1 %14, label %_llgo_133, label %_llgo_134
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_13:                                         ; preds = %_llgo_135
// CHECK-NEXT:   ret i64 32
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_14:                                         ; preds = %_llgo_135
// CHECK-NEXT:   %15 = call ptr @"{{.*}}/runtime/internal/runtime.IfaceType"(%"{{.*}}/runtime/internal/runtime.iface" %0)
// CHECK-NEXT:   %16 = icmp eq ptr %15, @"*_llgo_go/ast.BinaryExpr"
// CHECK-NEXT:   br i1 %16, label %_llgo_136, label %_llgo_137
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_15:                                         ; preds = %_llgo_138
// CHECK-NEXT:   ret i64 64
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_16:                                         ; preds = %_llgo_138
// CHECK-NEXT:   %17 = call ptr @"{{.*}}/runtime/internal/runtime.IfaceType"(%"{{.*}}/runtime/internal/runtime.iface" %0)
// CHECK-NEXT:   %18 = icmp eq ptr %17, @"*_llgo_go/ast.BlockStmt"
// CHECK-NEXT:   br i1 %18, label %_llgo_139, label %_llgo_140
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_17:                                         ; preds = %_llgo_141
// CHECK-NEXT:   ret i64 128
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_18:                                         ; preds = %_llgo_141
// CHECK-NEXT:   %19 = call ptr @"{{.*}}/runtime/internal/runtime.IfaceType"(%"{{.*}}/runtime/internal/runtime.iface" %0)
// CHECK-NEXT:   %20 = icmp eq ptr %19, @"*_llgo_go/ast.BranchStmt"
// CHECK-NEXT:   br i1 %20, label %_llgo_142, label %_llgo_143
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_19:                                         ; preds = %_llgo_144
// CHECK-NEXT:   ret i64 256
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_20:                                         ; preds = %_llgo_144
// CHECK-NEXT:   %21 = call ptr @"{{.*}}/runtime/internal/runtime.IfaceType"(%"{{.*}}/runtime/internal/runtime.iface" %0)
// CHECK-NEXT:   %22 = icmp eq ptr %21, @"*_llgo_go/ast.CallExpr"
// CHECK-NEXT:   br i1 %22, label %_llgo_145, label %_llgo_146
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_21:                                         ; preds = %_llgo_147
// CHECK-NEXT:   ret i64 512
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_22:                                         ; preds = %_llgo_147
// CHECK-NEXT:   %23 = call ptr @"{{.*}}/runtime/internal/runtime.IfaceType"(%"{{.*}}/runtime/internal/runtime.iface" %0)
// CHECK-NEXT:   %24 = icmp eq ptr %23, @"*_llgo_go/ast.CaseClause"
// CHECK-NEXT:   br i1 %24, label %_llgo_148, label %_llgo_149
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_23:                                         ; preds = %_llgo_150
// CHECK-NEXT:   ret i64 1024
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_24:                                         ; preds = %_llgo_150
// CHECK-NEXT:   %25 = call ptr @"{{.*}}/runtime/internal/runtime.IfaceType"(%"{{.*}}/runtime/internal/runtime.iface" %0)
// CHECK-NEXT:   %26 = icmp eq ptr %25, @"*_llgo_go/ast.ChanType"
// CHECK-NEXT:   br i1 %26, label %_llgo_151, label %_llgo_152
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_25:                                         ; preds = %_llgo_153
// CHECK-NEXT:   ret i64 2048
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_26:                                         ; preds = %_llgo_153
// CHECK-NEXT:   %27 = call ptr @"{{.*}}/runtime/internal/runtime.IfaceType"(%"{{.*}}/runtime/internal/runtime.iface" %0)
// CHECK-NEXT:   %28 = icmp eq ptr %27, @"*_llgo_go/ast.CommClause"
// CHECK-NEXT:   br i1 %28, label %_llgo_154, label %_llgo_155
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_27:                                         ; preds = %_llgo_156
// CHECK-NEXT:   ret i64 4096
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_28:                                         ; preds = %_llgo_156
// CHECK-NEXT:   %29 = call ptr @"{{.*}}/runtime/internal/runtime.IfaceType"(%"{{.*}}/runtime/internal/runtime.iface" %0)
// CHECK-NEXT:   %30 = icmp eq ptr %29, @"*_llgo_go/ast.Comment"
// CHECK-NEXT:   br i1 %30, label %_llgo_157, label %_llgo_158
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_29:                                         ; preds = %_llgo_159
// CHECK-NEXT:   ret i64 8192
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_30:                                         ; preds = %_llgo_159
// CHECK-NEXT:   %31 = call ptr @"{{.*}}/runtime/internal/runtime.IfaceType"(%"{{.*}}/runtime/internal/runtime.iface" %0)
// CHECK-NEXT:   %32 = icmp eq ptr %31, @"*_llgo_go/ast.CommentGroup"
// CHECK-NEXT:   br i1 %32, label %_llgo_160, label %_llgo_161
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_31:                                         ; preds = %_llgo_162
// CHECK-NEXT:   ret i64 16384
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_32:                                         ; preds = %_llgo_162
// CHECK-NEXT:   %33 = call ptr @"{{.*}}/runtime/internal/runtime.IfaceType"(%"{{.*}}/runtime/internal/runtime.iface" %0)
// CHECK-NEXT:   %34 = icmp eq ptr %33, @"*_llgo_go/ast.CompositeLit"
// CHECK-NEXT:   br i1 %34, label %_llgo_163, label %_llgo_164
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_33:                                         ; preds = %_llgo_165
// CHECK-NEXT:   ret i64 32768
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_34:                                         ; preds = %_llgo_165
// CHECK-NEXT:   %35 = call ptr @"{{.*}}/runtime/internal/runtime.IfaceType"(%"{{.*}}/runtime/internal/runtime.iface" %0)
// CHECK-NEXT:   %36 = icmp eq ptr %35, @"*_llgo_go/ast.DeclStmt"
// CHECK-NEXT:   br i1 %36, label %_llgo_166, label %_llgo_167
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_35:                                         ; preds = %_llgo_168
// CHECK-NEXT:   ret i64 65536
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_36:                                         ; preds = %_llgo_168
// CHECK-NEXT:   %37 = call ptr @"{{.*}}/runtime/internal/runtime.IfaceType"(%"{{.*}}/runtime/internal/runtime.iface" %0)
// CHECK-NEXT:   %38 = icmp eq ptr %37, @"*_llgo_go/ast.DeferStmt"
// CHECK-NEXT:   br i1 %38, label %_llgo_169, label %_llgo_170
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_37:                                         ; preds = %_llgo_171
// CHECK-NEXT:   ret i64 131072
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_38:                                         ; preds = %_llgo_171
// CHECK-NEXT:   %39 = call ptr @"{{.*}}/runtime/internal/runtime.IfaceType"(%"{{.*}}/runtime/internal/runtime.iface" %0)
// CHECK-NEXT:   %40 = icmp eq ptr %39, @"*_llgo_go/ast.Ellipsis"
// CHECK-NEXT:   br i1 %40, label %_llgo_172, label %_llgo_173
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_39:                                         ; preds = %_llgo_174
// CHECK-NEXT:   ret i64 262144
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_40:                                         ; preds = %_llgo_174
// CHECK-NEXT:   %41 = call ptr @"{{.*}}/runtime/internal/runtime.IfaceType"(%"{{.*}}/runtime/internal/runtime.iface" %0)
// CHECK-NEXT:   %42 = icmp eq ptr %41, @"*_llgo_go/ast.EmptyStmt"
// CHECK-NEXT:   br i1 %42, label %_llgo_175, label %_llgo_176
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_41:                                         ; preds = %_llgo_177
// CHECK-NEXT:   ret i64 524288
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_42:                                         ; preds = %_llgo_177
// CHECK-NEXT:   %43 = call ptr @"{{.*}}/runtime/internal/runtime.IfaceType"(%"{{.*}}/runtime/internal/runtime.iface" %0)
// CHECK-NEXT:   %44 = icmp eq ptr %43, @"*_llgo_go/ast.ExprStmt"
// CHECK-NEXT:   br i1 %44, label %_llgo_178, label %_llgo_179
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_43:                                         ; preds = %_llgo_180
// CHECK-NEXT:   ret i64 1048576
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_44:                                         ; preds = %_llgo_180
// CHECK-NEXT:   %45 = call ptr @"{{.*}}/runtime/internal/runtime.IfaceType"(%"{{.*}}/runtime/internal/runtime.iface" %0)
// CHECK-NEXT:   %46 = icmp eq ptr %45, @"*_llgo_go/ast.Field"
// CHECK-NEXT:   br i1 %46, label %_llgo_181, label %_llgo_182
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_45:                                         ; preds = %_llgo_183
// CHECK-NEXT:   ret i64 2097152
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_46:                                         ; preds = %_llgo_183
// CHECK-NEXT:   %47 = call ptr @"{{.*}}/runtime/internal/runtime.IfaceType"(%"{{.*}}/runtime/internal/runtime.iface" %0)
// CHECK-NEXT:   %48 = icmp eq ptr %47, @"*_llgo_go/ast.FieldList"
// CHECK-NEXT:   br i1 %48, label %_llgo_184, label %_llgo_185
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_47:                                         ; preds = %_llgo_186
// CHECK-NEXT:   ret i64 4194304
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_48:                                         ; preds = %_llgo_186
// CHECK-NEXT:   %49 = call ptr @"{{.*}}/runtime/internal/runtime.IfaceType"(%"{{.*}}/runtime/internal/runtime.iface" %0)
// CHECK-NEXT:   %50 = icmp eq ptr %49, @"*_llgo_go/ast.File"
// CHECK-NEXT:   br i1 %50, label %_llgo_187, label %_llgo_188
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_49:                                         ; preds = %_llgo_189
// CHECK-NEXT:   ret i64 8388608
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_50:                                         ; preds = %_llgo_189
// CHECK-NEXT:   %51 = call ptr @"{{.*}}/runtime/internal/runtime.IfaceType"(%"{{.*}}/runtime/internal/runtime.iface" %0)
// CHECK-NEXT:   %52 = icmp eq ptr %51, @"*_llgo_go/ast.ForStmt"
// CHECK-NEXT:   br i1 %52, label %_llgo_190, label %_llgo_191
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_51:                                         ; preds = %_llgo_192
// CHECK-NEXT:   ret i64 16777216
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_52:                                         ; preds = %_llgo_192
// CHECK-NEXT:   %53 = call ptr @"{{.*}}/runtime/internal/runtime.IfaceType"(%"{{.*}}/runtime/internal/runtime.iface" %0)
// CHECK-NEXT:   %54 = icmp eq ptr %53, @"*_llgo_go/ast.FuncDecl"
// CHECK-NEXT:   br i1 %54, label %_llgo_193, label %_llgo_194
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_53:                                         ; preds = %_llgo_195
// CHECK-NEXT:   ret i64 33554432
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_54:                                         ; preds = %_llgo_195
// CHECK-NEXT:   %55 = call ptr @"{{.*}}/runtime/internal/runtime.IfaceType"(%"{{.*}}/runtime/internal/runtime.iface" %0)
// CHECK-NEXT:   %56 = icmp eq ptr %55, @"*_llgo_go/ast.FuncLit"
// CHECK-NEXT:   br i1 %56, label %_llgo_196, label %_llgo_197
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_55:                                         ; preds = %_llgo_198
// CHECK-NEXT:   ret i64 67108864
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_56:                                         ; preds = %_llgo_198
// CHECK-NEXT:   %57 = call ptr @"{{.*}}/runtime/internal/runtime.IfaceType"(%"{{.*}}/runtime/internal/runtime.iface" %0)
// CHECK-NEXT:   %58 = icmp eq ptr %57, @"*_llgo_go/ast.FuncType"
// CHECK-NEXT:   br i1 %58, label %_llgo_199, label %_llgo_200
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_57:                                         ; preds = %_llgo_201
// CHECK-NEXT:   ret i64 134217728
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_58:                                         ; preds = %_llgo_201
// CHECK-NEXT:   %59 = call ptr @"{{.*}}/runtime/internal/runtime.IfaceType"(%"{{.*}}/runtime/internal/runtime.iface" %0)
// CHECK-NEXT:   %60 = icmp eq ptr %59, @"*_llgo_go/ast.GenDecl"
// CHECK-NEXT:   br i1 %60, label %_llgo_202, label %_llgo_203
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_59:                                         ; preds = %_llgo_204
// CHECK-NEXT:   ret i64 268435456
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_60:                                         ; preds = %_llgo_204
// CHECK-NEXT:   %61 = call ptr @"{{.*}}/runtime/internal/runtime.IfaceType"(%"{{.*}}/runtime/internal/runtime.iface" %0)
// CHECK-NEXT:   %62 = icmp eq ptr %61, @"*_llgo_go/ast.GoStmt"
// CHECK-NEXT:   br i1 %62, label %_llgo_205, label %_llgo_206
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_61:                                         ; preds = %_llgo_207
// CHECK-NEXT:   ret i64 536870912
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_62:                                         ; preds = %_llgo_207
// CHECK-NEXT:   %63 = call ptr @"{{.*}}/runtime/internal/runtime.IfaceType"(%"{{.*}}/runtime/internal/runtime.iface" %0)
// CHECK-NEXT:   %64 = icmp eq ptr %63, @"*_llgo_go/ast.Ident"
// CHECK-NEXT:   br i1 %64, label %_llgo_208, label %_llgo_209
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_63:                                         ; preds = %_llgo_210
// CHECK-NEXT:   ret i64 1073741824
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_64:                                         ; preds = %_llgo_210
// CHECK-NEXT:   %65 = call ptr @"{{.*}}/runtime/internal/runtime.IfaceType"(%"{{.*}}/runtime/internal/runtime.iface" %0)
// CHECK-NEXT:   %66 = icmp eq ptr %65, @"*_llgo_go/ast.IfStmt"
// CHECK-NEXT:   br i1 %66, label %_llgo_211, label %_llgo_212
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_65:                                         ; preds = %_llgo_213
// CHECK-NEXT:   ret i64 2147483648
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_66:                                         ; preds = %_llgo_213
// CHECK-NEXT:   %67 = call ptr @"{{.*}}/runtime/internal/runtime.IfaceType"(%"{{.*}}/runtime/internal/runtime.iface" %0)
// CHECK-NEXT:   %68 = icmp eq ptr %67, @"*_llgo_go/ast.ImportSpec"
// CHECK-NEXT:   br i1 %68, label %_llgo_214, label %_llgo_215
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_67:                                         ; preds = %_llgo_216
// CHECK-NEXT:   ret i64 4294967296
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_68:                                         ; preds = %_llgo_216
// CHECK-NEXT:   %69 = call ptr @"{{.*}}/runtime/internal/runtime.IfaceType"(%"{{.*}}/runtime/internal/runtime.iface" %0)
// CHECK-NEXT:   %70 = icmp eq ptr %69, @"*_llgo_go/ast.IncDecStmt"
// CHECK-NEXT:   br i1 %70, label %_llgo_217, label %_llgo_218
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_69:                                         ; preds = %_llgo_219
// CHECK-NEXT:   ret i64 8589934592
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_70:                                         ; preds = %_llgo_219
// CHECK-NEXT:   %71 = call ptr @"{{.*}}/runtime/internal/runtime.IfaceType"(%"{{.*}}/runtime/internal/runtime.iface" %0)
// CHECK-NEXT:   %72 = icmp eq ptr %71, @"*_llgo_go/ast.IndexExpr"
// CHECK-NEXT:   br i1 %72, label %_llgo_220, label %_llgo_221
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_71:                                         ; preds = %_llgo_222
// CHECK-NEXT:   ret i64 17179869184
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_72:                                         ; preds = %_llgo_222
// CHECK-NEXT:   %73 = call ptr @"{{.*}}/runtime/internal/runtime.IfaceType"(%"{{.*}}/runtime/internal/runtime.iface" %0)
// CHECK-NEXT:   %74 = icmp eq ptr %73, @"*_llgo_go/ast.IndexListExpr"
// CHECK-NEXT:   br i1 %74, label %_llgo_223, label %_llgo_224
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_73:                                         ; preds = %_llgo_225
// CHECK-NEXT:   ret i64 34359738368
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_74:                                         ; preds = %_llgo_225
// CHECK-NEXT:   %75 = call ptr @"{{.*}}/runtime/internal/runtime.IfaceType"(%"{{.*}}/runtime/internal/runtime.iface" %0)
// CHECK-NEXT:   %76 = icmp eq ptr %75, @"*_llgo_go/ast.InterfaceType"
// CHECK-NEXT:   br i1 %76, label %_llgo_226, label %_llgo_227
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_75:                                         ; preds = %_llgo_228
// CHECK-NEXT:   ret i64 68719476736
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_76:                                         ; preds = %_llgo_228
// CHECK-NEXT:   %77 = call ptr @"{{.*}}/runtime/internal/runtime.IfaceType"(%"{{.*}}/runtime/internal/runtime.iface" %0)
// CHECK-NEXT:   %78 = icmp eq ptr %77, @"*_llgo_go/ast.KeyValueExpr"
// CHECK-NEXT:   br i1 %78, label %_llgo_229, label %_llgo_230
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_77:                                         ; preds = %_llgo_231
// CHECK-NEXT:   ret i64 137438953472
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_78:                                         ; preds = %_llgo_231
// CHECK-NEXT:   %79 = call ptr @"{{.*}}/runtime/internal/runtime.IfaceType"(%"{{.*}}/runtime/internal/runtime.iface" %0)
// CHECK-NEXT:   %80 = icmp eq ptr %79, @"*_llgo_go/ast.LabeledStmt"
// CHECK-NEXT:   br i1 %80, label %_llgo_232, label %_llgo_233
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_79:                                         ; preds = %_llgo_234
// CHECK-NEXT:   ret i64 274877906944
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_80:                                         ; preds = %_llgo_234
// CHECK-NEXT:   %81 = call ptr @"{{.*}}/runtime/internal/runtime.IfaceType"(%"{{.*}}/runtime/internal/runtime.iface" %0)
// CHECK-NEXT:   %82 = icmp eq ptr %81, @"*_llgo_go/ast.MapType"
// CHECK-NEXT:   br i1 %82, label %_llgo_235, label %_llgo_236
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_81:                                         ; preds = %_llgo_237
// CHECK-NEXT:   ret i64 549755813888
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_82:                                         ; preds = %_llgo_237
// CHECK-NEXT:   %83 = call ptr @"{{.*}}/runtime/internal/runtime.IfaceType"(%"{{.*}}/runtime/internal/runtime.iface" %0)
// CHECK-NEXT:   %84 = icmp eq ptr %83, @"*_llgo_go/ast.Package"
// CHECK-NEXT:   br i1 %84, label %_llgo_238, label %_llgo_239
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_83:                                         ; preds = %_llgo_240
// CHECK-NEXT:   ret i64 1099511627776
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_84:                                         ; preds = %_llgo_240
// CHECK-NEXT:   %85 = call ptr @"{{.*}}/runtime/internal/runtime.IfaceType"(%"{{.*}}/runtime/internal/runtime.iface" %0)
// CHECK-NEXT:   %86 = icmp eq ptr %85, @"*_llgo_go/ast.ParenExpr"
// CHECK-NEXT:   br i1 %86, label %_llgo_241, label %_llgo_242
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_85:                                         ; preds = %_llgo_243
// CHECK-NEXT:   ret i64 2199023255552
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_86:                                         ; preds = %_llgo_243
// CHECK-NEXT:   %87 = call ptr @"{{.*}}/runtime/internal/runtime.IfaceType"(%"{{.*}}/runtime/internal/runtime.iface" %0)
// CHECK-NEXT:   %88 = icmp eq ptr %87, @"*_llgo_go/ast.RangeStmt"
// CHECK-NEXT:   br i1 %88, label %_llgo_244, label %_llgo_245
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_87:                                         ; preds = %_llgo_246
// CHECK-NEXT:   ret i64 4398046511104
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_88:                                         ; preds = %_llgo_246
// CHECK-NEXT:   %89 = call ptr @"{{.*}}/runtime/internal/runtime.IfaceType"(%"{{.*}}/runtime/internal/runtime.iface" %0)
// CHECK-NEXT:   %90 = icmp eq ptr %89, @"*_llgo_go/ast.ReturnStmt"
// CHECK-NEXT:   br i1 %90, label %_llgo_247, label %_llgo_248
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_89:                                         ; preds = %_llgo_249
// CHECK-NEXT:   ret i64 8796093022208
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_90:                                         ; preds = %_llgo_249
// CHECK-NEXT:   %91 = call ptr @"{{.*}}/runtime/internal/runtime.IfaceType"(%"{{.*}}/runtime/internal/runtime.iface" %0)
// CHECK-NEXT:   %92 = icmp eq ptr %91, @"*_llgo_go/ast.SelectStmt"
// CHECK-NEXT:   br i1 %92, label %_llgo_250, label %_llgo_251
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_91:                                         ; preds = %_llgo_252
// CHECK-NEXT:   ret i64 17592186044416
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_92:                                         ; preds = %_llgo_252
// CHECK-NEXT:   %93 = call ptr @"{{.*}}/runtime/internal/runtime.IfaceType"(%"{{.*}}/runtime/internal/runtime.iface" %0)
// CHECK-NEXT:   %94 = icmp eq ptr %93, @"*_llgo_go/ast.SelectorExpr"
// CHECK-NEXT:   br i1 %94, label %_llgo_253, label %_llgo_254
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_93:                                         ; preds = %_llgo_255
// CHECK-NEXT:   ret i64 35184372088832
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_94:                                         ; preds = %_llgo_255
// CHECK-NEXT:   %95 = call ptr @"{{.*}}/runtime/internal/runtime.IfaceType"(%"{{.*}}/runtime/internal/runtime.iface" %0)
// CHECK-NEXT:   %96 = icmp eq ptr %95, @"*_llgo_go/ast.SendStmt"
// CHECK-NEXT:   br i1 %96, label %_llgo_256, label %_llgo_257
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_95:                                         ; preds = %_llgo_258
// CHECK-NEXT:   ret i64 70368744177664
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_96:                                         ; preds = %_llgo_258
// CHECK-NEXT:   %97 = call ptr @"{{.*}}/runtime/internal/runtime.IfaceType"(%"{{.*}}/runtime/internal/runtime.iface" %0)
// CHECK-NEXT:   %98 = icmp eq ptr %97, @"*_llgo_go/ast.SliceExpr"
// CHECK-NEXT:   br i1 %98, label %_llgo_259, label %_llgo_260
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_97:                                         ; preds = %_llgo_261
// CHECK-NEXT:   ret i64 140737488355328
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_98:                                         ; preds = %_llgo_261
// CHECK-NEXT:   %99 = call ptr @"{{.*}}/runtime/internal/runtime.IfaceType"(%"{{.*}}/runtime/internal/runtime.iface" %0)
// CHECK-NEXT:   %100 = icmp eq ptr %99, @"*_llgo_go/ast.StarExpr"
// CHECK-NEXT:   br i1 %100, label %_llgo_262, label %_llgo_263
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_99:                                         ; preds = %_llgo_264
// CHECK-NEXT:   ret i64 281474976710656
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_100:                                        ; preds = %_llgo_264
// CHECK-NEXT:   %101 = call ptr @"{{.*}}/runtime/internal/runtime.IfaceType"(%"{{.*}}/runtime/internal/runtime.iface" %0)
// CHECK-NEXT:   %102 = icmp eq ptr %101, @"*_llgo_go/ast.StructType"
// CHECK-NEXT:   br i1 %102, label %_llgo_265, label %_llgo_266
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_101:                                        ; preds = %_llgo_267
// CHECK-NEXT:   ret i64 562949953421312
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_102:                                        ; preds = %_llgo_267
// CHECK-NEXT:   %103 = call ptr @"{{.*}}/runtime/internal/runtime.IfaceType"(%"{{.*}}/runtime/internal/runtime.iface" %0)
// CHECK-NEXT:   %104 = icmp eq ptr %103, @"*_llgo_go/ast.SwitchStmt"
// CHECK-NEXT:   br i1 %104, label %_llgo_268, label %_llgo_269
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_103:                                        ; preds = %_llgo_270
// CHECK-NEXT:   ret i64 1125899906842624
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_104:                                        ; preds = %_llgo_270
// CHECK-NEXT:   %105 = call ptr @"{{.*}}/runtime/internal/runtime.IfaceType"(%"{{.*}}/runtime/internal/runtime.iface" %0)
// CHECK-NEXT:   %106 = icmp eq ptr %105, @"*_llgo_go/ast.TypeAssertExpr"
// CHECK-NEXT:   br i1 %106, label %_llgo_271, label %_llgo_272
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_105:                                        ; preds = %_llgo_273
// CHECK-NEXT:   ret i64 2251799813685248
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_106:                                        ; preds = %_llgo_273
// CHECK-NEXT:   %107 = call ptr @"{{.*}}/runtime/internal/runtime.IfaceType"(%"{{.*}}/runtime/internal/runtime.iface" %0)
// CHECK-NEXT:   %108 = icmp eq ptr %107, @"*_llgo_go/ast.TypeSpec"
// CHECK-NEXT:   br i1 %108, label %_llgo_274, label %_llgo_275
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_107:                                        ; preds = %_llgo_276
// CHECK-NEXT:   ret i64 4503599627370496
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_108:                                        ; preds = %_llgo_276
// CHECK-NEXT:   %109 = call ptr @"{{.*}}/runtime/internal/runtime.IfaceType"(%"{{.*}}/runtime/internal/runtime.iface" %0)
// CHECK-NEXT:   %110 = icmp eq ptr %109, @"*_llgo_go/ast.TypeSwitchStmt"
// CHECK-NEXT:   br i1 %110, label %_llgo_277, label %_llgo_278
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_109:                                        ; preds = %_llgo_279
// CHECK-NEXT:   ret i64 9007199254740992
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_110:                                        ; preds = %_llgo_279
// CHECK-NEXT:   %111 = call ptr @"{{.*}}/runtime/internal/runtime.IfaceType"(%"{{.*}}/runtime/internal/runtime.iface" %0)
// CHECK-NEXT:   %112 = icmp eq ptr %111, @"*_llgo_go/ast.UnaryExpr"
// CHECK-NEXT:   br i1 %112, label %_llgo_280, label %_llgo_281
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_111:                                        ; preds = %_llgo_282
// CHECK-NEXT:   ret i64 18014398509481984
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_112:                                        ; preds = %_llgo_282
// CHECK-NEXT:   %113 = call ptr @"{{.*}}/runtime/internal/runtime.IfaceType"(%"{{.*}}/runtime/internal/runtime.iface" %0)
// CHECK-NEXT:   %114 = icmp eq ptr %113, @"*_llgo_go/ast.ValueSpec"
// CHECK-NEXT:   br i1 %114, label %_llgo_283, label %_llgo_284
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_113:                                        ; preds = %_llgo_285
// CHECK-NEXT:   ret i64 36028797018963968
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_114:                                        ; preds = %_llgo_285
// CHECK-NEXT:   ret i64 0
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_115:                                        ; preds = %_llgo_0
// CHECK-NEXT:   %115 = extractvalue %"{{.*}}/runtime/internal/runtime.iface" %0, 1
// CHECK-NEXT:   %116 = insertvalue { ptr, i1 } undef, ptr %115, 0
// CHECK-NEXT:   %117 = insertvalue { ptr, i1 } %116, i1 true, 1
// CHECK-NEXT:   br label %_llgo_117
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_116:                                        ; preds = %_llgo_0
// CHECK-NEXT:   br label %_llgo_117
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_117:                                        ; preds = %_llgo_116, %_llgo_115
// CHECK-NEXT:   %118 = phi { ptr, i1 } [ %117, %_llgo_115 ], [ zeroinitializer, %_llgo_116 ]
// CHECK-NEXT:   %119 = extractvalue { ptr, i1 } %118, 0
// CHECK-NEXT:   %120 = extractvalue { ptr, i1 } %118, 1
// CHECK-NEXT:   br i1 %120, label %_llgo_1, label %_llgo_2
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_118:                                        ; preds = %_llgo_2
// CHECK-NEXT:   %121 = extractvalue %"{{.*}}/runtime/internal/runtime.iface" %0, 1
// CHECK-NEXT:   %122 = insertvalue { ptr, i1 } undef, ptr %121, 0
// CHECK-NEXT:   %123 = insertvalue { ptr, i1 } %122, i1 true, 1
// CHECK-NEXT:   br label %_llgo_120
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_119:                                        ; preds = %_llgo_2
// CHECK-NEXT:   br label %_llgo_120
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_120:                                        ; preds = %_llgo_119, %_llgo_118
// CHECK-NEXT:   %124 = phi { ptr, i1 } [ %123, %_llgo_118 ], [ zeroinitializer, %_llgo_119 ]
// CHECK-NEXT:   %125 = extractvalue { ptr, i1 } %124, 0
// CHECK-NEXT:   %126 = extractvalue { ptr, i1 } %124, 1
// CHECK-NEXT:   br i1 %126, label %_llgo_3, label %_llgo_4
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_121:                                        ; preds = %_llgo_4
// CHECK-NEXT:   %127 = extractvalue %"{{.*}}/runtime/internal/runtime.iface" %0, 1
// CHECK-NEXT:   %128 = insertvalue { ptr, i1 } undef, ptr %127, 0
// CHECK-NEXT:   %129 = insertvalue { ptr, i1 } %128, i1 true, 1
// CHECK-NEXT:   br label %_llgo_123
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_122:                                        ; preds = %_llgo_4
// CHECK-NEXT:   br label %_llgo_123
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_123:                                        ; preds = %_llgo_122, %_llgo_121
// CHECK-NEXT:   %130 = phi { ptr, i1 } [ %129, %_llgo_121 ], [ zeroinitializer, %_llgo_122 ]
// CHECK-NEXT:   %131 = extractvalue { ptr, i1 } %130, 0
// CHECK-NEXT:   %132 = extractvalue { ptr, i1 } %130, 1
// CHECK-NEXT:   br i1 %132, label %_llgo_5, label %_llgo_6
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_124:                                        ; preds = %_llgo_6
// CHECK-NEXT:   %133 = extractvalue %"{{.*}}/runtime/internal/runtime.iface" %0, 1
// CHECK-NEXT:   %134 = insertvalue { ptr, i1 } undef, ptr %133, 0
// CHECK-NEXT:   %135 = insertvalue { ptr, i1 } %134, i1 true, 1
// CHECK-NEXT:   br label %_llgo_126
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_125:                                        ; preds = %_llgo_6
// CHECK-NEXT:   br label %_llgo_126
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_126:                                        ; preds = %_llgo_125, %_llgo_124
// CHECK-NEXT:   %136 = phi { ptr, i1 } [ %135, %_llgo_124 ], [ zeroinitializer, %_llgo_125 ]
// CHECK-NEXT:   %137 = extractvalue { ptr, i1 } %136, 0
// CHECK-NEXT:   %138 = extractvalue { ptr, i1 } %136, 1
// CHECK-NEXT:   br i1 %138, label %_llgo_7, label %_llgo_8
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_127:                                        ; preds = %_llgo_8
// CHECK-NEXT:   %139 = extractvalue %"{{.*}}/runtime/internal/runtime.iface" %0, 1
// CHECK-NEXT:   %140 = insertvalue { ptr, i1 } undef, ptr %139, 0
// CHECK-NEXT:   %141 = insertvalue { ptr, i1 } %140, i1 true, 1
// CHECK-NEXT:   br label %_llgo_129
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_128:                                        ; preds = %_llgo_8
// CHECK-NEXT:   br label %_llgo_129
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_129:                                        ; preds = %_llgo_128, %_llgo_127
// CHECK-NEXT:   %142 = phi { ptr, i1 } [ %141, %_llgo_127 ], [ zeroinitializer, %_llgo_128 ]
// CHECK-NEXT:   %143 = extractvalue { ptr, i1 } %142, 0
// CHECK-NEXT:   %144 = extractvalue { ptr, i1 } %142, 1
// CHECK-NEXT:   br i1 %144, label %_llgo_9, label %_llgo_10
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_130:                                        ; preds = %_llgo_10
// CHECK-NEXT:   %145 = extractvalue %"{{.*}}/runtime/internal/runtime.iface" %0, 1
// CHECK-NEXT:   %146 = insertvalue { ptr, i1 } undef, ptr %145, 0
// CHECK-NEXT:   %147 = insertvalue { ptr, i1 } %146, i1 true, 1
// CHECK-NEXT:   br label %_llgo_132
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_131:                                        ; preds = %_llgo_10
// CHECK-NEXT:   br label %_llgo_132
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_132:                                        ; preds = %_llgo_131, %_llgo_130
// CHECK-NEXT:   %148 = phi { ptr, i1 } [ %147, %_llgo_130 ], [ zeroinitializer, %_llgo_131 ]
// CHECK-NEXT:   %149 = extractvalue { ptr, i1 } %148, 0
// CHECK-NEXT:   %150 = extractvalue { ptr, i1 } %148, 1
// CHECK-NEXT:   br i1 %150, label %_llgo_11, label %_llgo_12
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_133:                                        ; preds = %_llgo_12
// CHECK-NEXT:   %151 = extractvalue %"{{.*}}/runtime/internal/runtime.iface" %0, 1
// CHECK-NEXT:   %152 = insertvalue { ptr, i1 } undef, ptr %151, 0
// CHECK-NEXT:   %153 = insertvalue { ptr, i1 } %152, i1 true, 1
// CHECK-NEXT:   br label %_llgo_135
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_134:                                        ; preds = %_llgo_12
// CHECK-NEXT:   br label %_llgo_135
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_135:                                        ; preds = %_llgo_134, %_llgo_133
// CHECK-NEXT:   %154 = phi { ptr, i1 } [ %153, %_llgo_133 ], [ zeroinitializer, %_llgo_134 ]
// CHECK-NEXT:   %155 = extractvalue { ptr, i1 } %154, 0
// CHECK-NEXT:   %156 = extractvalue { ptr, i1 } %154, 1
// CHECK-NEXT:   br i1 %156, label %_llgo_13, label %_llgo_14
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_136:                                        ; preds = %_llgo_14
// CHECK-NEXT:   %157 = extractvalue %"{{.*}}/runtime/internal/runtime.iface" %0, 1
// CHECK-NEXT:   %158 = insertvalue { ptr, i1 } undef, ptr %157, 0
// CHECK-NEXT:   %159 = insertvalue { ptr, i1 } %158, i1 true, 1
// CHECK-NEXT:   br label %_llgo_138
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_137:                                        ; preds = %_llgo_14
// CHECK-NEXT:   br label %_llgo_138
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_138:                                        ; preds = %_llgo_137, %_llgo_136
// CHECK-NEXT:   %160 = phi { ptr, i1 } [ %159, %_llgo_136 ], [ zeroinitializer, %_llgo_137 ]
// CHECK-NEXT:   %161 = extractvalue { ptr, i1 } %160, 0
// CHECK-NEXT:   %162 = extractvalue { ptr, i1 } %160, 1
// CHECK-NEXT:   br i1 %162, label %_llgo_15, label %_llgo_16
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_139:                                        ; preds = %_llgo_16
// CHECK-NEXT:   %163 = extractvalue %"{{.*}}/runtime/internal/runtime.iface" %0, 1
// CHECK-NEXT:   %164 = insertvalue { ptr, i1 } undef, ptr %163, 0
// CHECK-NEXT:   %165 = insertvalue { ptr, i1 } %164, i1 true, 1
// CHECK-NEXT:   br label %_llgo_141
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_140:                                        ; preds = %_llgo_16
// CHECK-NEXT:   br label %_llgo_141
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_141:                                        ; preds = %_llgo_140, %_llgo_139
// CHECK-NEXT:   %166 = phi { ptr, i1 } [ %165, %_llgo_139 ], [ zeroinitializer, %_llgo_140 ]
// CHECK-NEXT:   %167 = extractvalue { ptr, i1 } %166, 0
// CHECK-NEXT:   %168 = extractvalue { ptr, i1 } %166, 1
// CHECK-NEXT:   br i1 %168, label %_llgo_17, label %_llgo_18
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_142:                                        ; preds = %_llgo_18
// CHECK-NEXT:   %169 = extractvalue %"{{.*}}/runtime/internal/runtime.iface" %0, 1
// CHECK-NEXT:   %170 = insertvalue { ptr, i1 } undef, ptr %169, 0
// CHECK-NEXT:   %171 = insertvalue { ptr, i1 } %170, i1 true, 1
// CHECK-NEXT:   br label %_llgo_144
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_143:                                        ; preds = %_llgo_18
// CHECK-NEXT:   br label %_llgo_144
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_144:                                        ; preds = %_llgo_143, %_llgo_142
// CHECK-NEXT:   %172 = phi { ptr, i1 } [ %171, %_llgo_142 ], [ zeroinitializer, %_llgo_143 ]
// CHECK-NEXT:   %173 = extractvalue { ptr, i1 } %172, 0
// CHECK-NEXT:   %174 = extractvalue { ptr, i1 } %172, 1
// CHECK-NEXT:   br i1 %174, label %_llgo_19, label %_llgo_20
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_145:                                        ; preds = %_llgo_20
// CHECK-NEXT:   %175 = extractvalue %"{{.*}}/runtime/internal/runtime.iface" %0, 1
// CHECK-NEXT:   %176 = insertvalue { ptr, i1 } undef, ptr %175, 0
// CHECK-NEXT:   %177 = insertvalue { ptr, i1 } %176, i1 true, 1
// CHECK-NEXT:   br label %_llgo_147
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_146:                                        ; preds = %_llgo_20
// CHECK-NEXT:   br label %_llgo_147
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_147:                                        ; preds = %_llgo_146, %_llgo_145
// CHECK-NEXT:   %178 = phi { ptr, i1 } [ %177, %_llgo_145 ], [ zeroinitializer, %_llgo_146 ]
// CHECK-NEXT:   %179 = extractvalue { ptr, i1 } %178, 0
// CHECK-NEXT:   %180 = extractvalue { ptr, i1 } %178, 1
// CHECK-NEXT:   br i1 %180, label %_llgo_21, label %_llgo_22
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_148:                                        ; preds = %_llgo_22
// CHECK-NEXT:   %181 = extractvalue %"{{.*}}/runtime/internal/runtime.iface" %0, 1
// CHECK-NEXT:   %182 = insertvalue { ptr, i1 } undef, ptr %181, 0
// CHECK-NEXT:   %183 = insertvalue { ptr, i1 } %182, i1 true, 1
// CHECK-NEXT:   br label %_llgo_150
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_149:                                        ; preds = %_llgo_22
// CHECK-NEXT:   br label %_llgo_150
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_150:                                        ; preds = %_llgo_149, %_llgo_148
// CHECK-NEXT:   %184 = phi { ptr, i1 } [ %183, %_llgo_148 ], [ zeroinitializer, %_llgo_149 ]
// CHECK-NEXT:   %185 = extractvalue { ptr, i1 } %184, 0
// CHECK-NEXT:   %186 = extractvalue { ptr, i1 } %184, 1
// CHECK-NEXT:   br i1 %186, label %_llgo_23, label %_llgo_24
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_151:                                        ; preds = %_llgo_24
// CHECK-NEXT:   %187 = extractvalue %"{{.*}}/runtime/internal/runtime.iface" %0, 1
// CHECK-NEXT:   %188 = insertvalue { ptr, i1 } undef, ptr %187, 0
// CHECK-NEXT:   %189 = insertvalue { ptr, i1 } %188, i1 true, 1
// CHECK-NEXT:   br label %_llgo_153
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_152:                                        ; preds = %_llgo_24
// CHECK-NEXT:   br label %_llgo_153
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_153:                                        ; preds = %_llgo_152, %_llgo_151
// CHECK-NEXT:   %190 = phi { ptr, i1 } [ %189, %_llgo_151 ], [ zeroinitializer, %_llgo_152 ]
// CHECK-NEXT:   %191 = extractvalue { ptr, i1 } %190, 0
// CHECK-NEXT:   %192 = extractvalue { ptr, i1 } %190, 1
// CHECK-NEXT:   br i1 %192, label %_llgo_25, label %_llgo_26
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_154:                                        ; preds = %_llgo_26
// CHECK-NEXT:   %193 = extractvalue %"{{.*}}/runtime/internal/runtime.iface" %0, 1
// CHECK-NEXT:   %194 = insertvalue { ptr, i1 } undef, ptr %193, 0
// CHECK-NEXT:   %195 = insertvalue { ptr, i1 } %194, i1 true, 1
// CHECK-NEXT:   br label %_llgo_156
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_155:                                        ; preds = %_llgo_26
// CHECK-NEXT:   br label %_llgo_156
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_156:                                        ; preds = %_llgo_155, %_llgo_154
// CHECK-NEXT:   %196 = phi { ptr, i1 } [ %195, %_llgo_154 ], [ zeroinitializer, %_llgo_155 ]
// CHECK-NEXT:   %197 = extractvalue { ptr, i1 } %196, 0
// CHECK-NEXT:   %198 = extractvalue { ptr, i1 } %196, 1
// CHECK-NEXT:   br i1 %198, label %_llgo_27, label %_llgo_28
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_157:                                        ; preds = %_llgo_28
// CHECK-NEXT:   %199 = extractvalue %"{{.*}}/runtime/internal/runtime.iface" %0, 1
// CHECK-NEXT:   %200 = insertvalue { ptr, i1 } undef, ptr %199, 0
// CHECK-NEXT:   %201 = insertvalue { ptr, i1 } %200, i1 true, 1
// CHECK-NEXT:   br label %_llgo_159
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_158:                                        ; preds = %_llgo_28
// CHECK-NEXT:   br label %_llgo_159
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_159:                                        ; preds = %_llgo_158, %_llgo_157
// CHECK-NEXT:   %202 = phi { ptr, i1 } [ %201, %_llgo_157 ], [ zeroinitializer, %_llgo_158 ]
// CHECK-NEXT:   %203 = extractvalue { ptr, i1 } %202, 0
// CHECK-NEXT:   %204 = extractvalue { ptr, i1 } %202, 1
// CHECK-NEXT:   br i1 %204, label %_llgo_29, label %_llgo_30
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_160:                                        ; preds = %_llgo_30
// CHECK-NEXT:   %205 = extractvalue %"{{.*}}/runtime/internal/runtime.iface" %0, 1
// CHECK-NEXT:   %206 = insertvalue { ptr, i1 } undef, ptr %205, 0
// CHECK-NEXT:   %207 = insertvalue { ptr, i1 } %206, i1 true, 1
// CHECK-NEXT:   br label %_llgo_162
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_161:                                        ; preds = %_llgo_30
// CHECK-NEXT:   br label %_llgo_162
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_162:                                        ; preds = %_llgo_161, %_llgo_160
// CHECK-NEXT:   %208 = phi { ptr, i1 } [ %207, %_llgo_160 ], [ zeroinitializer, %_llgo_161 ]
// CHECK-NEXT:   %209 = extractvalue { ptr, i1 } %208, 0
// CHECK-NEXT:   %210 = extractvalue { ptr, i1 } %208, 1
// CHECK-NEXT:   br i1 %210, label %_llgo_31, label %_llgo_32
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_163:                                        ; preds = %_llgo_32
// CHECK-NEXT:   %211 = extractvalue %"{{.*}}/runtime/internal/runtime.iface" %0, 1
// CHECK-NEXT:   %212 = insertvalue { ptr, i1 } undef, ptr %211, 0
// CHECK-NEXT:   %213 = insertvalue { ptr, i1 } %212, i1 true, 1
// CHECK-NEXT:   br label %_llgo_165
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_164:                                        ; preds = %_llgo_32
// CHECK-NEXT:   br label %_llgo_165
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_165:                                        ; preds = %_llgo_164, %_llgo_163
// CHECK-NEXT:   %214 = phi { ptr, i1 } [ %213, %_llgo_163 ], [ zeroinitializer, %_llgo_164 ]
// CHECK-NEXT:   %215 = extractvalue { ptr, i1 } %214, 0
// CHECK-NEXT:   %216 = extractvalue { ptr, i1 } %214, 1
// CHECK-NEXT:   br i1 %216, label %_llgo_33, label %_llgo_34
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_166:                                        ; preds = %_llgo_34
// CHECK-NEXT:   %217 = extractvalue %"{{.*}}/runtime/internal/runtime.iface" %0, 1
// CHECK-NEXT:   %218 = insertvalue { ptr, i1 } undef, ptr %217, 0
// CHECK-NEXT:   %219 = insertvalue { ptr, i1 } %218, i1 true, 1
// CHECK-NEXT:   br label %_llgo_168
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_167:                                        ; preds = %_llgo_34
// CHECK-NEXT:   br label %_llgo_168
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_168:                                        ; preds = %_llgo_167, %_llgo_166
// CHECK-NEXT:   %220 = phi { ptr, i1 } [ %219, %_llgo_166 ], [ zeroinitializer, %_llgo_167 ]
// CHECK-NEXT:   %221 = extractvalue { ptr, i1 } %220, 0
// CHECK-NEXT:   %222 = extractvalue { ptr, i1 } %220, 1
// CHECK-NEXT:   br i1 %222, label %_llgo_35, label %_llgo_36
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_169:                                        ; preds = %_llgo_36
// CHECK-NEXT:   %223 = extractvalue %"{{.*}}/runtime/internal/runtime.iface" %0, 1
// CHECK-NEXT:   %224 = insertvalue { ptr, i1 } undef, ptr %223, 0
// CHECK-NEXT:   %225 = insertvalue { ptr, i1 } %224, i1 true, 1
// CHECK-NEXT:   br label %_llgo_171
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_170:                                        ; preds = %_llgo_36
// CHECK-NEXT:   br label %_llgo_171
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_171:                                        ; preds = %_llgo_170, %_llgo_169
// CHECK-NEXT:   %226 = phi { ptr, i1 } [ %225, %_llgo_169 ], [ zeroinitializer, %_llgo_170 ]
// CHECK-NEXT:   %227 = extractvalue { ptr, i1 } %226, 0
// CHECK-NEXT:   %228 = extractvalue { ptr, i1 } %226, 1
// CHECK-NEXT:   br i1 %228, label %_llgo_37, label %_llgo_38
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_172:                                        ; preds = %_llgo_38
// CHECK-NEXT:   %229 = extractvalue %"{{.*}}/runtime/internal/runtime.iface" %0, 1
// CHECK-NEXT:   %230 = insertvalue { ptr, i1 } undef, ptr %229, 0
// CHECK-NEXT:   %231 = insertvalue { ptr, i1 } %230, i1 true, 1
// CHECK-NEXT:   br label %_llgo_174
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_173:                                        ; preds = %_llgo_38
// CHECK-NEXT:   br label %_llgo_174
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_174:                                        ; preds = %_llgo_173, %_llgo_172
// CHECK-NEXT:   %232 = phi { ptr, i1 } [ %231, %_llgo_172 ], [ zeroinitializer, %_llgo_173 ]
// CHECK-NEXT:   %233 = extractvalue { ptr, i1 } %232, 0
// CHECK-NEXT:   %234 = extractvalue { ptr, i1 } %232, 1
// CHECK-NEXT:   br i1 %234, label %_llgo_39, label %_llgo_40
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_175:                                        ; preds = %_llgo_40
// CHECK-NEXT:   %235 = extractvalue %"{{.*}}/runtime/internal/runtime.iface" %0, 1
// CHECK-NEXT:   %236 = insertvalue { ptr, i1 } undef, ptr %235, 0
// CHECK-NEXT:   %237 = insertvalue { ptr, i1 } %236, i1 true, 1
// CHECK-NEXT:   br label %_llgo_177
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_176:                                        ; preds = %_llgo_40
// CHECK-NEXT:   br label %_llgo_177
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_177:                                        ; preds = %_llgo_176, %_llgo_175
// CHECK-NEXT:   %238 = phi { ptr, i1 } [ %237, %_llgo_175 ], [ zeroinitializer, %_llgo_176 ]
// CHECK-NEXT:   %239 = extractvalue { ptr, i1 } %238, 0
// CHECK-NEXT:   %240 = extractvalue { ptr, i1 } %238, 1
// CHECK-NEXT:   br i1 %240, label %_llgo_41, label %_llgo_42
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_178:                                        ; preds = %_llgo_42
// CHECK-NEXT:   %241 = extractvalue %"{{.*}}/runtime/internal/runtime.iface" %0, 1
// CHECK-NEXT:   %242 = insertvalue { ptr, i1 } undef, ptr %241, 0
// CHECK-NEXT:   %243 = insertvalue { ptr, i1 } %242, i1 true, 1
// CHECK-NEXT:   br label %_llgo_180
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_179:                                        ; preds = %_llgo_42
// CHECK-NEXT:   br label %_llgo_180
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_180:                                        ; preds = %_llgo_179, %_llgo_178
// CHECK-NEXT:   %244 = phi { ptr, i1 } [ %243, %_llgo_178 ], [ zeroinitializer, %_llgo_179 ]
// CHECK-NEXT:   %245 = extractvalue { ptr, i1 } %244, 0
// CHECK-NEXT:   %246 = extractvalue { ptr, i1 } %244, 1
// CHECK-NEXT:   br i1 %246, label %_llgo_43, label %_llgo_44
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_181:                                        ; preds = %_llgo_44
// CHECK-NEXT:   %247 = extractvalue %"{{.*}}/runtime/internal/runtime.iface" %0, 1
// CHECK-NEXT:   %248 = insertvalue { ptr, i1 } undef, ptr %247, 0
// CHECK-NEXT:   %249 = insertvalue { ptr, i1 } %248, i1 true, 1
// CHECK-NEXT:   br label %_llgo_183
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_182:                                        ; preds = %_llgo_44
// CHECK-NEXT:   br label %_llgo_183
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_183:                                        ; preds = %_llgo_182, %_llgo_181
// CHECK-NEXT:   %250 = phi { ptr, i1 } [ %249, %_llgo_181 ], [ zeroinitializer, %_llgo_182 ]
// CHECK-NEXT:   %251 = extractvalue { ptr, i1 } %250, 0
// CHECK-NEXT:   %252 = extractvalue { ptr, i1 } %250, 1
// CHECK-NEXT:   br i1 %252, label %_llgo_45, label %_llgo_46
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_184:                                        ; preds = %_llgo_46
// CHECK-NEXT:   %253 = extractvalue %"{{.*}}/runtime/internal/runtime.iface" %0, 1
// CHECK-NEXT:   %254 = insertvalue { ptr, i1 } undef, ptr %253, 0
// CHECK-NEXT:   %255 = insertvalue { ptr, i1 } %254, i1 true, 1
// CHECK-NEXT:   br label %_llgo_186
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_185:                                        ; preds = %_llgo_46
// CHECK-NEXT:   br label %_llgo_186
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_186:                                        ; preds = %_llgo_185, %_llgo_184
// CHECK-NEXT:   %256 = phi { ptr, i1 } [ %255, %_llgo_184 ], [ zeroinitializer, %_llgo_185 ]
// CHECK-NEXT:   %257 = extractvalue { ptr, i1 } %256, 0
// CHECK-NEXT:   %258 = extractvalue { ptr, i1 } %256, 1
// CHECK-NEXT:   br i1 %258, label %_llgo_47, label %_llgo_48
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_187:                                        ; preds = %_llgo_48
// CHECK-NEXT:   %259 = extractvalue %"{{.*}}/runtime/internal/runtime.iface" %0, 1
// CHECK-NEXT:   %260 = insertvalue { ptr, i1 } undef, ptr %259, 0
// CHECK-NEXT:   %261 = insertvalue { ptr, i1 } %260, i1 true, 1
// CHECK-NEXT:   br label %_llgo_189
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_188:                                        ; preds = %_llgo_48
// CHECK-NEXT:   br label %_llgo_189
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_189:                                        ; preds = %_llgo_188, %_llgo_187
// CHECK-NEXT:   %262 = phi { ptr, i1 } [ %261, %_llgo_187 ], [ zeroinitializer, %_llgo_188 ]
// CHECK-NEXT:   %263 = extractvalue { ptr, i1 } %262, 0
// CHECK-NEXT:   %264 = extractvalue { ptr, i1 } %262, 1
// CHECK-NEXT:   br i1 %264, label %_llgo_49, label %_llgo_50
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_190:                                        ; preds = %_llgo_50
// CHECK-NEXT:   %265 = extractvalue %"{{.*}}/runtime/internal/runtime.iface" %0, 1
// CHECK-NEXT:   %266 = insertvalue { ptr, i1 } undef, ptr %265, 0
// CHECK-NEXT:   %267 = insertvalue { ptr, i1 } %266, i1 true, 1
// CHECK-NEXT:   br label %_llgo_192
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_191:                                        ; preds = %_llgo_50
// CHECK-NEXT:   br label %_llgo_192
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_192:                                        ; preds = %_llgo_191, %_llgo_190
// CHECK-NEXT:   %268 = phi { ptr, i1 } [ %267, %_llgo_190 ], [ zeroinitializer, %_llgo_191 ]
// CHECK-NEXT:   %269 = extractvalue { ptr, i1 } %268, 0
// CHECK-NEXT:   %270 = extractvalue { ptr, i1 } %268, 1
// CHECK-NEXT:   br i1 %270, label %_llgo_51, label %_llgo_52
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_193:                                        ; preds = %_llgo_52
// CHECK-NEXT:   %271 = extractvalue %"{{.*}}/runtime/internal/runtime.iface" %0, 1
// CHECK-NEXT:   %272 = insertvalue { ptr, i1 } undef, ptr %271, 0
// CHECK-NEXT:   %273 = insertvalue { ptr, i1 } %272, i1 true, 1
// CHECK-NEXT:   br label %_llgo_195
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_194:                                        ; preds = %_llgo_52
// CHECK-NEXT:   br label %_llgo_195
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_195:                                        ; preds = %_llgo_194, %_llgo_193
// CHECK-NEXT:   %274 = phi { ptr, i1 } [ %273, %_llgo_193 ], [ zeroinitializer, %_llgo_194 ]
// CHECK-NEXT:   %275 = extractvalue { ptr, i1 } %274, 0
// CHECK-NEXT:   %276 = extractvalue { ptr, i1 } %274, 1
// CHECK-NEXT:   br i1 %276, label %_llgo_53, label %_llgo_54
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_196:                                        ; preds = %_llgo_54
// CHECK-NEXT:   %277 = extractvalue %"{{.*}}/runtime/internal/runtime.iface" %0, 1
// CHECK-NEXT:   %278 = insertvalue { ptr, i1 } undef, ptr %277, 0
// CHECK-NEXT:   %279 = insertvalue { ptr, i1 } %278, i1 true, 1
// CHECK-NEXT:   br label %_llgo_198
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_197:                                        ; preds = %_llgo_54
// CHECK-NEXT:   br label %_llgo_198
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_198:                                        ; preds = %_llgo_197, %_llgo_196
// CHECK-NEXT:   %280 = phi { ptr, i1 } [ %279, %_llgo_196 ], [ zeroinitializer, %_llgo_197 ]
// CHECK-NEXT:   %281 = extractvalue { ptr, i1 } %280, 0
// CHECK-NEXT:   %282 = extractvalue { ptr, i1 } %280, 1
// CHECK-NEXT:   br i1 %282, label %_llgo_55, label %_llgo_56
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_199:                                        ; preds = %_llgo_56
// CHECK-NEXT:   %283 = extractvalue %"{{.*}}/runtime/internal/runtime.iface" %0, 1
// CHECK-NEXT:   %284 = insertvalue { ptr, i1 } undef, ptr %283, 0
// CHECK-NEXT:   %285 = insertvalue { ptr, i1 } %284, i1 true, 1
// CHECK-NEXT:   br label %_llgo_201
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_200:                                        ; preds = %_llgo_56
// CHECK-NEXT:   br label %_llgo_201
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_201:                                        ; preds = %_llgo_200, %_llgo_199
// CHECK-NEXT:   %286 = phi { ptr, i1 } [ %285, %_llgo_199 ], [ zeroinitializer, %_llgo_200 ]
// CHECK-NEXT:   %287 = extractvalue { ptr, i1 } %286, 0
// CHECK-NEXT:   %288 = extractvalue { ptr, i1 } %286, 1
// CHECK-NEXT:   br i1 %288, label %_llgo_57, label %_llgo_58
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_202:                                        ; preds = %_llgo_58
// CHECK-NEXT:   %289 = extractvalue %"{{.*}}/runtime/internal/runtime.iface" %0, 1
// CHECK-NEXT:   %290 = insertvalue { ptr, i1 } undef, ptr %289, 0
// CHECK-NEXT:   %291 = insertvalue { ptr, i1 } %290, i1 true, 1
// CHECK-NEXT:   br label %_llgo_204
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_203:                                        ; preds = %_llgo_58
// CHECK-NEXT:   br label %_llgo_204
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_204:                                        ; preds = %_llgo_203, %_llgo_202
// CHECK-NEXT:   %292 = phi { ptr, i1 } [ %291, %_llgo_202 ], [ zeroinitializer, %_llgo_203 ]
// CHECK-NEXT:   %293 = extractvalue { ptr, i1 } %292, 0
// CHECK-NEXT:   %294 = extractvalue { ptr, i1 } %292, 1
// CHECK-NEXT:   br i1 %294, label %_llgo_59, label %_llgo_60
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_205:                                        ; preds = %_llgo_60
// CHECK-NEXT:   %295 = extractvalue %"{{.*}}/runtime/internal/runtime.iface" %0, 1
// CHECK-NEXT:   %296 = insertvalue { ptr, i1 } undef, ptr %295, 0
// CHECK-NEXT:   %297 = insertvalue { ptr, i1 } %296, i1 true, 1
// CHECK-NEXT:   br label %_llgo_207
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_206:                                        ; preds = %_llgo_60
// CHECK-NEXT:   br label %_llgo_207
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_207:                                        ; preds = %_llgo_206, %_llgo_205
// CHECK-NEXT:   %298 = phi { ptr, i1 } [ %297, %_llgo_205 ], [ zeroinitializer, %_llgo_206 ]
// CHECK-NEXT:   %299 = extractvalue { ptr, i1 } %298, 0
// CHECK-NEXT:   %300 = extractvalue { ptr, i1 } %298, 1
// CHECK-NEXT:   br i1 %300, label %_llgo_61, label %_llgo_62
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_208:                                        ; preds = %_llgo_62
// CHECK-NEXT:   %301 = extractvalue %"{{.*}}/runtime/internal/runtime.iface" %0, 1
// CHECK-NEXT:   %302 = insertvalue { ptr, i1 } undef, ptr %301, 0
// CHECK-NEXT:   %303 = insertvalue { ptr, i1 } %302, i1 true, 1
// CHECK-NEXT:   br label %_llgo_210
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_209:                                        ; preds = %_llgo_62
// CHECK-NEXT:   br label %_llgo_210
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_210:                                        ; preds = %_llgo_209, %_llgo_208
// CHECK-NEXT:   %304 = phi { ptr, i1 } [ %303, %_llgo_208 ], [ zeroinitializer, %_llgo_209 ]
// CHECK-NEXT:   %305 = extractvalue { ptr, i1 } %304, 0
// CHECK-NEXT:   %306 = extractvalue { ptr, i1 } %304, 1
// CHECK-NEXT:   br i1 %306, label %_llgo_63, label %_llgo_64
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_211:                                        ; preds = %_llgo_64
// CHECK-NEXT:   %307 = extractvalue %"{{.*}}/runtime/internal/runtime.iface" %0, 1
// CHECK-NEXT:   %308 = insertvalue { ptr, i1 } undef, ptr %307, 0
// CHECK-NEXT:   %309 = insertvalue { ptr, i1 } %308, i1 true, 1
// CHECK-NEXT:   br label %_llgo_213
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_212:                                        ; preds = %_llgo_64
// CHECK-NEXT:   br label %_llgo_213
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_213:                                        ; preds = %_llgo_212, %_llgo_211
// CHECK-NEXT:   %310 = phi { ptr, i1 } [ %309, %_llgo_211 ], [ zeroinitializer, %_llgo_212 ]
// CHECK-NEXT:   %311 = extractvalue { ptr, i1 } %310, 0
// CHECK-NEXT:   %312 = extractvalue { ptr, i1 } %310, 1
// CHECK-NEXT:   br i1 %312, label %_llgo_65, label %_llgo_66
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_214:                                        ; preds = %_llgo_66
// CHECK-NEXT:   %313 = extractvalue %"{{.*}}/runtime/internal/runtime.iface" %0, 1
// CHECK-NEXT:   %314 = insertvalue { ptr, i1 } undef, ptr %313, 0
// CHECK-NEXT:   %315 = insertvalue { ptr, i1 } %314, i1 true, 1
// CHECK-NEXT:   br label %_llgo_216
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_215:                                        ; preds = %_llgo_66
// CHECK-NEXT:   br label %_llgo_216
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_216:                                        ; preds = %_llgo_215, %_llgo_214
// CHECK-NEXT:   %316 = phi { ptr, i1 } [ %315, %_llgo_214 ], [ zeroinitializer, %_llgo_215 ]
// CHECK-NEXT:   %317 = extractvalue { ptr, i1 } %316, 0
// CHECK-NEXT:   %318 = extractvalue { ptr, i1 } %316, 1
// CHECK-NEXT:   br i1 %318, label %_llgo_67, label %_llgo_68
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_217:                                        ; preds = %_llgo_68
// CHECK-NEXT:   %319 = extractvalue %"{{.*}}/runtime/internal/runtime.iface" %0, 1
// CHECK-NEXT:   %320 = insertvalue { ptr, i1 } undef, ptr %319, 0
// CHECK-NEXT:   %321 = insertvalue { ptr, i1 } %320, i1 true, 1
// CHECK-NEXT:   br label %_llgo_219
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_218:                                        ; preds = %_llgo_68
// CHECK-NEXT:   br label %_llgo_219
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_219:                                        ; preds = %_llgo_218, %_llgo_217
// CHECK-NEXT:   %322 = phi { ptr, i1 } [ %321, %_llgo_217 ], [ zeroinitializer, %_llgo_218 ]
// CHECK-NEXT:   %323 = extractvalue { ptr, i1 } %322, 0
// CHECK-NEXT:   %324 = extractvalue { ptr, i1 } %322, 1
// CHECK-NEXT:   br i1 %324, label %_llgo_69, label %_llgo_70
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_220:                                        ; preds = %_llgo_70
// CHECK-NEXT:   %325 = extractvalue %"{{.*}}/runtime/internal/runtime.iface" %0, 1
// CHECK-NEXT:   %326 = insertvalue { ptr, i1 } undef, ptr %325, 0
// CHECK-NEXT:   %327 = insertvalue { ptr, i1 } %326, i1 true, 1
// CHECK-NEXT:   br label %_llgo_222
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_221:                                        ; preds = %_llgo_70
// CHECK-NEXT:   br label %_llgo_222
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_222:                                        ; preds = %_llgo_221, %_llgo_220
// CHECK-NEXT:   %328 = phi { ptr, i1 } [ %327, %_llgo_220 ], [ zeroinitializer, %_llgo_221 ]
// CHECK-NEXT:   %329 = extractvalue { ptr, i1 } %328, 0
// CHECK-NEXT:   %330 = extractvalue { ptr, i1 } %328, 1
// CHECK-NEXT:   br i1 %330, label %_llgo_71, label %_llgo_72
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_223:                                        ; preds = %_llgo_72
// CHECK-NEXT:   %331 = extractvalue %"{{.*}}/runtime/internal/runtime.iface" %0, 1
// CHECK-NEXT:   %332 = insertvalue { ptr, i1 } undef, ptr %331, 0
// CHECK-NEXT:   %333 = insertvalue { ptr, i1 } %332, i1 true, 1
// CHECK-NEXT:   br label %_llgo_225
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_224:                                        ; preds = %_llgo_72
// CHECK-NEXT:   br label %_llgo_225
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_225:                                        ; preds = %_llgo_224, %_llgo_223
// CHECK-NEXT:   %334 = phi { ptr, i1 } [ %333, %_llgo_223 ], [ zeroinitializer, %_llgo_224 ]
// CHECK-NEXT:   %335 = extractvalue { ptr, i1 } %334, 0
// CHECK-NEXT:   %336 = extractvalue { ptr, i1 } %334, 1
// CHECK-NEXT:   br i1 %336, label %_llgo_73, label %_llgo_74
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_226:                                        ; preds = %_llgo_74
// CHECK-NEXT:   %337 = extractvalue %"{{.*}}/runtime/internal/runtime.iface" %0, 1
// CHECK-NEXT:   %338 = insertvalue { ptr, i1 } undef, ptr %337, 0
// CHECK-NEXT:   %339 = insertvalue { ptr, i1 } %338, i1 true, 1
// CHECK-NEXT:   br label %_llgo_228
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_227:                                        ; preds = %_llgo_74
// CHECK-NEXT:   br label %_llgo_228
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_228:                                        ; preds = %_llgo_227, %_llgo_226
// CHECK-NEXT:   %340 = phi { ptr, i1 } [ %339, %_llgo_226 ], [ zeroinitializer, %_llgo_227 ]
// CHECK-NEXT:   %341 = extractvalue { ptr, i1 } %340, 0
// CHECK-NEXT:   %342 = extractvalue { ptr, i1 } %340, 1
// CHECK-NEXT:   br i1 %342, label %_llgo_75, label %_llgo_76
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_229:                                        ; preds = %_llgo_76
// CHECK-NEXT:   %343 = extractvalue %"{{.*}}/runtime/internal/runtime.iface" %0, 1
// CHECK-NEXT:   %344 = insertvalue { ptr, i1 } undef, ptr %343, 0
// CHECK-NEXT:   %345 = insertvalue { ptr, i1 } %344, i1 true, 1
// CHECK-NEXT:   br label %_llgo_231
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_230:                                        ; preds = %_llgo_76
// CHECK-NEXT:   br label %_llgo_231
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_231:                                        ; preds = %_llgo_230, %_llgo_229
// CHECK-NEXT:   %346 = phi { ptr, i1 } [ %345, %_llgo_229 ], [ zeroinitializer, %_llgo_230 ]
// CHECK-NEXT:   %347 = extractvalue { ptr, i1 } %346, 0
// CHECK-NEXT:   %348 = extractvalue { ptr, i1 } %346, 1
// CHECK-NEXT:   br i1 %348, label %_llgo_77, label %_llgo_78
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_232:                                        ; preds = %_llgo_78
// CHECK-NEXT:   %349 = extractvalue %"{{.*}}/runtime/internal/runtime.iface" %0, 1
// CHECK-NEXT:   %350 = insertvalue { ptr, i1 } undef, ptr %349, 0
// CHECK-NEXT:   %351 = insertvalue { ptr, i1 } %350, i1 true, 1
// CHECK-NEXT:   br label %_llgo_234
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_233:                                        ; preds = %_llgo_78
// CHECK-NEXT:   br label %_llgo_234
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_234:                                        ; preds = %_llgo_233, %_llgo_232
// CHECK-NEXT:   %352 = phi { ptr, i1 } [ %351, %_llgo_232 ], [ zeroinitializer, %_llgo_233 ]
// CHECK-NEXT:   %353 = extractvalue { ptr, i1 } %352, 0
// CHECK-NEXT:   %354 = extractvalue { ptr, i1 } %352, 1
// CHECK-NEXT:   br i1 %354, label %_llgo_79, label %_llgo_80
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_235:                                        ; preds = %_llgo_80
// CHECK-NEXT:   %355 = extractvalue %"{{.*}}/runtime/internal/runtime.iface" %0, 1
// CHECK-NEXT:   %356 = insertvalue { ptr, i1 } undef, ptr %355, 0
// CHECK-NEXT:   %357 = insertvalue { ptr, i1 } %356, i1 true, 1
// CHECK-NEXT:   br label %_llgo_237
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_236:                                        ; preds = %_llgo_80
// CHECK-NEXT:   br label %_llgo_237
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_237:                                        ; preds = %_llgo_236, %_llgo_235
// CHECK-NEXT:   %358 = phi { ptr, i1 } [ %357, %_llgo_235 ], [ zeroinitializer, %_llgo_236 ]
// CHECK-NEXT:   %359 = extractvalue { ptr, i1 } %358, 0
// CHECK-NEXT:   %360 = extractvalue { ptr, i1 } %358, 1
// CHECK-NEXT:   br i1 %360, label %_llgo_81, label %_llgo_82
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_238:                                        ; preds = %_llgo_82
// CHECK-NEXT:   %361 = extractvalue %"{{.*}}/runtime/internal/runtime.iface" %0, 1
// CHECK-NEXT:   %362 = insertvalue { ptr, i1 } undef, ptr %361, 0
// CHECK-NEXT:   %363 = insertvalue { ptr, i1 } %362, i1 true, 1
// CHECK-NEXT:   br label %_llgo_240
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_239:                                        ; preds = %_llgo_82
// CHECK-NEXT:   br label %_llgo_240
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_240:                                        ; preds = %_llgo_239, %_llgo_238
// CHECK-NEXT:   %364 = phi { ptr, i1 } [ %363, %_llgo_238 ], [ zeroinitializer, %_llgo_239 ]
// CHECK-NEXT:   %365 = extractvalue { ptr, i1 } %364, 0
// CHECK-NEXT:   %366 = extractvalue { ptr, i1 } %364, 1
// CHECK-NEXT:   br i1 %366, label %_llgo_83, label %_llgo_84
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_241:                                        ; preds = %_llgo_84
// CHECK-NEXT:   %367 = extractvalue %"{{.*}}/runtime/internal/runtime.iface" %0, 1
// CHECK-NEXT:   %368 = insertvalue { ptr, i1 } undef, ptr %367, 0
// CHECK-NEXT:   %369 = insertvalue { ptr, i1 } %368, i1 true, 1
// CHECK-NEXT:   br label %_llgo_243
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_242:                                        ; preds = %_llgo_84
// CHECK-NEXT:   br label %_llgo_243
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_243:                                        ; preds = %_llgo_242, %_llgo_241
// CHECK-NEXT:   %370 = phi { ptr, i1 } [ %369, %_llgo_241 ], [ zeroinitializer, %_llgo_242 ]
// CHECK-NEXT:   %371 = extractvalue { ptr, i1 } %370, 0
// CHECK-NEXT:   %372 = extractvalue { ptr, i1 } %370, 1
// CHECK-NEXT:   br i1 %372, label %_llgo_85, label %_llgo_86
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_244:                                        ; preds = %_llgo_86
// CHECK-NEXT:   %373 = extractvalue %"{{.*}}/runtime/internal/runtime.iface" %0, 1
// CHECK-NEXT:   %374 = insertvalue { ptr, i1 } undef, ptr %373, 0
// CHECK-NEXT:   %375 = insertvalue { ptr, i1 } %374, i1 true, 1
// CHECK-NEXT:   br label %_llgo_246
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_245:                                        ; preds = %_llgo_86
// CHECK-NEXT:   br label %_llgo_246
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_246:                                        ; preds = %_llgo_245, %_llgo_244
// CHECK-NEXT:   %376 = phi { ptr, i1 } [ %375, %_llgo_244 ], [ zeroinitializer, %_llgo_245 ]
// CHECK-NEXT:   %377 = extractvalue { ptr, i1 } %376, 0
// CHECK-NEXT:   %378 = extractvalue { ptr, i1 } %376, 1
// CHECK-NEXT:   br i1 %378, label %_llgo_87, label %_llgo_88
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_247:                                        ; preds = %_llgo_88
// CHECK-NEXT:   %379 = extractvalue %"{{.*}}/runtime/internal/runtime.iface" %0, 1
// CHECK-NEXT:   %380 = insertvalue { ptr, i1 } undef, ptr %379, 0
// CHECK-NEXT:   %381 = insertvalue { ptr, i1 } %380, i1 true, 1
// CHECK-NEXT:   br label %_llgo_249
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_248:                                        ; preds = %_llgo_88
// CHECK-NEXT:   br label %_llgo_249
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_249:                                        ; preds = %_llgo_248, %_llgo_247
// CHECK-NEXT:   %382 = phi { ptr, i1 } [ %381, %_llgo_247 ], [ zeroinitializer, %_llgo_248 ]
// CHECK-NEXT:   %383 = extractvalue { ptr, i1 } %382, 0
// CHECK-NEXT:   %384 = extractvalue { ptr, i1 } %382, 1
// CHECK-NEXT:   br i1 %384, label %_llgo_89, label %_llgo_90
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_250:                                        ; preds = %_llgo_90
// CHECK-NEXT:   %385 = extractvalue %"{{.*}}/runtime/internal/runtime.iface" %0, 1
// CHECK-NEXT:   %386 = insertvalue { ptr, i1 } undef, ptr %385, 0
// CHECK-NEXT:   %387 = insertvalue { ptr, i1 } %386, i1 true, 1
// CHECK-NEXT:   br label %_llgo_252
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_251:                                        ; preds = %_llgo_90
// CHECK-NEXT:   br label %_llgo_252
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_252:                                        ; preds = %_llgo_251, %_llgo_250
// CHECK-NEXT:   %388 = phi { ptr, i1 } [ %387, %_llgo_250 ], [ zeroinitializer, %_llgo_251 ]
// CHECK-NEXT:   %389 = extractvalue { ptr, i1 } %388, 0
// CHECK-NEXT:   %390 = extractvalue { ptr, i1 } %388, 1
// CHECK-NEXT:   br i1 %390, label %_llgo_91, label %_llgo_92
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_253:                                        ; preds = %_llgo_92
// CHECK-NEXT:   %391 = extractvalue %"{{.*}}/runtime/internal/runtime.iface" %0, 1
// CHECK-NEXT:   %392 = insertvalue { ptr, i1 } undef, ptr %391, 0
// CHECK-NEXT:   %393 = insertvalue { ptr, i1 } %392, i1 true, 1
// CHECK-NEXT:   br label %_llgo_255
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_254:                                        ; preds = %_llgo_92
// CHECK-NEXT:   br label %_llgo_255
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_255:                                        ; preds = %_llgo_254, %_llgo_253
// CHECK-NEXT:   %394 = phi { ptr, i1 } [ %393, %_llgo_253 ], [ zeroinitializer, %_llgo_254 ]
// CHECK-NEXT:   %395 = extractvalue { ptr, i1 } %394, 0
// CHECK-NEXT:   %396 = extractvalue { ptr, i1 } %394, 1
// CHECK-NEXT:   br i1 %396, label %_llgo_93, label %_llgo_94
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_256:                                        ; preds = %_llgo_94
// CHECK-NEXT:   %397 = extractvalue %"{{.*}}/runtime/internal/runtime.iface" %0, 1
// CHECK-NEXT:   %398 = insertvalue { ptr, i1 } undef, ptr %397, 0
// CHECK-NEXT:   %399 = insertvalue { ptr, i1 } %398, i1 true, 1
// CHECK-NEXT:   br label %_llgo_258
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_257:                                        ; preds = %_llgo_94
// CHECK-NEXT:   br label %_llgo_258
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_258:                                        ; preds = %_llgo_257, %_llgo_256
// CHECK-NEXT:   %400 = phi { ptr, i1 } [ %399, %_llgo_256 ], [ zeroinitializer, %_llgo_257 ]
// CHECK-NEXT:   %401 = extractvalue { ptr, i1 } %400, 0
// CHECK-NEXT:   %402 = extractvalue { ptr, i1 } %400, 1
// CHECK-NEXT:   br i1 %402, label %_llgo_95, label %_llgo_96
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_259:                                        ; preds = %_llgo_96
// CHECK-NEXT:   %403 = extractvalue %"{{.*}}/runtime/internal/runtime.iface" %0, 1
// CHECK-NEXT:   %404 = insertvalue { ptr, i1 } undef, ptr %403, 0
// CHECK-NEXT:   %405 = insertvalue { ptr, i1 } %404, i1 true, 1
// CHECK-NEXT:   br label %_llgo_261
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_260:                                        ; preds = %_llgo_96
// CHECK-NEXT:   br label %_llgo_261
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_261:                                        ; preds = %_llgo_260, %_llgo_259
// CHECK-NEXT:   %406 = phi { ptr, i1 } [ %405, %_llgo_259 ], [ zeroinitializer, %_llgo_260 ]
// CHECK-NEXT:   %407 = extractvalue { ptr, i1 } %406, 0
// CHECK-NEXT:   %408 = extractvalue { ptr, i1 } %406, 1
// CHECK-NEXT:   br i1 %408, label %_llgo_97, label %_llgo_98
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_262:                                        ; preds = %_llgo_98
// CHECK-NEXT:   %409 = extractvalue %"{{.*}}/runtime/internal/runtime.iface" %0, 1
// CHECK-NEXT:   %410 = insertvalue { ptr, i1 } undef, ptr %409, 0
// CHECK-NEXT:   %411 = insertvalue { ptr, i1 } %410, i1 true, 1
// CHECK-NEXT:   br label %_llgo_264
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_263:                                        ; preds = %_llgo_98
// CHECK-NEXT:   br label %_llgo_264
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_264:                                        ; preds = %_llgo_263, %_llgo_262
// CHECK-NEXT:   %412 = phi { ptr, i1 } [ %411, %_llgo_262 ], [ zeroinitializer, %_llgo_263 ]
// CHECK-NEXT:   %413 = extractvalue { ptr, i1 } %412, 0
// CHECK-NEXT:   %414 = extractvalue { ptr, i1 } %412, 1
// CHECK-NEXT:   br i1 %414, label %_llgo_99, label %_llgo_100
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_265:                                        ; preds = %_llgo_100
// CHECK-NEXT:   %415 = extractvalue %"{{.*}}/runtime/internal/runtime.iface" %0, 1
// CHECK-NEXT:   %416 = insertvalue { ptr, i1 } undef, ptr %415, 0
// CHECK-NEXT:   %417 = insertvalue { ptr, i1 } %416, i1 true, 1
// CHECK-NEXT:   br label %_llgo_267
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_266:                                        ; preds = %_llgo_100
// CHECK-NEXT:   br label %_llgo_267
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_267:                                        ; preds = %_llgo_266, %_llgo_265
// CHECK-NEXT:   %418 = phi { ptr, i1 } [ %417, %_llgo_265 ], [ zeroinitializer, %_llgo_266 ]
// CHECK-NEXT:   %419 = extractvalue { ptr, i1 } %418, 0
// CHECK-NEXT:   %420 = extractvalue { ptr, i1 } %418, 1
// CHECK-NEXT:   br i1 %420, label %_llgo_101, label %_llgo_102
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_268:                                        ; preds = %_llgo_102
// CHECK-NEXT:   %421 = extractvalue %"{{.*}}/runtime/internal/runtime.iface" %0, 1
// CHECK-NEXT:   %422 = insertvalue { ptr, i1 } undef, ptr %421, 0
// CHECK-NEXT:   %423 = insertvalue { ptr, i1 } %422, i1 true, 1
// CHECK-NEXT:   br label %_llgo_270
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_269:                                        ; preds = %_llgo_102
// CHECK-NEXT:   br label %_llgo_270
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_270:                                        ; preds = %_llgo_269, %_llgo_268
// CHECK-NEXT:   %424 = phi { ptr, i1 } [ %423, %_llgo_268 ], [ zeroinitializer, %_llgo_269 ]
// CHECK-NEXT:   %425 = extractvalue { ptr, i1 } %424, 0
// CHECK-NEXT:   %426 = extractvalue { ptr, i1 } %424, 1
// CHECK-NEXT:   br i1 %426, label %_llgo_103, label %_llgo_104
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_271:                                        ; preds = %_llgo_104
// CHECK-NEXT:   %427 = extractvalue %"{{.*}}/runtime/internal/runtime.iface" %0, 1
// CHECK-NEXT:   %428 = insertvalue { ptr, i1 } undef, ptr %427, 0
// CHECK-NEXT:   %429 = insertvalue { ptr, i1 } %428, i1 true, 1
// CHECK-NEXT:   br label %_llgo_273
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_272:                                        ; preds = %_llgo_104
// CHECK-NEXT:   br label %_llgo_273
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_273:                                        ; preds = %_llgo_272, %_llgo_271
// CHECK-NEXT:   %430 = phi { ptr, i1 } [ %429, %_llgo_271 ], [ zeroinitializer, %_llgo_272 ]
// CHECK-NEXT:   %431 = extractvalue { ptr, i1 } %430, 0
// CHECK-NEXT:   %432 = extractvalue { ptr, i1 } %430, 1
// CHECK-NEXT:   br i1 %432, label %_llgo_105, label %_llgo_106
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_274:                                        ; preds = %_llgo_106
// CHECK-NEXT:   %433 = extractvalue %"{{.*}}/runtime/internal/runtime.iface" %0, 1
// CHECK-NEXT:   %434 = insertvalue { ptr, i1 } undef, ptr %433, 0
// CHECK-NEXT:   %435 = insertvalue { ptr, i1 } %434, i1 true, 1
// CHECK-NEXT:   br label %_llgo_276
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_275:                                        ; preds = %_llgo_106
// CHECK-NEXT:   br label %_llgo_276
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_276:                                        ; preds = %_llgo_275, %_llgo_274
// CHECK-NEXT:   %436 = phi { ptr, i1 } [ %435, %_llgo_274 ], [ zeroinitializer, %_llgo_275 ]
// CHECK-NEXT:   %437 = extractvalue { ptr, i1 } %436, 0
// CHECK-NEXT:   %438 = extractvalue { ptr, i1 } %436, 1
// CHECK-NEXT:   br i1 %438, label %_llgo_107, label %_llgo_108
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_277:                                        ; preds = %_llgo_108
// CHECK-NEXT:   %439 = extractvalue %"{{.*}}/runtime/internal/runtime.iface" %0, 1
// CHECK-NEXT:   %440 = insertvalue { ptr, i1 } undef, ptr %439, 0
// CHECK-NEXT:   %441 = insertvalue { ptr, i1 } %440, i1 true, 1
// CHECK-NEXT:   br label %_llgo_279
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_278:                                        ; preds = %_llgo_108
// CHECK-NEXT:   br label %_llgo_279
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_279:                                        ; preds = %_llgo_278, %_llgo_277
// CHECK-NEXT:   %442 = phi { ptr, i1 } [ %441, %_llgo_277 ], [ zeroinitializer, %_llgo_278 ]
// CHECK-NEXT:   %443 = extractvalue { ptr, i1 } %442, 0
// CHECK-NEXT:   %444 = extractvalue { ptr, i1 } %442, 1
// CHECK-NEXT:   br i1 %444, label %_llgo_109, label %_llgo_110
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_280:                                        ; preds = %_llgo_110
// CHECK-NEXT:   %445 = extractvalue %"{{.*}}/runtime/internal/runtime.iface" %0, 1
// CHECK-NEXT:   %446 = insertvalue { ptr, i1 } undef, ptr %445, 0
// CHECK-NEXT:   %447 = insertvalue { ptr, i1 } %446, i1 true, 1
// CHECK-NEXT:   br label %_llgo_282
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_281:                                        ; preds = %_llgo_110
// CHECK-NEXT:   br label %_llgo_282
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_282:                                        ; preds = %_llgo_281, %_llgo_280
// CHECK-NEXT:   %448 = phi { ptr, i1 } [ %447, %_llgo_280 ], [ zeroinitializer, %_llgo_281 ]
// CHECK-NEXT:   %449 = extractvalue { ptr, i1 } %448, 0
// CHECK-NEXT:   %450 = extractvalue { ptr, i1 } %448, 1
// CHECK-NEXT:   br i1 %450, label %_llgo_111, label %_llgo_112
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_283:                                        ; preds = %_llgo_112
// CHECK-NEXT:   %451 = extractvalue %"{{.*}}/runtime/internal/runtime.iface" %0, 1
// CHECK-NEXT:   %452 = insertvalue { ptr, i1 } undef, ptr %451, 0
// CHECK-NEXT:   %453 = insertvalue { ptr, i1 } %452, i1 true, 1
// CHECK-NEXT:   br label %_llgo_285
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_284:                                        ; preds = %_llgo_112
// CHECK-NEXT:   br label %_llgo_285
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_285:                                        ; preds = %_llgo_284, %_llgo_283
// CHECK-NEXT:   %454 = phi { ptr, i1 } [ %453, %_llgo_283 ], [ zeroinitializer, %_llgo_284 ]
// CHECK-NEXT:   %455 = extractvalue { ptr, i1 } %454, 0
// CHECK-NEXT:   %456 = extractvalue { ptr, i1 } %454, 1
// CHECK-NEXT:   br i1 %456, label %_llgo_113, label %_llgo_114
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce i1 @"__llgo_stub.{{.*}}/runtime/internal/runtime.memequal64"(ptr %0, ptr %1, ptr %2){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %3 = tail call i1 @"{{.*}}/runtime/internal/runtime.memequal64"(ptr %1, ptr %2)
// CHECK-NEXT:   ret i1 %3
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce i1 @"__llgo_stub.{{.*}}/runtime/internal/runtime.memequal8"(ptr %0, ptr %1, ptr %2){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %3 = tail call i1 @"{{.*}}/runtime/internal/runtime.memequal8"(ptr %1, ptr %2)
// CHECK-NEXT:   ret i1 %3
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce i1 @"__llgo_stub.{{.*}}/runtime/internal/runtime.nilinterequal"(ptr %0, ptr %1, ptr %2){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %3 = tail call i1 @"{{.*}}/runtime/internal/runtime.nilinterequal"(ptr %1, ptr %2)
// CHECK-NEXT:   ret i1 %3
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce i1 @"__llgo_stub.{{.*}}/runtime/internal/runtime.interequal"(ptr %0, ptr %1, ptr %2){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %3 = tail call i1 @"{{.*}}/runtime/internal/runtime.interequal"(ptr %1, ptr %2)
// CHECK-NEXT:   ret i1 %3
// CHECK-NEXT: }
