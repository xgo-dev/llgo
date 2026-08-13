// LITTEST
package main

import (
	"strings"
	"unicode"
)

// CHECK: [[HELLO:@[0-9]+]] = private unnamed_addr constant [6 x i8] c"Hello "
// CHECK: [[WORLD:@[0-9]+]] = private unnamed_addr constant [5 x i8] c"World"
// CHECK: [[WITH_HAN:@[0-9]+]] = private unnamed_addr constant [13 x i8] c"Hello, \E4\B8\96\E7\95\8C"
// CHECK: [[WITHOUT_HAN:@[0-9]+]] = private unnamed_addr constant [12 x i8] c"Hello, world"
// CHECK-LABEL: define void @main.main(){{.*}} {
func main() {
	// All Builder operations use one receiver, and the queried values are printed.
	// CHECK: [[BUILDER:%.*]] = call ptr @"{{.*}}AllocZ"(i64 32)
	// CHECK: [[HELLO_BYTES:%.*]] = call %"{{.*}}Slice" @"{{.*}}StringToBytes"(%"{{.*}}String" { ptr [[HELLO]], i64 6 })
	// CHECK: call { i64, %"{{.*}}iface" } @"strings.(*Builder).Write"(ptr [[BUILDER]], %"{{.*}}Slice" [[HELLO_BYTES]])
	// CHECK: call { i64, %"{{.*}}iface" } @"strings.(*Builder).WriteString"(ptr [[BUILDER]], %"{{.*}}String" { ptr [[WORLD]], i64 5 })
	// CHECK: [[BUILDER_LEN:%.*]] = call i64 @"strings.(*Builder).Len"(ptr [[BUILDER]])
	// CHECK: [[BUILDER_CAP:%.*]] = call i64 @"strings.(*Builder).Cap"(ptr [[BUILDER]])
	// CHECK: [[BUILDER_STRING:%.*]] = call %"{{.*}}String" @"strings.(*Builder).String"(ptr [[BUILDER]])
	// CHECK: call void @"{{.*}}PrintInt"(i64 [[BUILDER_LEN]])
	// CHECK: call void @"{{.*}}PrintInt"(i64 [[BUILDER_CAP]])
	// CHECK: call void @"{{.*}}PrintString"(%"{{.*}}String" [[BUILDER_STRING]])
	var b strings.Builder
	b.Write([]byte("Hello "))
	b.WriteString("World")

	println("len:", b.Len(), "cap:", b.Cap(), "string:", b.String())

	f := func(c rune) bool {
		return unicode.Is(unicode.Han, c)
	}
	// CHECK: [[WITH_HAN_INDEX:%.*]] = call i64 @strings.IndexFunc(%"{{.*}}String" { ptr [[WITH_HAN]], i64 13 }, { ptr, ptr } { ptr @"main.main$1", ptr null })
	// CHECK-NEXT: call void @"{{.*}}PrintInt"(i64 [[WITH_HAN_INDEX]])
	// CHECK: [[WITHOUT_HAN_INDEX:%.*]] = call i64 @strings.IndexFunc(%"{{.*}}String" { ptr [[WITHOUT_HAN]], i64 12 }, { ptr, ptr } { ptr @"main.main$1", ptr null })
	// CHECK-NEXT: call void @"{{.*}}PrintInt"(i64 [[WITHOUT_HAN_INDEX]])
	println(strings.IndexFunc("Hello, 世界", f))
	println(strings.IndexFunc("Hello, world", f))
}

// CHECK-LABEL: define i1 @"main.main$1"(i32 %0){{.*}} {
// CHECK: [[HAN_TABLE:%.*]] = load ptr, ptr @unicode.Han
// CHECK-NEXT: [[IS_HAN:%.*]] = call i1 @unicode.Is(ptr [[HAN_TABLE]], i32 %0)
// CHECK-NEXT: ret i1 [[IS_HAN]]
