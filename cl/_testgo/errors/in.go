// LITTEST
package main

// New returns an error that formats as the given text.
// Each call to New returns a distinct error value even if the text is identical.
// CHECK-LABEL: define %"{{.*}}iface" @main.New(%"{{.*}}String" %0){{.*}} {
// CHECK: [[NEW_ERROR:%[0-9]+]] = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 16)
// CHECK-NEXT: [[NEW_ERROR_FIELD:%[0-9]+]] = getelementptr inbounds %main.errorString, ptr [[NEW_ERROR]], i32 0, i32 0
// CHECK-NEXT: store %"{{.*}}String" %0, ptr [[NEW_ERROR_FIELD]]
// CHECK: [[NEW_ERROR_ITAB:%[0-9]+]] = call ptr @"{{.*}}NewItab"(ptr @"{{.*}}iface{{.*}}", ptr @"*_llgo_main.errorString")
// CHECK-NEXT: [[NEW_ERROR_IFACE0:%[0-9]+]] = insertvalue %"{{.*}}iface" undef, ptr [[NEW_ERROR_ITAB]], 0
// CHECK-NEXT: [[NEW_ERROR_IFACE:%[0-9]+]] = insertvalue %"{{.*}}iface" [[NEW_ERROR_IFACE0]], ptr [[NEW_ERROR]], 1
// CHECK-NEXT: ret %"{{.*}}iface" [[NEW_ERROR_IFACE]]
func New(text string) error {
	return &errorString{text}
}

// errorString is a trivial implementation of error.
type errorString struct {
	s string
}

// CHECK-LABEL: define %"{{.*}}String" @"main.(*errorString).Error"(ptr %0){{.*}} {
// CHECK: [[ERROR_FIELD:%[0-9]+]] = getelementptr inbounds %main.errorString, ptr %0, i32 0, i32 0
// CHECK-NEXT: [[ERROR_TEXT:%[0-9]+]] = load %"{{.*}}String", ptr [[ERROR_FIELD]]
// CHECK-NEXT: ret %"{{.*}}String" [[ERROR_TEXT]]
func (e *errorString) Error() string {
	return e.s
}

// CHECK-LABEL: define void @main.main(){{.*}} {
func main() {
	// CHECK: call %"{{.*}}iface" @main.New(%"{{.*}}String" { ptr @{{[0-9]+}}, i64 8 })
	// CHECK: call ptr @"{{.*}}IfacePtrData"(%"{{.*}}iface" %{{[0-9]+}})
	err := New("an error")
	println(err)
	println(err.Error())
}
