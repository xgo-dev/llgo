#include <stdio.h>
#include <stdlib.h>
#include <inttypes.h>
#include <assert.h>
#include <errno.h>
#include <string.h>
#include "../libexport.h"

static int go_string_equals(GoString got, const char *want) {
    size_t want_len = strlen(want);
    return got.n == (intptr_t)want_len && memcmp(got.p, want, want_len) == 0;
}

static int go_string_has_suffix(GoString got, const char *suffix) {
    size_t suffix_len = strlen(suffix);
    return got.n >= (intptr_t)suffix_len &&
        memcmp(got.p + got.n - suffix_len, suffix, suffix_len) == 0;
}

static intptr_t int_callback(intptr_t value) {
    return value + 700;
}

static GoString string_callback(GoString value) {
    return value;
}

static int void_callback_count;

static void void_callback(void) {
    void_callback_count++;
}

int main() {
    printf("=== C Export Demo ===\n");
    fflush(stdout);  // Force output
    
    // Initialize packages - call init functions first
    github_com_xgo_dev_llgo__demo_go_export_c_init();
    main_init();

    // Verify that funcinfo is not merely linkable: runtime.Callers must yield
    // a symbolized frame and runtime.FuncForPC must resolve that PC's details.
    main_FuncInfoResult func_info = GetFuncInfo();
    assert(func_info.CallersCount > 0);
    assert(func_info.FramePC != 0);
    assert(go_string_equals(func_info.FrameFunction, "main.captureCFuncInfo"));
    assert(go_string_has_suffix(func_info.FrameFile, "export.go"));
    assert(func_info.FrameLine > 0);
    assert(go_string_equals(func_info.FuncName, "main.captureCFuncInfo"));
    assert(func_info.FuncEntry != 0);
    assert(go_string_has_suffix(func_info.FuncFile, "export.go"));
    assert(func_info.FuncLine == func_info.FrameLine);
    printf("FuncInfo: %.*s %.*s:%d\n",
        (int)func_info.FuncName.n, func_info.FuncName.p,
        (int)func_info.FuncFile.n, func_info.FuncFile.p,
        func_info.FuncLine);

    // Test HelloWorld
    HelloWorld();
    printf("\n");

    // Verify that a C library can initialize and call standard-library paths
    // that depend on the runtime hooks supplied by LLGo.
    GoString formatted = FormatValue((GoString){"answer", 6}, 42);
    assert(go_string_equals(formatted, "answer:42"));
#ifdef __linux__
    assert(AllThreadsSyscallStatus() == ENOTSUP);
#else
    assert(AllThreadsSyscallStatus() == 0);
#endif

    // Test small struct
    main_SmallStruct small = CreateSmallStruct(5, 1);  // 1 for true
    assert(small.ID == 5);
    assert(small.Flag == 1);
    printf("Small struct: %d %d\n", small.ID, small.Flag);

    main_SmallStruct processed = ProcessSmallStruct(small);
    assert(processed.ID == 6);
    assert(processed.Flag == 0);
    printf("Processed small: %d %d\n", processed.ID, processed.Flag);

    main_SmallStruct* ptrSmall = ProcessSmallStructPtr(&small);
    if (ptrSmall != NULL) {
        printf("Ptr small: %d %d\n", ptrSmall->ID, ptrSmall->Flag);
    }

    // Test large struct - create GoString for name parameter
    GoString name = {"test_large", 10};  // name and length
    main_LargeStruct large = CreateLargeStruct(12345, name);
    assert(large.ID == 12345);
    printf("Large struct ID: %" PRId64 "\n", large.ID);

    int64_t total = ProcessLargeStruct(large);
    printf("Large struct total: %" PRId64 "\n", total);

    main_LargeStruct* ptrLarge = ProcessLargeStructPtr(&large);
    if (ptrLarge != NULL) {
        printf("Ptr large ID: %" PRId64 "\n", ptrLarge->ID);
    }

    // Test self-referential struct
    main_Node* node1 = CreateNode(100);
    main_Node* node2 = CreateNode(200);
    int link_result = LinkNodes(node1, node2);
    assert(link_result == 300);  // LinkNodes returns 100 + 200 = 300
    printf("LinkNodes result: %d\n", link_result);

    int count = TraverseNodes(node1);
    assert(count == 2);  // Should traverse 2 nodes
    printf("Node count: %d\n", count);

    // Test basic types with assertions
    assert(ProcessBool(1) == 0);  // ProcessBool(true) returns !true = false
    printf("Bool: %d\n", ProcessBool(1));
    
    assert(ProcessInt8(10) == 11);  // ProcessInt8(x) returns x + 1
    printf("Int8: %d\n", ProcessInt8(10));
    
    assert(ProcessUint8(10) == 11);  // ProcessUint8(x) returns x + 1
    printf("Uint8: %d\n", ProcessUint8(10));
    
    assert(ProcessInt16(10) == 20);  // ProcessInt16(x) returns x * 2
    printf("Int16: %d\n", ProcessInt16(10));
    
    assert(ProcessUint16(10) == 20);  // ProcessUint16(x) returns x * 2
    printf("Uint16: %d\n", ProcessUint16(10));
    
    assert(ProcessInt32(10) == 30);  // ProcessInt32(x) returns x * 3
    printf("Int32: %d\n", ProcessInt32(10));
    
    assert(ProcessUint32(10) == 30);  // ProcessUint32(x) returns x * 3
    printf("Uint32: %u\n", ProcessUint32(10));
    
    assert(ProcessInt64(10) == 40);  // ProcessInt64(x) returns x * 4
    printf("Int64: %" PRId64 "\n", ProcessInt64(10));
    
    assert(ProcessUint64(10) == 40);  // ProcessUint64(x) returns x * 4
    printf("Uint64: %" PRIu64 "\n", ProcessUint64(10));
    
    assert(ProcessInt(10) == 110);  // ProcessInt(x) returns x * 11
    printf("Int: %ld\n", ProcessInt(10));
    
    assert(ProcessUint(10) == 210);  // ProcessUint(x) returns x * 21
    printf("Uint: %lu\n", ProcessUint(10));
    
    assert(ProcessUintptr(0x1000) == 4396);  // ProcessUintptr(x) returns x + 300 = 4096 + 300
    printf("Uintptr: %lu\n", ProcessUintptr(0x1000));
    
    // Float comparisons with tolerance
    float f32_result = ProcessFloat32(3.14f);
    assert(f32_result > 4.7f && f32_result < 4.72f);  // ProcessFloat32(x) returns x * 1.5 ≈ 4.71
    printf("Float32: %f\n", f32_result);
    
    double f64_result = ProcessFloat64(3.14);
    assert(f64_result > 7.84 && f64_result < 7.86);  // ProcessFloat64(x) returns x * 2.5 ≈ 7.85
    printf("Float64: %f\n", f64_result);

    GoString raw_string = {"value", 5};
    GoString processed_string = ProcessString(raw_string);
    assert(go_string_equals(processed_string, "processed_value"));
    assert(RunGoroutine(41) == 42);
    assert(go_string_equals(ProcessMyString(raw_string), "modified_value"));
    assert(go_string_equals(TwoParams(65, (GoString){"bc", 2}), "Abc"));
    assert(go_string_equals(MultipleParams(1, 2, 3, 4, 5.0f, 6.0,
        (GoString){"multi", 5}, 1), "multi_B23_true_4_5_6"));

    // Test unsafe pointer
    int test_val = 42;
    void* ptr_result = ProcessUnsafePointer(&test_val);
    printf("UnsafePointer: %p\n", ptr_result);

    // Test named types
    main_MyInt myInt = ProcessMyInt(42);
    printf("MyInt: %ld\n", (long)myInt);

    // Test arrays
    Array_intptr_t_5 arr = {.data = {1, 2, 3, 4, 5}};
    intptr_t arr_sum = ProcessIntArray(arr);
    assert(arr_sum == 15);
    printf("Array sum: %ld\n", (long)arr_sum);

    // Test complex data with multidimensional arrays
    main_ComplexData complex = CreateComplexData();
    assert(ProcessComplexData(complex) == 78);
    assert(complex.IntArray[0] == 10 && complex.IntArray[4] == 50);
    assert(complex.Slices.len == 1);
    assert(complex.DataList.len == 1);
    assert(((double*)complex.DataList.data)[0] == 1.0);
    printf("Complex data matrix sum: %" PRId32 "\n", ProcessComplexData(complex));

    intptr_t c_slice_values[] = {3, 6, 9};
    GoSlice c_slice = {c_slice_values, 3, 3};
    assert(ProcessIntSlice(c_slice) == 18);
    GoSlice int_slice = CreateIntSlice();
    assert(int_slice.len == 4);
    assert(ProcessIntSlice(int_slice) == 20);
    GoMap string_map = CreateStringMap();
    assert(string_map.data != NULL);
    assert(ProcessStringMap(string_map) == 60);
    GoChan int_channel = CreateIntChannel();
    assert(int_channel.data != NULL);
    assert(ProcessIntChannel(int_channel) == 321);
    GoInterface int_interface = CreateIntInterface();
    GoInterface string_interface = CreateStringInterface();
    assert(ProcessInterface(int_interface) == 123);
    assert(ProcessInterface(string_interface) == 40);

    // Test various parameter counts
    assert(NoParams() == 42);  // NoParams() always returns 42
    printf("NoParams: %ld\n", NoParams());
    
    assert(OneParam(5) == 10);  // OneParam(x) returns x * 2
    printf("OneParam: %ld\n", OneParam(5));
    
    assert(ThreeParams(10, 2.5, 1) == 25.0);  // ThreeParams calculates result
    printf("ThreeParams: %f\n", ThreeParams(10, 2.5, 1));  // 1 for true
    
    // Test ProcessThreeUnnamedParams - now uses all parameters
    GoString test_str = {"hello", 5};
    double unnamed_result = ProcessThreeUnnamedParams(10, test_str, 1);
    assert(unnamed_result == 22.5);  // (10 + 5) * 1.5 = 22.5
    printf("ProcessThreeUnnamedParams: %f\n", unnamed_result);
    
    assert(ProcessWithIntCallback(23, int_callback) == 723);
    assert(go_string_equals(ProcessWithStringCallback(raw_string, string_callback), "value"));
    assert(ProcessWithVoidCallback(void_callback) == 123);
    assert(void_callback_count == 1);

    // Test ProcessWithVoidCallback - now returns int
    int void_callback_result = ProcessWithVoidCallback(NULL);
    assert(void_callback_result == 456);  // Returns 456 when callback is nil
    printf("ProcessWithVoidCallback(NULL): %d\n", void_callback_result);
    
    // Test NoParamNames - function with unnamed parameters
    int32_t no_names_result = NoParamNames(5, 10, 0);
    assert(no_names_result == 789);  // Returns fixed value 789
    printf("NoParamNames: %d\n", no_names_result);

    // Test XType from c package - create GoString for name parameter
    GoString xname = {"test_x", 6};  // name and length
    C_XType xtype = CreateXType(42, xname, 3.14, 1);  // 1 for true
    printf("XType: %d %f %d\n", xtype.ID, xtype.Value, xtype.Flag);

    C_XType processedX = ProcessXType(xtype);
    printf("Processed XType: %d %f %d\n", processedX.ID, processedX.Value, processedX.Flag);

    C_XType* ptrX = ProcessXTypePtr(&xtype);
    if (ptrX != NULL) {
        printf("Ptr XType: %d %f %d\n", ptrX->ID, ptrX->Value, ptrX->Flag);
    }

    // Test multidimensional arrays
    printf("\n=== Multidimensional Array Tests ===\n");
    
    // Create and test 2D matrix [3][4]
    // Note: CreateMatrix2D returns [3][4]int32, but function returns need special handling in C
    printf("Testing 2D matrix functions...\n");
    
    // Create a test 2D matrix [3][4]int32
    Array_int32_t_3_4 test_matrix = {.data = {
        {1, 2, 3, 4},
        {5, 6, 7, 8},
        {9, 10, 11, 12}
    }};
    int32_t matrix_sum = ProcessMatrix2D(test_matrix);
    assert(matrix_sum == 78);  // Sum of 1+2+3+...+12 = 78
    printf("Matrix2D sum: %d\n", matrix_sum);
    
    // Create a test 3D cube [2][3][4]uint8
    Array_uint8_t_2_3_4 test_cube;
    uint8_t val = 1;
    for (int i = 0; i < 2; i++) {
        for (int j = 0; j < 3; j++) {
            for (int k = 0; k < 4; k++) {
                test_cube.data[i][j][k] = val++;
            }
        }
    }
    uint32_t cube_sum = ProcessMatrix3D(test_cube);
    assert(cube_sum == 300);  // Sum of 1+2+3+...+24 = 300
    printf("Matrix3D (cube) sum: %u\n", cube_sum);
    
    // Create a test 5x4 grid [5][4]double
    Array_double_5_4 test_grid;
    double grid_val = 1.0;
    for (int i = 0; i < 5; i++) {
        for (int j = 0; j < 4; j++) {
            test_grid.data[i][j] = grid_val;
            grid_val += 0.5;
        }
    }
    double grid_sum = ProcessGrid5x4(test_grid);
    assert(grid_sum == 115.0);  // Sum of 1.0+1.5+2.0+...+10.5 = 115.0
    printf("Grid5x4 sum: %f\n", grid_sum);
    
    // Test functions that return multidimensional arrays (as multi-level pointers)
    printf("\n=== Testing Return Value Functions ===\n");
    
    // Test CreateMatrix1D() which returns Array_int32_t_4
    printf("About to call CreateMatrix1D()...\n");
    fflush(stdout);
    Array_int32_t_4 matrix1d = CreateMatrix1D();
    assert(matrix1d.data[0] == 1);
    printf("CreateMatrix1D() call completed\n");
    printf("CreateMatrix1D() returned struct, first element: %d\n", matrix1d.data[0]);
    
    // Test CreateMatrix2D() which returns Array_int32_t_3_4
    printf("About to call CreateMatrix2D()...\n");
    fflush(stdout);
    Array_int32_t_3_4 matrix2d = CreateMatrix2D();
    assert(matrix2d.data[0][0] == 1);
    printf("CreateMatrix2D() call completed\n");
    printf("CreateMatrix2D() returned struct, first element: %d\n", matrix2d.data[0][0]);
    
    // Test CreateMatrix3D() which returns Array_uint8_t_2_3_4
    Array_uint8_t_2_3_4 cube = CreateMatrix3D();
    assert(cube.data[0][0][0] == 1);
    printf("CreateMatrix3D() returned struct, first element: %u\n", cube.data[0][0][0]);
    
    // Test CreateGrid5x4() which returns Array_double_5_4
    Array_double_5_4 grid = CreateGrid5x4();
    assert(grid.data[0][0] == 1.0);
    printf("CreateGrid5x4() returned struct, first element: %f\n", grid.data[0][0]);

    // Test a void function with a string parameter.
    NoReturn((GoString){"called from C", 13});

    printf("C demo completed!\n");

    return 0;
}
