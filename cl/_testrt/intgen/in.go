// LITTEST
package main

import (
	"github.com/goplus/lib/c"
)

// genInts invokes the supplied code/environment pair once per output slot and
// stores that result into the slice at the checked loop index.
// CHECK-LABEL: define %"{{.*}}Slice" @main.genInts(i64 %0, { ptr, ptr } %1){{.*}} {
// CHECK: [[OUT:%[0-9]+]] = call %"{{.*}}Slice" @"{{.*}}MakeSlice"(i64 %0, i64 %0, i64 4)
// CHECK-NEXT: [[OUT_LEN:%[0-9]+]] = extractvalue %"{{.*}}Slice" [[OUT]], 1
// CHECK: [[GEN_INDEX:%[0-9]+]] = add i64 {{.*}}, 1
// CHECK-NEXT: [[GEN_MORE:%[0-9]+]] = icmp slt i64 [[GEN_INDEX]], [[OUT_LEN]]
// CHECK: [[CALL_ENV:%[0-9]+]] = extractvalue { ptr, ptr } %1, 1
// CHECK-NEXT: [[CALL_RAW_CODE:%[0-9]+]] = extractvalue { ptr, ptr } %1, 0
// CHECK-NEXT: %__llgo_funcval_code = call ptr asm "", "=r,0"(ptr [[CALL_RAW_CODE]])
// CHECK-NEXT: [[GENERATED:%[0-9]+]] = call i32 %__llgo_funcval_code(ptr {{(nest|swiftself)}} [[CALL_ENV]])
// CHECK-NEXT: [[OUT_DATA:%[0-9]+]] = extractvalue %"{{.*}}Slice" [[OUT]], 0
// CHECK-NEXT: [[BOUNDS_LEN:%[0-9]+]] = extractvalue %"{{.*}}Slice" [[OUT]], 1
// CHECK: call void @"{{.*}}CheckIndexRange"(i1 {{.*}}, i64 [[GEN_INDEX]], i1 true, i64 [[BOUNDS_LEN]])
// CHECK-NEXT: [[OUT_SLOT:%[0-9]+]] = getelementptr inbounds i32, ptr [[OUT_DATA]], i64 [[GEN_INDEX]]
// CHECK-NEXT: store i32 [[GENERATED]], ptr [[OUT_SLOT]]
// CHECK: ret %"{{.*}}Slice" [[OUT]]

// CHECK-LABEL: define i32 @"main.(*generator).next"(ptr %0){{.*}} {
// CHECK: [[NEXT_FIELD:%[0-9]+]] = getelementptr inbounds %main.generator, ptr %0, i32 0, i32 0
// CHECK-NEXT: [[NEXT_OLD:%[0-9]+]] = load i32, ptr [[NEXT_FIELD]]
// CHECK-NEXT: [[NEXT_NEW:%[0-9]+]] = add i32 [[NEXT_OLD]], 1
// CHECK-NEXT: [[NEXT_STORE:%[0-9]+]] = getelementptr inbounds %main.generator, ptr %0, i32 0, i32 0
// CHECK-NEXT: store i32 [[NEXT_NEW]], ptr [[NEXT_STORE]]
// CHECK-NEXT: [[NEXT_RESULT_FIELD:%[0-9]+]] = getelementptr inbounds %main.generator, ptr %0, i32 0, i32 0
// CHECK-NEXT: [[NEXT_RESULT:%[0-9]+]] = load i32, ptr [[NEXT_RESULT_FIELD]]
// CHECK-NEXT: ret i32 [[NEXT_RESULT]]

// CHECK-LABEL: define void @main.main(){{.*}} {
// CHECK: [[RAND_VALUES:%[0-9]+]] = call %"{{.*}}Slice" @main.genInts(i64 5, { ptr, ptr } { ptr @rand, ptr null })
// CHECK: [[RAND_DATA:%[0-9]+]] = extractvalue %"{{.*}}Slice" [[RAND_VALUES]], 0
// CHECK: [[RAND_SLOT:%[0-9]+]] = getelementptr inbounds i32, ptr [[RAND_DATA]], i64 [[RAND_INDEX:%[0-9]+]]
// CHECK-NEXT: [[RAND_VALUE:%[0-9]+]] = load i32, ptr [[RAND_SLOT]]
// CHECK-NEXT: call i32 (ptr, ...) @printf(ptr @{{[0-9]+}}, i32 [[RAND_VALUE]])
// CHECK: [[CAPTURED_VALUE:%[0-9]+]] = call ptr @"{{.*}}AllocZ"(i64 4)
// CHECK-NEXT: store i32 1, ptr [[CAPTURED_VALUE]]
// CHECK-NEXT: [[DOUBLE_ENV:%[0-9]+]] = call ptr @"{{.*}}AllocU"(i64 8)
// CHECK-NEXT: [[CAPTURE_SLOT:%[0-9]+]] = getelementptr inbounds { ptr }, ptr [[DOUBLE_ENV]], i32 0, i32 0
// CHECK-NEXT: store ptr [[CAPTURED_VALUE]], ptr [[CAPTURE_SLOT]]
// CHECK-NEXT: [[DOUBLE_CLOSURE:%[0-9]+]] = insertvalue { ptr, ptr } { ptr @"main.main$1", ptr undef }, ptr [[DOUBLE_ENV]], 1
// CHECK-NEXT: [[DOUBLE_VALUES:%[0-9]+]] = call %"{{.*}}Slice" @main.genInts(i64 5, { ptr, ptr } [[DOUBLE_CLOSURE]])
// CHECK: [[DOUBLE_DATA:%[0-9]+]] = extractvalue %"{{.*}}Slice" [[DOUBLE_VALUES]], 0
// CHECK: [[DOUBLE_SLOT:%[0-9]+]] = getelementptr inbounds i32, ptr [[DOUBLE_DATA]], i64 [[DOUBLE_INDEX:%[0-9]+]]
// CHECK-NEXT: [[DOUBLE_VALUE:%[0-9]+]] = load i32, ptr [[DOUBLE_SLOT]]
// CHECK-NEXT: call i32 (ptr, ...) @printf(ptr @{{[0-9]+}}, i32 [[DOUBLE_VALUE]])
// CHECK: [[GENERATOR:%[0-9]+]] = call ptr @"{{.*}}AllocZ"(i64 4)
// CHECK: [[BOUND_ENV_ADDR:%[0-9]+]] = call ptr @"{{.*}}AllocU"(i64 8)
// CHECK-NEXT: [[BOUND_SLOT:%[0-9]+]] = getelementptr inbounds { ptr }, ptr [[BOUND_ENV_ADDR]], i32 0, i32 0
// CHECK-NEXT: store ptr [[GENERATOR]], ptr [[BOUND_SLOT]]
// CHECK-NEXT: [[BOUND_NEXT:%[0-9]+]] = insertvalue { ptr, ptr } { ptr @"main.(*generator).next$bound", ptr undef }, ptr [[BOUND_ENV_ADDR]], 1
// CHECK-NEXT: [[NEXT_VALUES:%[0-9]+]] = call %"{{.*}}Slice" @main.genInts(i64 5, { ptr, ptr } [[BOUND_NEXT]])
// CHECK: [[BOUND_DATA:%[0-9]+]] = extractvalue %"{{.*}}Slice" [[NEXT_VALUES]], 0
// CHECK: [[BOUND_VALUE_SLOT:%[0-9]+]] = getelementptr inbounds i32, ptr [[BOUND_DATA]], i64 [[BOUND_INDEX:%[0-9]+]]
// CHECK-NEXT: [[BOUND_VALUE:%[0-9]+]] = load i32, ptr [[BOUND_VALUE_SLOT]]
// CHECK-NEXT: call i32 (ptr, ...) @printf(ptr @{{[0-9]+}}, i32 [[BOUND_VALUE]])

// CHECK-LABEL: define i32 @"main.main$1"(ptr {{(nest|swiftself)}} %0){{.*}} {
// CHECK: [[GEN_ENV:%[0-9]+]] = load { ptr }, ptr %0
// CHECK-NEXT: [[GEN_VALUE_PTR:%[0-9]+]] = extractvalue { ptr } [[GEN_ENV]], 0
// CHECK-NEXT: [[GEN_VALUE:%[0-9]+]] = load i32, ptr [[GEN_VALUE_PTR]]
// CHECK-NEXT: [[GEN_DOUBLE:%[0-9]+]] = mul i32 [[GEN_VALUE]], 2
// CHECK-NEXT: [[GEN_STORE:%[0-9]+]] = extractvalue { ptr } [[GEN_ENV]], 0
// CHECK-NEXT: store i32 [[GEN_DOUBLE]], ptr [[GEN_STORE]]
// CHECK-NEXT: [[GEN_RESULT_PTR:%[0-9]+]] = extractvalue { ptr } [[GEN_ENV]], 0
// CHECK-NEXT: [[GEN_RESULT:%[0-9]+]] = load i32, ptr [[GEN_RESULT_PTR]]
// CHECK-NEXT: ret i32 [[GEN_RESULT]]
// CHECK-LABEL: define i32 @"main.(*generator).next$bound"(ptr {{(nest|swiftself)}} %0){{.*}} {
// CHECK: [[BOUND_ENV:%[0-9]+]] = load { ptr }, ptr %0
// CHECK-NEXT: [[BOUND_RECEIVER:%[0-9]+]] = extractvalue { ptr } [[BOUND_ENV]], 0
// CHECK-NEXT: [[BOUND_RESULT:%[0-9]+]] = call i32 @"main.(*generator).next"(ptr [[BOUND_RECEIVER]])
// CHECK-NEXT: ret i32 [[BOUND_RESULT]]

func genInts(n int, gen func() c.Int) []c.Int {
	a := make([]c.Int, n)
	for i := range a {
		a[i] = gen()
	}
	return a
}

func (g *generator) next() c.Int {
	g.val++
	return g.val
}

type generator struct {
	val c.Int
}

func main() {
	for _, v := range genInts(5, c.Rand) {

		c.Printf(c.Str("%d\n"), v)
	}

	initVal := c.Int(1)
	ints := genInts(5, func() c.Int {
		initVal *= 2
		return initVal
	})
	for _, v := range ints {
		c.Printf(c.Str("%d\n"), v)
	}

	g := &generator{val: 1}
	for _, v := range genInts(5, g.next) {
		c.Printf(c.Str("%d\n"), v)
	}
}
