// LITTEST
package main

// New returns an error that formats as the given text.
// Each call to New returns a distinct error value even if the text is identical.
// CHECK-LABEL: define %"{{.*}}iface" @"{{.*}}errors.New"(%"{{.*}}String" %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = call ptr @"{{.*}}AllocZ"(i64 16)
// CHECK-NEXT:   %2 = getelementptr inbounds %"{{.*}}errors.errorString", ptr %1, i32 0, i32 0
// CHECK-NEXT:   store %"{{.*}}String" %0, ptr %2, align 8
// CHECK-NEXT:   %3 = call ptr @"{{.*}}NewItab"(ptr @"{{.*}}iface{{.*}}", ptr @"{{.*}}errors.errorString")
// CHECK-NEXT:   %4 = insertvalue %"{{.*}}iface" undef, ptr %3, 0
// CHECK-NEXT:   %5 = insertvalue %"{{.*}}iface" %4, ptr %1, 1
// CHECK-NEXT:   ret %"{{.*}}iface" %5
func New(text string) error {
	return &errorString{text}
}

// errorString is a trivial implementation of error.
type errorString struct {
	s string
}

// CHECK-LABEL: define %"{{.*}}String" @"{{.*}}errors.(*errorString).Error"(ptr %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = getelementptr inbounds %"{{.*}}errors.errorString", ptr %0, i32 0, i32 0
// CHECK-NEXT:   %2 = load %"{{.*}}String", ptr %1, align 8
// CHECK-NEXT:   ret %"{{.*}}String" %2
func (e *errorString) Error() string {
	return e.s
}

// CHECK-LABEL: define void @"{{.*}}errors.main"(){{.*}} {
func main() {
	// CHECK: call %"{{.*}}iface" @"{{.*}}errors.New"(%"{{.*}}String" { ptr @7, i64 8 })
	// CHECK: call void @"{{.*}}PrintIface"(%"{{.*}}iface" %0)
	// CHECK: call ptr @"{{.*}}IfacePtrData"(%"{{.*}}iface" %0)
	// CHECK: call %"{{.*}}String" %8(ptr %7)
	// CHECK: call void @"{{.*}}PrintString"(%"{{.*}}String" %9)
	// CHECK: call void @"{{.*}}PrintByte"(i8 10)
	// CHECK: ret void
	err := New("an error")
	println(err)
	println(err.Error())
}
