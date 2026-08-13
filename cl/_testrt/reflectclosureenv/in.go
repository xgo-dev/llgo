// LITTEST
package main

import "reflect"

type receiver struct {
	base int
}

// CHECK-LABEL: define void @main.checkInt(%reflect.Value %0, %"{{.*}}Slice" %1){{.*}} {
// CHECK: [[CHECK_RESULTS:%.*]] = call %"{{.*}}Slice" @reflect.Value.Call(%reflect.Value %0, %"{{.*}}Slice" %1)
// CHECK: [[CHECK_RESULTS_PTR:%.*]] = extractvalue %"{{.*}}Slice" [[CHECK_RESULTS]], 0
// CHECK: [[CHECK_RESULTS_LEN:%.*]] = extractvalue %"{{.*}}Slice" [[CHECK_RESULTS]], 1
// CHECK: call void @"{{.*}}/runtime/internal/runtime.CheckIndexRange"(i1 %{{.*}}, i64 0, i1 true, i64 [[CHECK_RESULTS_LEN]])
// CHECK: [[CHECK_FIRST_PTR:%.*]] = getelementptr inbounds %reflect.Value, ptr [[CHECK_RESULTS_PTR]], i64 0
// CHECK: [[CHECK_FIRST_SAFE:%.*]] = call ptr @"{{.*}}/runtime/internal/runtime.AssertNilDerefPtr"(ptr [[CHECK_FIRST_PTR]])
// CHECK: [[CHECK_FIRST:%.*]] = load %reflect.Value, ptr [[CHECK_FIRST_SAFE]]
// CHECK: [[CHECK_GOT:%.*]] = call i64 @reflect.Value.Int(%reflect.Value [[CHECK_FIRST]])
// CHECK: [[CHECK_BAD:%.*]] = icmp ne i64 [[CHECK_GOT]], 55
// CHECK: br i1 [[CHECK_BAD]], label %{{.*}}, label %{{.*}}
// CHECK: store i64 [[CHECK_GOT]], ptr %{{.*}}
// CHECK: call void @"{{.*}}/runtime/internal/runtime.Panic"

// CHECK-LABEL: define %"{{.*}}Slice" @main.floatArgs(){{.*}} {
// CHECK: [[FLOAT_ARG_STORAGE:%.*]] = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 216)
// CHECK: [[FLOAT_ARGS:%.*]] = call %"{{.*}}Slice" @"{{.*}}/runtime/internal/runtime.NewSlice2"(ptr [[FLOAT_ARG_STORAGE]], i64 24, i64 9, i64 0, i64 9, i1 true, i1 true, i1 true)
// CHECK: [[FLOAT_INDEX:%.*]] = add i64 %{{.*}}, 1
// CHECK: [[FLOAT_ORDINAL:%.*]] = add i64 [[FLOAT_INDEX]], 1
// CHECK: [[FLOAT_NUMBER:%.*]] = sitofp i64 [[FLOAT_ORDINAL]] to double
// CHECK: store double [[FLOAT_NUMBER]], ptr [[FLOAT_BOX_ADDR:%[-A-Za-z0-9_.]+]]
// CHECK: [[FLOAT_BOX:%.*]] = insertvalue %"{{.*}}eface" { ptr @_llgo_float64, ptr undef }, ptr [[FLOAT_BOX_ADDR]], 1
// CHECK: [[FLOAT_REFLECT:%.*]] = call %reflect.Value @reflect.ValueOf(%"{{.*}}eface" [[FLOAT_BOX]])
// CHECK: [[FLOAT_ARGS_PTR:%.*]] = extractvalue %"{{.*}}Slice" [[FLOAT_ARGS]], 0
// CHECK: [[FLOAT_DEST:%.*]] = getelementptr inbounds %reflect.Value, ptr [[FLOAT_ARGS_PTR]], i64 [[FLOAT_INDEX]]
// CHECK: store %reflect.Value [[FLOAT_REFLECT]], ptr [[FLOAT_DEST]]
// CHECK: ret %"{{.*}}Slice" [[FLOAT_ARGS]]

// CHECK-LABEL: define %"{{.*}}Slice" @main.intArgs(){{.*}} {
// CHECK: [[INT_ARG_STORAGE:%.*]] = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 216)
// CHECK: [[INT_ARGS:%.*]] = call %"{{.*}}Slice" @"{{.*}}/runtime/internal/runtime.NewSlice2"(ptr [[INT_ARG_STORAGE]], i64 24, i64 9, i64 0, i64 9, i1 true, i1 true, i1 true)
// CHECK: [[INT_INDEX:%.*]] = add i64 %{{.*}}, 1
// CHECK: [[INT_ORDINAL:%.*]] = add i64 [[INT_INDEX]], 1
// CHECK: store i64 [[INT_ORDINAL]], ptr [[INT_BOX_ADDR:%[-A-Za-z0-9_.]+]]
// CHECK: [[INT_BOX:%.*]] = insertvalue %"{{.*}}eface" { ptr @_llgo_int, ptr undef }, ptr [[INT_BOX_ADDR]], 1
// CHECK: [[INT_REFLECT:%.*]] = call %reflect.Value @reflect.ValueOf(%"{{.*}}eface" [[INT_BOX]])
// CHECK: [[INT_ARGS_PTR:%.*]] = extractvalue %"{{.*}}Slice" [[INT_ARGS]], 0
// CHECK: [[INT_DEST:%.*]] = getelementptr inbounds %reflect.Value, ptr [[INT_ARGS_PTR]], i64 [[INT_INDEX]]
// CHECK: store %reflect.Value [[INT_REFLECT]], ptr [[INT_DEST]]
// CHECK: ret %"{{.*}}Slice" [[INT_ARGS]]

// CHECK-LABEL: define void @main.main(){{.*}} {
// The direct and nested integer closures are boxed and called with the same nine arguments.
// CHECK: [[MAIN_INTS:%.*]] = call %"{{.*}}Slice" @main.intArgs()
// CHECK: [[MAIN_SUM_FN:%.*]] = call { ptr, ptr } @main.makeSum(i64 10)
// CHECK: store { ptr, ptr } [[MAIN_SUM_FN]], ptr [[MAIN_SUM_ADDR:%[-A-Za-z0-9_.]+]]
// CHECK: [[MAIN_SUM_BOX:%.*]] = insertvalue %"{{.*}}eface" { ptr @"_llgo_closure${{[-A-Za-z0-9_]+}}", ptr undef }, ptr [[MAIN_SUM_ADDR]], 1
// CHECK: [[MAIN_SUM_VALUE:%.*]] = call %reflect.Value @reflect.ValueOf(%"{{.*}}eface" [[MAIN_SUM_BOX]])
// CHECK: call void @main.checkInt(%reflect.Value [[MAIN_SUM_VALUE]], %"{{.*}}Slice" [[MAIN_INTS]])
// CHECK: [[MAIN_NESTED_FN:%.*]] = call { ptr, ptr } @main.makeNestedSum(i64 10)
// CHECK: store { ptr, ptr } [[MAIN_NESTED_FN]], ptr [[MAIN_NESTED_ADDR:%[-A-Za-z0-9_.]+]]
// CHECK: [[MAIN_NESTED_BOX:%.*]] = insertvalue %"{{.*}}eface" { ptr @"_llgo_closure${{[-A-Za-z0-9_]+}}", ptr undef }, ptr [[MAIN_NESTED_ADDR]], 1
// CHECK: [[MAIN_NESTED_VALUE:%.*]] = call %reflect.Value @reflect.ValueOf(%"{{.*}}eface" [[MAIN_NESTED_BOX]])
// CHECK: call void @main.checkInt(%reflect.Value [[MAIN_NESTED_VALUE]], %"{{.*}}Slice" [[MAIN_INTS]])
// MakeFunc receives the type of the nine-argument literal and the slice callback main$2.
// CHECK: [[FUNC_TYPE:%.*]] = call %"{{.*}}iface" @reflect.TypeOf(%"{{.*}}eface" %{{.*}})
// CHECK: [[MADE:%.*]] = call %reflect.Value @reflect.MakeFunc(%"{{.*}}iface" [[FUNC_TYPE]], { ptr, ptr } { ptr @"main.main$2", ptr null })
// CHECK: call void @main.checkInt(%reflect.Value [[MADE]], %"{{.*}}Slice" [[MAIN_INTS]])
// MethodByName is performed on receiver{base: 10} and checked with the same arguments.
// CHECK: store i64 10, ptr %{{.*}}
// CHECK: [[RECEIVER_BOX:%.*]] = insertvalue %"{{.*}}eface" { ptr @_llgo_main.receiver, ptr undef }, ptr %{{.*}}, 1
// CHECK: [[RECEIVER_VALUE:%.*]] = call %reflect.Value @reflect.ValueOf(%"{{.*}}eface" [[RECEIVER_BOX]])
// CHECK: [[METHOD:%.*]] = call %reflect.Value @reflect.Value.MethodByName(%reflect.Value [[RECEIVER_VALUE]], %"{{.*}}String" { ptr @{{.*}}, i64 3 })
// CHECK: call void @main.checkInt(%reflect.Value [[METHOD]], %"{{.*}}Slice" [[MAIN_INTS]])
// CHECK: [[METHOD_IFACE:%.*]] = call %"{{.*}}eface" @reflect.Value.Interface(%reflect.Value [[METHOD]])
// CHECK: [[METHOD_TYPE:%.*]] = extractvalue %"{{.*}}eface" [[METHOD_IFACE]], 0
// CHECK: [[METHOD_MATCH:%.*]] = call i1 @"{{.*}}/runtime/internal/runtime.MatchesClosure"(ptr @"_llgo_closure${{[-A-Za-z0-9_]+}}", ptr [[METHOD_TYPE]])
// CHECK: br i1 [[METHOD_MATCH]], label %{{.*}}, label %{{.*}}
// The floating closure is reflect-called with floatArgs and compared with 55.
// CHECK: [[MAIN_FLOAT_FN:%.*]] = call { ptr, ptr } @main.makeFloatSum(double 1.000000e+01)
// CHECK: store { ptr, ptr } [[MAIN_FLOAT_FN]], ptr [[MAIN_FLOAT_ADDR:%[-A-Za-z0-9_.]+]]
// CHECK: [[MAIN_FLOAT_BOX:%.*]] = insertvalue %"{{.*}}eface" { ptr @"_llgo_closure${{[-A-Za-z0-9_]+}}", ptr undef }, ptr [[MAIN_FLOAT_ADDR]], 1
// CHECK: [[MAIN_FLOAT_VALUE:%.*]] = call %reflect.Value @reflect.ValueOf(%"{{.*}}eface" [[MAIN_FLOAT_BOX]])
// CHECK: [[MAIN_FLOAT_ARGS:%.*]] = call %"{{.*}}Slice" @main.floatArgs()
// CHECK: [[MAIN_FLOAT_RESULTS:%.*]] = call %"{{.*}}Slice" @reflect.Value.Call(%reflect.Value [[MAIN_FLOAT_VALUE]], %"{{.*}}Slice" [[MAIN_FLOAT_ARGS]])
// CHECK: [[MAIN_FLOAT_GOT:%.*]] = call double @reflect.Value.Float(%reflect.Value %{{.*}})
// CHECK: [[MAIN_FLOAT_BAD:%.*]] = fcmp une double [[MAIN_FLOAT_GOT]], 5.500000e+01
// CHECK: br i1 [[MAIN_FLOAT_BAD]], label %{{.*}}, label %{{.*}}
// The asserted method closure keeps code and environment through the direct nine-argument call.
// CHECK: [[METHOD_DATA:%.*]] = extractvalue %"{{.*}}eface" [[METHOD_IFACE]], 1
// CHECK: [[METHOD_FN:%.*]] = load { ptr, ptr }, ptr [[METHOD_DATA]]
// CHECK: [[METHOD_ENV:%.*]] = extractvalue { ptr, ptr } [[METHOD_FN]], 1
// CHECK: [[METHOD_CODE_RAW:%.*]] = extractvalue { ptr, ptr } [[METHOD_FN]], 0
// CHECK: [[METHOD_CODE:%.*]] = call ptr asm "", "=r,0"(ptr [[METHOD_CODE_RAW]])
// CHECK: [[METHOD_GOT:%.*]] = call i64 [[METHOD_CODE]](ptr {{(nest|swiftself)}} [[METHOD_ENV]], i64 1, i64 2, i64 3, i64 4, i64 5, i64 6, i64 7, i64 8, i64 9)
// CHECK: [[METHOD_BAD:%.*]] = icmp ne i64 [[METHOD_GOT]], 55

// CHECK-LABEL: define i64 @"main.main$1"(i64 %0, i64 %1, i64 %2, i64 %3, i64 %4, i64 %5, i64 %6, i64 %7, i64 %8){{.*}} {
// CHECK: ret i64 0

// CHECK-LABEL: define %"{{.*}}Slice" @"main.main$2"(%"{{.*}}Slice" %0){{.*}} {
// CHECK: [[MAKEFUNC_ARGS_LEN:%.*]] = extractvalue %"{{.*}}Slice" %0, 1
// CHECK: [[MAKEFUNC_SUM:%.*]] = phi i64 [ 10, %{{.*}} ], [ [[MAKEFUNC_NEXT:%.*]], %{{.*}} ]
// CHECK: [[MAKEFUNC_ARG:%.*]] = call i64 @reflect.Value.Int(%reflect.Value %{{.*}})
// CHECK: [[MAKEFUNC_NEXT]] = add i64 [[MAKEFUNC_SUM]], [[MAKEFUNC_ARG]]
// CHECK: store i64 [[MAKEFUNC_SUM]], ptr [[MAKEFUNC_RESULT_ADDR:%[-A-Za-z0-9_.]+]]
// CHECK: [[MAKEFUNC_RESULT_BOX:%.*]] = insertvalue %"{{.*}}eface" { ptr @_llgo_int, ptr undef }, ptr [[MAKEFUNC_RESULT_ADDR]], 1
// CHECK: [[MAKEFUNC_RESULT:%.*]] = call %reflect.Value @reflect.ValueOf(%"{{.*}}eface" [[MAKEFUNC_RESULT_BOX]])
// CHECK: store %reflect.Value [[MAKEFUNC_RESULT]], ptr %{{.*}}
// CHECK: ret %"{{.*}}Slice" %{{.*}}

// CHECK-LABEL: define { ptr, ptr } @main.makeFloatSum(double %0){{.*}} {
// CHECK: store double %0, ptr [[FLOAT_BASE_ADDR:%[-A-Za-z0-9_.]+]]
// CHECK: store ptr [[FLOAT_BASE_ADDR]], ptr %{{.*}}
// CHECK: [[FLOAT_CLOSURE:%.*]] = insertvalue { ptr, ptr } { ptr @"main.makeFloatSum$1", ptr undef }, ptr %{{.*}}, 1
// CHECK: ret { ptr, ptr } [[FLOAT_CLOSURE]]

// CHECK-LABEL: define double @"main.makeFloatSum$1"(ptr {{(nest|swiftself)}} %0, double %1, double %2, double %3, double %4, double %5, double %6, double %7, double %8, double %9){{.*}} {
// CHECK: [[FLOAT_ENV:%.*]] = load { ptr }, ptr %0
// CHECK: [[FLOAT_BASE_PTR:%.*]] = extractvalue { ptr } [[FLOAT_ENV]], 0
// CHECK: [[FLOAT_BASE_VALUE:%.*]] = load double, ptr [[FLOAT_BASE_PTR]]
// CHECK: [[FLOAT_S1:%.*]] = fadd double [[FLOAT_BASE_VALUE]], %1
// CHECK: [[FLOAT_S2:%.*]] = fadd double [[FLOAT_S1]], %2
// CHECK: [[FLOAT_S3:%.*]] = fadd double [[FLOAT_S2]], %3
// CHECK: [[FLOAT_S4:%.*]] = fadd double [[FLOAT_S3]], %4
// CHECK: [[FLOAT_S5:%.*]] = fadd double [[FLOAT_S4]], %5
// CHECK: [[FLOAT_S6:%.*]] = fadd double [[FLOAT_S5]], %6
// CHECK: [[FLOAT_S7:%.*]] = fadd double [[FLOAT_S6]], %7
// CHECK: [[FLOAT_S8:%.*]] = fadd double [[FLOAT_S7]], %8
// CHECK: [[FLOAT_S9:%.*]] = fadd double [[FLOAT_S8]], %9
// CHECK: ret double [[FLOAT_S9]]

// CHECK-LABEL: define { ptr, ptr } @main.makeNestedSum(i64 %0){{.*}} {
// CHECK: store i64 %0, ptr [[NESTED_BASE_ADDR:%[-A-Za-z0-9_.]+]]
// CHECK: store ptr [[NESTED_BASE_ADDR]], ptr %{{.*}}
// CHECK: [[NESTED_CLOSURE:%.*]] = insertvalue { ptr, ptr } { ptr @"main.makeNestedSum$1", ptr undef }, ptr %{{.*}}, 1
// CHECK: ret { ptr, ptr } [[NESTED_CLOSURE]]

// CHECK-LABEL: define i64 @"main.makeNestedSum$1"(ptr {{(nest|swiftself)}} %0, i64 %1, i64 %2, i64 %3, i64 %4, i64 %5, i64 %6, i64 %7, i64 %8, i64 %9){{.*}} {
// CHECK: [[NESTED_VALUES:%.*]] = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 216)
// CHECK: [[NESTED_FIRST_PTR:%.*]] = getelementptr inbounds %reflect.Value, ptr [[NESTED_VALUES]], i64 0
// CHECK: store i64 %1, ptr [[NESTED_FIRST_BOX_ADDR:%[-A-Za-z0-9_.]+]]
// CHECK: [[NESTED_FIRST_BOX:%.*]] = insertvalue %"{{.*}}eface" { ptr @_llgo_int, ptr undef }, ptr [[NESTED_FIRST_BOX_ADDR]], 1
// CHECK: [[NESTED_FIRST_VALUE:%.*]] = call %reflect.Value @reflect.ValueOf(%"{{.*}}eface" [[NESTED_FIRST_BOX]])
// CHECK: store %reflect.Value [[NESTED_FIRST_VALUE]], ptr [[NESTED_FIRST_PTR]]
// CHECK: [[NESTED_NINTH_PTR:%.*]] = getelementptr inbounds %reflect.Value, ptr [[NESTED_VALUES]], i64 8
// CHECK: store i64 %9, ptr [[NESTED_NINTH_BOX_ADDR:%[-A-Za-z0-9_.]+]]
// CHECK: [[NESTED_NINTH_BOX:%.*]] = insertvalue %"{{.*}}eface" { ptr @_llgo_int, ptr undef }, ptr [[NESTED_NINTH_BOX_ADDR]], 1
// CHECK: [[NESTED_NINTH_VALUE:%.*]] = call %reflect.Value @reflect.ValueOf(%"{{.*}}eface" [[NESTED_NINTH_BOX]])
// CHECK: store %reflect.Value [[NESTED_NINTH_VALUE]], ptr [[NESTED_NINTH_PTR]]
// CHECK: [[NESTED_ARGS_0:%.*]] = insertvalue %"{{.*}}Slice" undef, ptr [[NESTED_VALUES]], 0
// CHECK: [[NESTED_ARGS_1:%.*]] = insertvalue %"{{.*}}Slice" [[NESTED_ARGS_0]], i64 9, 1
// CHECK: [[NESTED_ARGS:%.*]] = insertvalue %"{{.*}}Slice" [[NESTED_ARGS_1]], i64 9, 2
// CHECK: [[NESTED_ENV:%.*]] = load { ptr }, ptr %0
// CHECK: [[NESTED_CAPTURE_ADDR:%.*]] = extractvalue { ptr } [[NESTED_ENV]], 0
// CHECK: [[NESTED_CAPTURE:%.*]] = load i64, ptr [[NESTED_CAPTURE_ADDR]]
// CHECK: [[NESTED_SUM_FN:%.*]] = call { ptr, ptr } @main.makeSum(i64 [[NESTED_CAPTURE]])
// CHECK: store { ptr, ptr } [[NESTED_SUM_FN]], ptr [[NESTED_SUM_ADDR:%[-A-Za-z0-9_.]+]]
// CHECK: [[NESTED_SUM_BOX:%.*]] = insertvalue %"{{.*}}eface" { ptr @"_llgo_closure${{[-A-Za-z0-9_]+}}", ptr undef }, ptr [[NESTED_SUM_ADDR]], 1
// CHECK: [[NESTED_SUM_VALUE:%.*]] = call %reflect.Value @reflect.ValueOf(%"{{.*}}eface" [[NESTED_SUM_BOX]])
// CHECK: [[NESTED_RESULTS:%.*]] = call %"{{.*}}Slice" @reflect.Value.Call(%reflect.Value [[NESTED_SUM_VALUE]], %"{{.*}}Slice" [[NESTED_ARGS]])
// CHECK: [[NESTED_RESULT:%.*]] = call i64 @reflect.Value.Int(%reflect.Value %{{.*}})
// CHECK: ret i64 [[NESTED_RESULT]]

// CHECK-LABEL: define { ptr, ptr } @main.makeSum(i64 %0){{.*}} {
// CHECK: store i64 %0, ptr [[SUM_BASE_ADDR:%[-A-Za-z0-9_.]+]]
// CHECK: store ptr [[SUM_BASE_ADDR]], ptr %{{.*}}
// CHECK: [[SUM_CLOSURE:%.*]] = insertvalue { ptr, ptr } { ptr @"main.makeSum$1", ptr undef }, ptr %{{.*}}, 1
// CHECK: ret { ptr, ptr } [[SUM_CLOSURE]]

// CHECK-LABEL: define i64 @"main.makeSum$1"(ptr {{(nest|swiftself)}} %0, i64 %1, i64 %2, i64 %3, i64 %4, i64 %5, i64 %6, i64 %7, i64 %8, i64 %9){{.*}} {
// CHECK: [[SUM_ENV:%.*]] = load { ptr }, ptr %0
// CHECK: [[SUM_BASE_PTR:%.*]] = extractvalue { ptr } [[SUM_ENV]], 0
// CHECK: [[SUM_BASE_VALUE:%.*]] = load i64, ptr [[SUM_BASE_PTR]]
// CHECK: [[SUM_S1:%.*]] = add i64 [[SUM_BASE_VALUE]], %1
// CHECK: [[SUM_S2:%.*]] = add i64 [[SUM_S1]], %2
// CHECK: [[SUM_S3:%.*]] = add i64 [[SUM_S2]], %3
// CHECK: [[SUM_S4:%.*]] = add i64 [[SUM_S3]], %4
// CHECK: [[SUM_S5:%.*]] = add i64 [[SUM_S4]], %5
// CHECK: [[SUM_S6:%.*]] = add i64 [[SUM_S5]], %6
// CHECK: [[SUM_S7:%.*]] = add i64 [[SUM_S6]], %7
// CHECK: [[SUM_S8:%.*]] = add i64 [[SUM_S7]], %8
// CHECK: [[SUM_S9:%.*]] = add i64 [[SUM_S8]], %9
// CHECK: ret i64 [[SUM_S9]]

// CHECK-LABEL: define i64 @main.receiver.Sum(%main.receiver %0, i64 %1, i64 %2, i64 %3, i64 %4, i64 %5, i64 %6, i64 %7, i64 %8, i64 %9){{.*}} {
// CHECK: store %main.receiver %0, ptr [[RECEIVER_ADDR:%.*]]
// CHECK: [[RECEIVER_BASE:%.*]] = load i64, ptr %{{.*}}
// CHECK: [[RECEIVER_S1:%.*]] = add i64 [[RECEIVER_BASE]], %1
// CHECK: [[RECEIVER_S2:%.*]] = add i64 [[RECEIVER_S1]], %2
// CHECK: [[RECEIVER_S3:%.*]] = add i64 [[RECEIVER_S2]], %3
// CHECK: [[RECEIVER_S4:%.*]] = add i64 [[RECEIVER_S3]], %4
// CHECK: [[RECEIVER_S5:%.*]] = add i64 [[RECEIVER_S4]], %5
// CHECK: [[RECEIVER_S6:%.*]] = add i64 [[RECEIVER_S5]], %6
// CHECK: [[RECEIVER_S7:%.*]] = add i64 [[RECEIVER_S6]], %7
// CHECK: [[RECEIVER_S8:%.*]] = add i64 [[RECEIVER_S7]], %8
// CHECK: [[RECEIVER_S9:%.*]] = add i64 [[RECEIVER_S8]], %9
// CHECK: ret i64 [[RECEIVER_S9]]

// CHECK-LABEL: define i64 @"main.(*receiver).Sum"(ptr %0, i64 %1, i64 %2, i64 %3, i64 %4, i64 %5, i64 %6, i64 %7, i64 %8, i64 %9){{.*}} {
// CHECK: [[RECEIVER_NIL:%.*]] = icmp eq ptr %0, null
// CHECK: call void @"{{.*}}/runtime/internal/runtime.PanicWrapNilPointer"(i1 [[RECEIVER_NIL]], %"{{.*}}String" {{.*}}, %"{{.*}}String" {{.*}})
// CHECK: [[RECEIVER_VALUE_LOAD:%.*]] = load %main.receiver, ptr %0
// CHECK: [[RECEIVER_WRAPPER_RESULT:%.*]] = call i64 @main.receiver.Sum(%main.receiver [[RECEIVER_VALUE_LOAD]], i64 %1, i64 %2, i64 %3, i64 %4, i64 %5, i64 %6, i64 %7, i64 %8, i64 %9)
// CHECK: ret i64 [[RECEIVER_WRAPPER_RESULT]]

func (r receiver) Sum(a, b, c, d, e, f, g, h, i int) int {
	return r.base + a + b + c + d + e + f + g + h + i
}

func makeSum(base int) func(int, int, int, int, int, int, int, int, int) int {
	return func(a, b, c, d, e, f, g, h, i int) int {
		return base + a + b + c + d + e + f + g + h + i
	}
}

func makeFloatSum(base float64) func(float64, float64, float64, float64, float64, float64, float64, float64, float64) float64 {
	return func(a, b, c, d, e, f, g, h, i float64) float64 {
		return base + a + b + c + d + e + f + g + h + i
	}
}

func makeNestedSum(base int) func(int, int, int, int, int, int, int, int, int) int {
	return func(a, b, c, d, e, f, g, h, i int) int {
		args := []reflect.Value{
			reflect.ValueOf(a), reflect.ValueOf(b), reflect.ValueOf(c),
			reflect.ValueOf(d), reflect.ValueOf(e), reflect.ValueOf(f),
			reflect.ValueOf(g), reflect.ValueOf(h), reflect.ValueOf(i),
		}
		return int(reflect.ValueOf(makeSum(base)).Call(args)[0].Int())
	}
}

func intArgs() []reflect.Value {
	args := make([]reflect.Value, 9)
	for i := range args {
		args[i] = reflect.ValueOf(i + 1)
	}
	return args
}

func floatArgs() []reflect.Value {
	args := make([]reflect.Value, 9)
	for i := range args {
		args[i] = reflect.ValueOf(float64(i + 1))
	}
	return args
}

func checkInt(value reflect.Value, args []reflect.Value) {
	if got := value.Call(args)[0].Int(); got != 55 {
		panic(got)
	}
}

func main() {
	ints := intArgs()
	checkInt(reflect.ValueOf(makeSum(10)), ints)
	checkInt(reflect.ValueOf(makeNestedSum(10)), ints)

	ft := reflect.TypeOf(func(int, int, int, int, int, int, int, int, int) int { return 0 })
	made := reflect.MakeFunc(ft, func(args []reflect.Value) []reflect.Value {
		var sum int64 = 10
		for _, arg := range args {
			sum += arg.Int()
		}
		return []reflect.Value{reflect.ValueOf(int(sum))}
	})
	checkInt(made, ints)

	method := reflect.ValueOf(receiver{base: 10}).MethodByName("Sum")
	checkInt(method, ints)
	bound := method.Interface().(func(int, int, int, int, int, int, int, int, int) int)
	if got := bound(1, 2, 3, 4, 5, 6, 7, 8, 9); got != 55 {
		panic(got)
	}

	if got := reflect.ValueOf(makeFloatSum(10)).Call(floatArgs())[0].Float(); got != 55 {
		panic(got)
	}
	println("ok")
}
