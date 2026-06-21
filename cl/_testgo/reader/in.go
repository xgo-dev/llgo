// LITTEST
package main

import (
	"unicode/utf8"
)

// CHECK: @2 = private unnamed_addr constant [7 x i8] c"WriteTo", align 1
// CHECK: @17 = private unnamed_addr constant [5 x i8] c"Close", align 1
// CHECK: @28 = private unnamed_addr constant [3 x i8] c"EOF", align 1
// CHECK: @29 = private unnamed_addr constant [11 x i8] c"short write", align 1
// CHECK: @30 = private unnamed_addr constant [11 x i8] c"hello world", align 1
// CHECK: @53 = private unnamed_addr constant [50 x i8] c"{{.*}}/cl/_testgo/reader.nopCloser", align 1
// CHECK: @54 = private unnamed_addr constant [49 x i8] c"{{.*}}/cl/_testgo/reader.WriterTo", align 1
// CHECK: @55 = private unnamed_addr constant [58 x i8] c"{{.*}}/cl/_testgo/reader.nopCloserWriterTo", align 1
// CHECK: @56 = private unnamed_addr constant [37 x i8] c"stringsReader.ReadAt: negative offset", align 1
// CHECK: @57 = private unnamed_addr constant [34 x i8] c"stringsReader.Seek: invalid whence", align 1
// CHECK: @58 = private unnamed_addr constant [37 x i8] c"stringsReader.Seek: negative position", align 1
// CHECK: @59 = private unnamed_addr constant [48 x i8] c"stringsReader.UnreadByte: at beginning of string", align 1
// CHECK: @60 = private unnamed_addr constant [49 x i8] c"strings.Reader.UnreadRune: at beginning of string", align 1
// CHECK: @61 = private unnamed_addr constant [62 x i8] c"strings.Reader.UnreadRune: previous operation was not ReadRune", align 1
// CHECK: @62 = private unnamed_addr constant [48 x i8] c"stringsReader.WriteTo: invalid WriteString count", align 1

type Reader interface {
	Read(p []byte) (n int, err error)
}

type Writer interface {
	Write(p []byte) (n int, err error)
}

type Closer interface {
	Close() error
}

type Seeker interface {
	Seek(offset int64, whence int) (int64, error)
}

type ReadWriter interface {
	Reader
	Writer
}

type ReadCloser interface {
	Reader
	Closer
}

type WriteCloser interface {
	Writer
	Closer
}

type ReadWriteCloser interface {
	Reader
	Writer
	Closer
}

type ReadSeeker interface {
	Reader
	Seeker
}

type ReadSeekCloser interface {
	Reader
	Seeker
	Closer
}

type WriteSeeker interface {
	Writer
	Seeker
}

type ReadWriteSeeker interface {
	Reader
	Writer
	Seeker
}

type ReaderFrom interface {
	ReadFrom(r Reader) (n int64, err error)
}

type WriterTo interface {
	WriteTo(w Writer) (n int64, err error)
}

type ReaderAt interface {
	ReadAt(p []byte, off int64) (n int, err error)
}

type WriterAt interface {
	WriteAt(p []byte, off int64) (n int, err error)
}

type ByteReader interface {
	ReadByte() (byte, error)
}

type ByteScanner interface {
	ByteReader
	UnreadByte() error
}

type ByteWriter interface {
	WriteByte(c byte) error
}

type RuneReader interface {
	ReadRune() (r rune, size int, err error)
}

type RuneScanner interface {
	RuneReader
	UnreadRune() error
}

type StringWriter interface {
	WriteString(s string) (n int, err error)
}

func WriteString(w Writer, s string) (n int, err error) {
	if sw, ok := w.(StringWriter); ok {
		return sw.WriteString(s)
	}
	return w.Write([]byte(s))
}

// CHECK-LABEL: define %"{{.*}}/runtime/internal/runtime.iface" @"{{.*}}/cl/_testgo/reader.NopCloser"(%"{{.*}}/runtime/internal/runtime.iface" %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = call ptr @"{{.*}}/runtime/internal/runtime.IfaceType"(%"{{.*}}/runtime/internal/runtime.iface" %0)
// CHECK-NEXT:   %2 = call i1 @"{{.*}}/runtime/internal/runtime.Implements"(ptr @"_llgo_{{.*}}/cl/_testgo/reader.WriterTo", ptr %1)
// CHECK-NEXT:   br i1 %2, label %_llgo_3, label %_llgo_4
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_5
// CHECK-NEXT:   %3 = alloca %"{{.*}}/cl/_testgo/reader.nopCloserWriterTo", align 8
// CHECK-NEXT:   call void @llvm.memset(ptr %3, i8 0, i64 16, i1 false)
// CHECK-NEXT:   %4 = getelementptr inbounds %"{{.*}}/cl/_testgo/reader.nopCloserWriterTo", ptr %3, i32 0, i32 0
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.iface" %0, ptr %4, align 8
// CHECK-NEXT:   %5 = load %"{{.*}}/cl/_testgo/reader.nopCloserWriterTo", ptr %3, align 8
// CHECK-NEXT:   %6 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/cl/_testgo/reader.nopCloserWriterTo" %5, ptr %6, align 8
// CHECK-NEXT:   %7 = call ptr @"{{.*}}/runtime/internal/runtime.NewItab"(ptr @"_llgo_iface$2bmbYDBStAIdmbXPPn7qIaCcpVcj2I5k6AqgqwAfh84", ptr @"_llgo_{{.*}}/cl/_testgo/reader.nopCloserWriterTo")
// CHECK-NEXT:   %8 = insertvalue %"{{.*}}/runtime/internal/runtime.iface" undef, ptr %7, 0
// CHECK-NEXT:   %9 = insertvalue %"{{.*}}/runtime/internal/runtime.iface" %8, ptr %6, 1
// CHECK-NEXT:   ret %"{{.*}}/runtime/internal/runtime.iface" %9
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_5
// CHECK-NEXT:   %10 = alloca %"{{.*}}/cl/_testgo/reader.nopCloser", align 8
// CHECK-NEXT:   call void @llvm.memset(ptr %10, i8 0, i64 16, i1 false)
// CHECK-NEXT:   %11 = getelementptr inbounds %"{{.*}}/cl/_testgo/reader.nopCloser", ptr %10, i32 0, i32 0
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.iface" %0, ptr %11, align 8
// CHECK-NEXT:   %12 = load %"{{.*}}/cl/_testgo/reader.nopCloser", ptr %10, align 8
// CHECK-NEXT:   %13 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/cl/_testgo/reader.nopCloser" %12, ptr %13, align 8
// CHECK-NEXT:   %14 = call ptr @"{{.*}}/runtime/internal/runtime.NewItab"(ptr @"_llgo_iface$2bmbYDBStAIdmbXPPn7qIaCcpVcj2I5k6AqgqwAfh84", ptr @"_llgo_{{.*}}/cl/_testgo/reader.nopCloser")
// CHECK-NEXT:   %15 = insertvalue %"{{.*}}/runtime/internal/runtime.iface" undef, ptr %14, 0
// CHECK-NEXT:   %16 = insertvalue %"{{.*}}/runtime/internal/runtime.iface" %15, ptr %13, 1
// CHECK-NEXT:   ret %"{{.*}}/runtime/internal/runtime.iface" %16
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_3:                                          ; preds = %_llgo_0
// CHECK-NEXT:   %17 = extractvalue %"{{.*}}/runtime/internal/runtime.iface" %0, 1
// CHECK-NEXT:   %18 = call ptr @"{{.*}}/runtime/internal/runtime.NewItab"(ptr @"_llgo_iface$p5Bo_emI1h8acs1rFbUxZTrpeDbIQ34gFcsbwK9YIgs", ptr %1)
// CHECK-NEXT:   %19 = insertvalue %"{{.*}}/runtime/internal/runtime.iface" undef, ptr %18, 0
// CHECK-NEXT:   %20 = insertvalue %"{{.*}}/runtime/internal/runtime.iface" %19, ptr %17, 1
// CHECK-NEXT:   %21 = insertvalue { %"{{.*}}/runtime/internal/runtime.iface", i1 } undef, %"{{.*}}/runtime/internal/runtime.iface" %20, 0
// CHECK-NEXT:   %22 = insertvalue { %"{{.*}}/runtime/internal/runtime.iface", i1 } %21, i1 true, 1
// CHECK-NEXT:   br label %_llgo_5
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_4:                                          ; preds = %_llgo_0
// CHECK-NEXT:   br label %_llgo_5
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_5:                                          ; preds = %_llgo_4, %_llgo_3
// CHECK-NEXT:   %23 = phi { %"{{.*}}/runtime/internal/runtime.iface", i1 } [ %22, %_llgo_3 ], [ zeroinitializer, %_llgo_4 ]
// CHECK-NEXT:   %24 = extractvalue { %"{{.*}}/runtime/internal/runtime.iface", i1 } %23, 0
// CHECK-NEXT:   %25 = extractvalue { %"{{.*}}/runtime/internal/runtime.iface", i1 } %23, 1
// CHECK-NEXT:   br i1 %25, label %_llgo_1, label %_llgo_2
// CHECK-NEXT: }

func NopCloser(r Reader) ReadCloser {
	if _, ok := r.(WriterTo); ok {
		return nopCloserWriterTo{r}
	}
	return nopCloser{r}
}

type nopCloser struct {
	Reader
}

func (nopCloser) Close() error { return nil }

type nopCloserWriterTo struct {
	Reader
}

func (nopCloserWriterTo) Close() error { return nil }

func (c nopCloserWriterTo) WriteTo(w Writer) (n int64, err error) {
	return c.Reader.(WriterTo).WriteTo(w)
}

// CHECK-LABEL: define { %"{{.*}}/runtime/internal/runtime.Slice", %"{{.*}}/runtime/internal/runtime.iface" } @"{{.*}}/cl/_testgo/reader.ReadAll"(%"{{.*}}/runtime/internal/runtime.iface" %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 512)
// CHECK-NEXT:   %2 = call %"{{.*}}/runtime/internal/runtime.Slice" @"{{.*}}/runtime/internal/runtime.NewSlice2"(ptr %1, i64 1, i64 512, i64 0, i64 0, i1 true, i1 true, i1 true)
// CHECK-NEXT:   br label %_llgo_1
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_6, %_llgo_3, %_llgo_0
// CHECK-NEXT:   %3 = phi %"{{.*}}/runtime/internal/runtime.Slice" [ %2, %_llgo_0 ], [ %24, %_llgo_3 ], [ %61, %_llgo_6 ]
// CHECK-NEXT:   %4 = extractvalue %"{{.*}}/runtime/internal/runtime.Slice" %3, 1
// CHECK-NEXT:   %5 = extractvalue %"{{.*}}/runtime/internal/runtime.Slice" %3, 2
// CHECK-NEXT:   %6 = extractvalue %"{{.*}}/runtime/internal/runtime.Slice" %3, 2
// CHECK-NEXT:   %7 = extractvalue %"{{.*}}/runtime/internal/runtime.Slice" %3, 0
// CHECK-NEXT:   %8 = call %"{{.*}}/runtime/internal/runtime.Slice" @"{{.*}}/runtime/internal/runtime.NewSlice2"(ptr %7, i64 1, i64 %6, i64 %4, i64 %5, i1 true, i1 true, i1 false)
// CHECK-NEXT:   %9 = call ptr @"{{.*}}/runtime/internal/runtime.IfacePtrData"(%"{{.*}}/runtime/internal/runtime.iface" %0)
// CHECK-NEXT:   %10 = extractvalue %"{{.*}}/runtime/internal/runtime.iface" %0, 0
// CHECK-NEXT:   %11 = getelementptr ptr, ptr %10, i64 3
// CHECK-NEXT:   %12 = load ptr, ptr %11, align 8
// CHECK-NEXT:   %13 = insertvalue { ptr, ptr } undef, ptr %12, 0
// CHECK-NEXT:   %14 = insertvalue { ptr, ptr } %13, ptr %9, 1
// CHECK-NEXT:   %15 = extractvalue { ptr, ptr } %14, 1
// CHECK-NEXT:   %16 = extractvalue { ptr, ptr } %14, 0
// CHECK-NEXT:   %17 = call { i64, %"{{.*}}/runtime/internal/runtime.iface" } %16(ptr %15, %"{{.*}}/runtime/internal/runtime.Slice" %8)
// CHECK-NEXT:   %18 = extractvalue { i64, %"{{.*}}/runtime/internal/runtime.iface" } %17, 0
// CHECK-NEXT:   %19 = extractvalue { i64, %"{{.*}}/runtime/internal/runtime.iface" } %17, 1
// CHECK-NEXT:   %20 = extractvalue %"{{.*}}/runtime/internal/runtime.Slice" %3, 1
// CHECK-NEXT:   %21 = add i64 %20, %18
// CHECK-NEXT:   %22 = extractvalue %"{{.*}}/runtime/internal/runtime.Slice" %3, 2
// CHECK-NEXT:   %23 = extractvalue %"{{.*}}/runtime/internal/runtime.Slice" %3, 0
// CHECK-NEXT:   %24 = call %"{{.*}}/runtime/internal/runtime.Slice" @"{{.*}}/runtime/internal/runtime.NewSlice2"(ptr %23, i64 1, i64 %22, i64 0, i64 %21, i1 true, i1 true, i1 false)
// CHECK-NEXT:   %25 = call ptr @"{{.*}}/runtime/internal/runtime.IfaceType"(%"{{.*}}/runtime/internal/runtime.iface" %19)
// CHECK-NEXT:   %26 = extractvalue %"{{.*}}/runtime/internal/runtime.iface" %19, 1
// CHECK-NEXT:   %27 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" undef, ptr %25, 0
// CHECK-NEXT:   %28 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" %27, ptr %26, 1
// CHECK-NEXT:   %29 = call ptr @"{{.*}}/runtime/internal/runtime.IfaceType"(%"{{.*}}/runtime/internal/runtime.iface" zeroinitializer)
// CHECK-NEXT:   %30 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" undef, ptr %29, 0
// CHECK-NEXT:   %31 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" %30, ptr null, 1
// CHECK-NEXT:   %32 = call i1 @"{{.*}}/runtime/internal/runtime.EfaceEqual"(%"{{.*}}/runtime/internal/runtime.eface" %28, %"{{.*}}/runtime/internal/runtime.eface" %31)
// CHECK-NEXT:   %33 = xor i1 %32, true
// CHECK-NEXT:   br i1 %33, label %_llgo_2, label %_llgo_3
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_1
// CHECK-NEXT:   %34 = load %"{{.*}}/runtime/internal/runtime.iface", ptr @"{{.*}}/cl/_testgo/reader.EOF", align 8
// CHECK-NEXT:   %35 = call ptr @"{{.*}}/runtime/internal/runtime.IfaceType"(%"{{.*}}/runtime/internal/runtime.iface" %19)
// CHECK-NEXT:   %36 = extractvalue %"{{.*}}/runtime/internal/runtime.iface" %19, 1
// CHECK-NEXT:   %37 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" undef, ptr %35, 0
// CHECK-NEXT:   %38 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" %37, ptr %36, 1
// CHECK-NEXT:   %39 = call ptr @"{{.*}}/runtime/internal/runtime.IfaceType"(%"{{.*}}/runtime/internal/runtime.iface" %34)
// CHECK-NEXT:   %40 = extractvalue %"{{.*}}/runtime/internal/runtime.iface" %34, 1
// CHECK-NEXT:   %41 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" undef, ptr %39, 0
// CHECK-NEXT:   %42 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" %41, ptr %40, 1
// CHECK-NEXT:   %43 = call i1 @"{{.*}}/runtime/internal/runtime.EfaceEqual"(%"{{.*}}/runtime/internal/runtime.eface" %38, %"{{.*}}/runtime/internal/runtime.eface" %42)
// CHECK-NEXT:   br i1 %43, label %_llgo_4, label %_llgo_5
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_3:                                          ; preds = %_llgo_1
// CHECK-NEXT:   %44 = extractvalue %"{{.*}}/runtime/internal/runtime.Slice" %24, 1
// CHECK-NEXT:   %45 = extractvalue %"{{.*}}/runtime/internal/runtime.Slice" %24, 2
// CHECK-NEXT:   %46 = icmp eq i64 %44, %45
// CHECK-NEXT:   br i1 %46, label %_llgo_6, label %_llgo_1
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_4:                                          ; preds = %_llgo_2
// CHECK-NEXT:   br label %_llgo_5
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_5:                                          ; preds = %_llgo_4, %_llgo_2
// CHECK-NEXT:   %47 = phi %"{{.*}}/runtime/internal/runtime.iface" [ %19, %_llgo_2 ], [ zeroinitializer, %_llgo_4 ]
// CHECK-NEXT:   %48 = insertvalue { %"{{.*}}/runtime/internal/runtime.Slice", %"{{.*}}/runtime/internal/runtime.iface" } undef, %"{{.*}}/runtime/internal/runtime.Slice" %24, 0
// CHECK-NEXT:   %49 = insertvalue { %"{{.*}}/runtime/internal/runtime.Slice", %"{{.*}}/runtime/internal/runtime.iface" } %48, %"{{.*}}/runtime/internal/runtime.iface" %47, 1
// CHECK-NEXT:   ret { %"{{.*}}/runtime/internal/runtime.Slice", %"{{.*}}/runtime/internal/runtime.iface" } %49
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_6:                                          ; preds = %_llgo_3
// CHECK-NEXT:   %50 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 1)
// CHECK-NEXT:   %51 = getelementptr inbounds i8, ptr %50, i64 0
// CHECK-NEXT:   store i8 0, ptr %51, align 1
// CHECK-NEXT:   %52 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" undef, ptr %50, 0
// CHECK-NEXT:   %53 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" %52, i64 1, 1
// CHECK-NEXT:   %54 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" %53, i64 1, 2
// CHECK-NEXT:   %55 = extractvalue %"{{.*}}/runtime/internal/runtime.Slice" %54, 0
// CHECK-NEXT:   %56 = extractvalue %"{{.*}}/runtime/internal/runtime.Slice" %54, 1
// CHECK-NEXT:   %57 = call %"{{.*}}/runtime/internal/runtime.Slice" @"{{.*}}/runtime/internal/runtime.SliceAppend"(%"{{.*}}/runtime/internal/runtime.Slice" %24, ptr %55, i64 %56, i64 1)
// CHECK-NEXT:   %58 = extractvalue %"{{.*}}/runtime/internal/runtime.Slice" %24, 1
// CHECK-NEXT:   %59 = extractvalue %"{{.*}}/runtime/internal/runtime.Slice" %57, 2
// CHECK-NEXT:   %60 = extractvalue %"{{.*}}/runtime/internal/runtime.Slice" %57, 0
// CHECK-NEXT:   %61 = call %"{{.*}}/runtime/internal/runtime.Slice" @"{{.*}}/runtime/internal/runtime.NewSlice2"(ptr %60, i64 1, i64 %59, i64 0, i64 %58, i1 true, i1 true, i1 false)
// CHECK-NEXT:   br label %_llgo_1
// CHECK-NEXT: }

// CHECK-LABEL: define { i64, %"{{.*}}/runtime/internal/runtime.iface" } @"{{.*}}/cl/_testgo/reader.WriteString"(%"{{.*}}/runtime/internal/runtime.iface" %0, %"{{.*}}/runtime/internal/runtime.String" %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = call ptr @"{{.*}}/runtime/internal/runtime.IfaceType"(%"{{.*}}/runtime/internal/runtime.iface" %0)
// CHECK-NEXT:   %3 = call i1 @"{{.*}}/runtime/internal/runtime.Implements"(ptr @"_llgo_{{.*}}/cl/_testgo/reader.StringWriter", ptr %2)
// CHECK-NEXT:   br i1 %3, label %_llgo_3, label %_llgo_4
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_5
// CHECK-NEXT:   %4 = call ptr @"{{.*}}/runtime/internal/runtime.IfacePtrData"(%"{{.*}}/runtime/internal/runtime.iface" %38)
// CHECK-NEXT:   %5 = extractvalue %"{{.*}}/runtime/internal/runtime.iface" %38, 0
// CHECK-NEXT:   %6 = getelementptr ptr, ptr %5, i64 3
// CHECK-NEXT:   %7 = load ptr, ptr %6, align 8
// CHECK-NEXT:   %8 = insertvalue { ptr, ptr } undef, ptr %7, 0
// CHECK-NEXT:   %9 = insertvalue { ptr, ptr } %8, ptr %4, 1
// CHECK-NEXT:   %10 = extractvalue { ptr, ptr } %9, 1
// CHECK-NEXT:   %11 = extractvalue { ptr, ptr } %9, 0
// CHECK-NEXT:   %12 = call { i64, %"{{.*}}/runtime/internal/runtime.iface" } %11(ptr %10, %"{{.*}}/runtime/internal/runtime.String" %1)
// CHECK-NEXT:   %13 = extractvalue { i64, %"{{.*}}/runtime/internal/runtime.iface" } %12, 0
// CHECK-NEXT:   %14 = extractvalue { i64, %"{{.*}}/runtime/internal/runtime.iface" } %12, 1
// CHECK-NEXT:   %15 = insertvalue { i64, %"{{.*}}/runtime/internal/runtime.iface" } undef, i64 %13, 0
// CHECK-NEXT:   %16 = insertvalue { i64, %"{{.*}}/runtime/internal/runtime.iface" } %15, %"{{.*}}/runtime/internal/runtime.iface" %14, 1
// CHECK-NEXT:   ret { i64, %"{{.*}}/runtime/internal/runtime.iface" } %16
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_5
// CHECK-NEXT:   %17 = call %"{{.*}}/runtime/internal/runtime.Slice" @"{{.*}}/runtime/internal/runtime.StringToBytes"(%"{{.*}}/runtime/internal/runtime.String" %1)
// CHECK-NEXT:   %18 = call ptr @"{{.*}}/runtime/internal/runtime.IfacePtrData"(%"{{.*}}/runtime/internal/runtime.iface" %0)
// CHECK-NEXT:   %19 = extractvalue %"{{.*}}/runtime/internal/runtime.iface" %0, 0
// CHECK-NEXT:   %20 = getelementptr ptr, ptr %19, i64 3
// CHECK-NEXT:   %21 = load ptr, ptr %20, align 8
// CHECK-NEXT:   %22 = insertvalue { ptr, ptr } undef, ptr %21, 0
// CHECK-NEXT:   %23 = insertvalue { ptr, ptr } %22, ptr %18, 1
// CHECK-NEXT:   %24 = extractvalue { ptr, ptr } %23, 1
// CHECK-NEXT:   %25 = extractvalue { ptr, ptr } %23, 0
// CHECK-NEXT:   %26 = call { i64, %"{{.*}}/runtime/internal/runtime.iface" } %25(ptr %24, %"{{.*}}/runtime/internal/runtime.Slice" %17)
// CHECK-NEXT:   %27 = extractvalue { i64, %"{{.*}}/runtime/internal/runtime.iface" } %26, 0
// CHECK-NEXT:   %28 = extractvalue { i64, %"{{.*}}/runtime/internal/runtime.iface" } %26, 1
// CHECK-NEXT:   %29 = insertvalue { i64, %"{{.*}}/runtime/internal/runtime.iface" } undef, i64 %27, 0
// CHECK-NEXT:   %30 = insertvalue { i64, %"{{.*}}/runtime/internal/runtime.iface" } %29, %"{{.*}}/runtime/internal/runtime.iface" %28, 1
// CHECK-NEXT:   ret { i64, %"{{.*}}/runtime/internal/runtime.iface" } %30
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_3:                                          ; preds = %_llgo_0
// CHECK-NEXT:   %31 = extractvalue %"{{.*}}/runtime/internal/runtime.iface" %0, 1
// CHECK-NEXT:   %32 = call ptr @"{{.*}}/runtime/internal/runtime.NewItab"(ptr @"_llgo_iface$Ly4zXiUMEac-hYAMw6b6miJ1JEhGfLyBWyBOhpsRZcU", ptr %2)
// CHECK-NEXT:   %33 = insertvalue %"{{.*}}/runtime/internal/runtime.iface" undef, ptr %32, 0
// CHECK-NEXT:   %34 = insertvalue %"{{.*}}/runtime/internal/runtime.iface" %33, ptr %31, 1
// CHECK-NEXT:   %35 = insertvalue { %"{{.*}}/runtime/internal/runtime.iface", i1 } undef, %"{{.*}}/runtime/internal/runtime.iface" %34, 0
// CHECK-NEXT:   %36 = insertvalue { %"{{.*}}/runtime/internal/runtime.iface", i1 } %35, i1 true, 1
// CHECK-NEXT:   br label %_llgo_5
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_4:                                          ; preds = %_llgo_0
// CHECK-NEXT:   br label %_llgo_5
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_5:                                          ; preds = %_llgo_4, %_llgo_3
// CHECK-NEXT:   %37 = phi { %"{{.*}}/runtime/internal/runtime.iface", i1 } [ %36, %_llgo_3 ], [ zeroinitializer, %_llgo_4 ]
// CHECK-NEXT:   %38 = extractvalue { %"{{.*}}/runtime/internal/runtime.iface", i1 } %37, 0
// CHECK-NEXT:   %39 = extractvalue { %"{{.*}}/runtime/internal/runtime.iface", i1 } %37, 1
// CHECK-NEXT:   br i1 %39, label %_llgo_1, label %_llgo_2
// CHECK-NEXT: }

func ReadAll(r Reader) ([]byte, error) {
	b := make([]byte, 0, 512)
	for {
		n, err := r.Read(b[len(b):cap(b)])
		b = b[:len(b)+n]
		if err != nil {
			if err == EOF {
				err = nil
			}
			return b, err
		}

		if len(b) == cap(b) {
			// Add more capacity (let append pick how much).
			b = append(b, 0)[:len(b)]
		}
	}
}

type stringReader struct {
	s        string
	i        int64 // current reading index
	prevRune int   // index of previous rune; or < 0
}

func (r *stringReader) Len() int {
	if r.i >= int64(len(r.s)) {
		return 0
	}
	return int(int64(len(r.s)) - r.i)
}

func (r *stringReader) Size() int64 { return int64(len(r.s)) }

func (r *stringReader) Read(b []byte) (n int, err error) {
	if r.i >= int64(len(r.s)) {
		return 0, EOF
	}
	r.prevRune = -1
	n = copy(b, r.s[r.i:])
	r.i += int64(n)
	return
}

func (r *stringReader) ReadAt(b []byte, off int64) (n int, err error) {
	if off < 0 {
		return 0, newError("stringsReader.ReadAt: negative offset")
	}
	if off >= int64(len(r.s)) {
		return 0, EOF
	}
	n = copy(b, r.s[off:])
	if n < len(b) {
		err = EOF
	}
	return
}

func (r *stringReader) ReadByte() (byte, error) {
	r.prevRune = -1
	if r.i >= int64(len(r.s)) {
		return 0, EOF
	}
	b := r.s[r.i]
	r.i++
	return b, nil
}

func (r *stringReader) UnreadByte() error {
	if r.i <= 0 {
		return newError("stringsReader.UnreadByte: at beginning of string")
	}
	r.prevRune = -1
	r.i--
	return nil
}

func (r *stringReader) ReadRune() (ch rune, size int, err error) {
	if r.i >= int64(len(r.s)) {
		r.prevRune = -1
		return 0, 0, EOF
	}
	r.prevRune = int(r.i)
	if c := r.s[r.i]; c < utf8.RuneSelf {
		r.i++
		return rune(c), 1, nil
	}
	ch, size = utf8.DecodeRuneInString(r.s[r.i:])
	r.i += int64(size)
	return
}

func (r *stringReader) UnreadRune() error {
	if r.i <= 0 {
		return newError("strings.Reader.UnreadRune: at beginning of string")
	}
	if r.prevRune < 0 {
		return newError("strings.Reader.UnreadRune: previous operation was not ReadRune")
	}
	r.i = int64(r.prevRune)
	r.prevRune = -1
	return nil
}

const (
	SeekStart   = 0 // seek relative to the origin of the file
	SeekCurrent = 1 // seek relative to the current offset
	SeekEnd     = 2 // seek relative to the end
)

func (r *stringReader) Seek(offset int64, whence int) (int64, error) {
	r.prevRune = -1
	var abs int64
	switch whence {
	case SeekStart:
		abs = offset
	case SeekCurrent:
		abs = r.i + offset
	case SeekEnd:
		abs = int64(len(r.s)) + offset
	default:
		return 0, newError("stringsReader.Seek: invalid whence")
	}
	if abs < 0 {
		return 0, newError("stringsReader.Seek: negative position")
	}
	r.i = abs
	return abs, nil
}

func (r *stringReader) WriteTo(w Writer) (n int64, err error) {
	r.prevRune = -1
	if r.i >= int64(len(r.s)) {
		return 0, nil
	}
	s := r.s[r.i:]
	m, err := WriteString(w, s)
	if m > len(s) {
		panic("stringsReader.WriteTo: invalid WriteString count")
	}
	r.i += int64(m)
	n = int64(m)
	if m != len(s) && err == nil {
		err = ErrShortWrite
	}
	return
}

func newError(text string) error {
	return &errorString{text}
}

type errorString struct {
	s string
}

// CHECK-LABEL: define %"{{.*}}/runtime/internal/runtime.String" @"{{.*}}/cl/_testgo/reader.(*errorString).Error"(ptr %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = getelementptr inbounds %"{{.*}}/cl/_testgo/reader.errorString", ptr %0, i32 0, i32 0
// CHECK-NEXT:   %2 = load %"{{.*}}/runtime/internal/runtime.String", ptr %1, align 8
// CHECK-NEXT:   ret %"{{.*}}/runtime/internal/runtime.String" %2
// CHECK-NEXT: }

// CHECK-LABEL: define void @"{{.*}}/cl/_testgo/reader.init"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = load i1, ptr @"{{.*}}/cl/_testgo/reader.init$guard", align 1
// CHECK-NEXT:   br i1 %0, label %_llgo_2, label %_llgo_1
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_0
// CHECK-NEXT:   store i1 true, ptr @"{{.*}}/cl/_testgo/reader.init$guard", align 1
// CHECK-NEXT:   call void @"unicode/utf8.init"()
// CHECK-NEXT:   %1 = call %"{{.*}}/runtime/internal/runtime.iface" @"{{.*}}/cl/_testgo/reader.newError"(%"{{.*}}/runtime/internal/runtime.String" { ptr @28, i64 3 })
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.iface" %1, ptr @"{{.*}}/cl/_testgo/reader.EOF", align 8
// CHECK-NEXT:   %2 = call %"{{.*}}/runtime/internal/runtime.iface" @"{{.*}}/cl/_testgo/reader.newError"(%"{{.*}}/runtime/internal/runtime.String" { ptr @29, i64 11 })
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.iface" %2, ptr @"{{.*}}/cl/_testgo/reader.ErrShortWrite", align 8
// CHECK-NEXT:   br label %_llgo_2
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_1, %_llgo_0
// CHECK-NEXT:   ret void
// CHECK-NEXT: }

func (e *errorString) Error() string {
	return e.s
}

var (
	EOF           = newError("EOF")
	ErrShortWrite = newError("short write")
)

// CHECK-LABEL: define void @"{{.*}}/cl/_testgo/reader.main"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 32)
// CHECK-NEXT:   %1 = getelementptr inbounds %"{{.*}}/cl/_testgo/reader.stringReader", ptr %0, i32 0, i32 0
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @30, i64 11 }, ptr %1, align 8
// CHECK-NEXT:   %2 = call ptr @"{{.*}}/runtime/internal/runtime.NewItab"(ptr @"_llgo_iface$uycIKA3bbxRhudEjW1hHKWKdLqHQsCVy8NdW1bkQmNw", ptr @"*_llgo_{{.*}}/cl/_testgo/reader.stringReader")
// CHECK-NEXT:   %3 = insertvalue %"{{.*}}/runtime/internal/runtime.iface" undef, ptr %2, 0
// CHECK-NEXT:   %4 = insertvalue %"{{.*}}/runtime/internal/runtime.iface" %3, ptr %0, 1
// CHECK-NEXT:   %5 = call { %"{{.*}}/runtime/internal/runtime.Slice", %"{{.*}}/runtime/internal/runtime.iface" } @"{{.*}}/cl/_testgo/reader.ReadAll"(%"{{.*}}/runtime/internal/runtime.iface" %4)
// CHECK-NEXT:   %6 = extractvalue { %"{{.*}}/runtime/internal/runtime.Slice", %"{{.*}}/runtime/internal/runtime.iface" } %5, 0
// CHECK-NEXT:   %7 = extractvalue { %"{{.*}}/runtime/internal/runtime.Slice", %"{{.*}}/runtime/internal/runtime.iface" } %5, 1
// CHECK-NEXT:   %8 = call %"{{.*}}/runtime/internal/runtime.String" @"{{.*}}/runtime/internal/runtime.StringFromBytes"(%"{{.*}}/runtime/internal/runtime.Slice" %6)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintString"(%"{{.*}}/runtime/internal/runtime.String" %8)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintIface"(%"{{.*}}/runtime/internal/runtime.iface" %7)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   ret void
// CHECK-NEXT: }

// CHECK-LABEL: define %"{{.*}}/runtime/internal/runtime.iface" @"{{.*}}/cl/_testgo/reader.newError"(%"{{.*}}/runtime/internal/runtime.String" %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 16)
// CHECK-NEXT:   %2 = getelementptr inbounds %"{{.*}}/cl/_testgo/reader.errorString", ptr %1, i32 0, i32 0
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" %0, ptr %2, align 8
// CHECK-NEXT:   %3 = call ptr @"{{.*}}/runtime/internal/runtime.NewItab"(ptr @"_llgo_iface$Fh8eUJ-Gw4e6TYuajcFIOSCuqSPKAt5nS4ow7xeGXEU", ptr @"*_llgo_{{.*}}/cl/_testgo/reader.errorString")
// CHECK-NEXT:   %4 = insertvalue %"{{.*}}/runtime/internal/runtime.iface" undef, ptr %3, 0
// CHECK-NEXT:   %5 = insertvalue %"{{.*}}/runtime/internal/runtime.iface" %4, ptr %1, 1
// CHECK-NEXT:   ret %"{{.*}}/runtime/internal/runtime.iface" %5
// CHECK-NEXT: }

// CHECK-LABEL: define %"{{.*}}/runtime/internal/runtime.iface" @"{{.*}}/cl/_testgo/reader.nopCloser.Close"(%"{{.*}}/cl/_testgo/reader.nopCloser" %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   ret %"{{.*}}/runtime/internal/runtime.iface" zeroinitializer
// CHECK-NEXT: }

func main() {
	r := &stringReader{s: "hello world"}
	data, err := ReadAll(r)
	println(string(data), err)
}

// CHECK-LABEL: define { i64, %"{{.*}}/runtime/internal/runtime.iface" } @"{{.*}}/cl/_testgo/reader.nopCloser.Read"(%"{{.*}}/cl/_testgo/reader.nopCloser" %0, %"{{.*}}/runtime/internal/runtime.Slice" %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = alloca %"{{.*}}/cl/_testgo/reader.nopCloser", align 8
// CHECK-NEXT:   call void @llvm.memset(ptr %2, i8 0, i64 16, i1 false)
// CHECK-NEXT:   store %"{{.*}}/cl/_testgo/reader.nopCloser" %0, ptr %2, align 8
// CHECK-NEXT:   %3 = getelementptr inbounds %"{{.*}}/cl/_testgo/reader.nopCloser", ptr %2, i32 0, i32 0
// CHECK-NEXT:   %4 = load %"{{.*}}/runtime/internal/runtime.iface", ptr %3, align 8
// CHECK-NEXT:   %5 = call ptr @"{{.*}}/runtime/internal/runtime.IfacePtrData"(%"{{.*}}/runtime/internal/runtime.iface" %4)
// CHECK-NEXT:   %6 = extractvalue %"{{.*}}/runtime/internal/runtime.iface" %4, 0
// CHECK-NEXT:   %7 = getelementptr ptr, ptr %6, i64 3
// CHECK-NEXT:   %8 = load ptr, ptr %7, align 8
// CHECK-NEXT:   %9 = insertvalue { ptr, ptr } undef, ptr %8, 0
// CHECK-NEXT:   %10 = insertvalue { ptr, ptr } %9, ptr %5, 1
// CHECK-NEXT:   %11 = extractvalue { ptr, ptr } %10, 1
// CHECK-NEXT:   %12 = extractvalue { ptr, ptr } %10, 0
// CHECK-NEXT:   %13 = call { i64, %"{{.*}}/runtime/internal/runtime.iface" } %12(ptr %11, %"{{.*}}/runtime/internal/runtime.Slice" %1)
// CHECK-NEXT:   %14 = extractvalue { i64, %"{{.*}}/runtime/internal/runtime.iface" } %13, 0
// CHECK-NEXT:   %15 = extractvalue { i64, %"{{.*}}/runtime/internal/runtime.iface" } %13, 1
// CHECK-NEXT:   %16 = insertvalue { i64, %"{{.*}}/runtime/internal/runtime.iface" } undef, i64 %14, 0
// CHECK-NEXT:   %17 = insertvalue { i64, %"{{.*}}/runtime/internal/runtime.iface" } %16, %"{{.*}}/runtime/internal/runtime.iface" %15, 1
// CHECK-NEXT:   ret { i64, %"{{.*}}/runtime/internal/runtime.iface" } %17
// CHECK-NEXT: }

// CHECK-LABEL: define %"{{.*}}/runtime/internal/runtime.iface" @"{{.*}}/cl/_testgo/reader.(*nopCloser).Close"(ptr %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = icmp eq ptr %0, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PanicWrapNilPointer"(i1 %1, %"{{.*}}/runtime/internal/runtime.String" { ptr @53, i64 50 }, %"{{.*}}/runtime/internal/runtime.String" { ptr @17, i64 5 })
// CHECK-NEXT:   %2 = load %"{{.*}}/cl/_testgo/reader.nopCloser", ptr %0, align 8
// CHECK-NEXT:   %3 = call %"{{.*}}/runtime/internal/runtime.iface" @"{{.*}}/cl/_testgo/reader.nopCloser.Close"(%"{{.*}}/cl/_testgo/reader.nopCloser" %2)
// CHECK-NEXT:   ret %"{{.*}}/runtime/internal/runtime.iface" %3
// CHECK-NEXT: }

// CHECK-LABEL: define { i64, %"{{.*}}/runtime/internal/runtime.iface" } @"{{.*}}/cl/_testgo/reader.(*nopCloser).Read"(ptr %0, %"{{.*}}/runtime/internal/runtime.Slice" %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = getelementptr inbounds %"{{.*}}/cl/_testgo/reader.nopCloser", ptr %0, i32 0, i32 0
// CHECK-NEXT:   %3 = load %"{{.*}}/runtime/internal/runtime.iface", ptr %2, align 8
// CHECK-NEXT:   %4 = call ptr @"{{.*}}/runtime/internal/runtime.IfacePtrData"(%"{{.*}}/runtime/internal/runtime.iface" %3)
// CHECK-NEXT:   %5 = extractvalue %"{{.*}}/runtime/internal/runtime.iface" %3, 0
// CHECK-NEXT:   %6 = getelementptr ptr, ptr %5, i64 3
// CHECK-NEXT:   %7 = load ptr, ptr %6, align 8
// CHECK-NEXT:   %8 = insertvalue { ptr, ptr } undef, ptr %7, 0
// CHECK-NEXT:   %9 = insertvalue { ptr, ptr } %8, ptr %4, 1
// CHECK-NEXT:   %10 = extractvalue { ptr, ptr } %9, 1
// CHECK-NEXT:   %11 = extractvalue { ptr, ptr } %9, 0
// CHECK-NEXT:   %12 = call { i64, %"{{.*}}/runtime/internal/runtime.iface" } %11(ptr %10, %"{{.*}}/runtime/internal/runtime.Slice" %1)
// CHECK-NEXT:   %13 = extractvalue { i64, %"{{.*}}/runtime/internal/runtime.iface" } %12, 0
// CHECK-NEXT:   %14 = extractvalue { i64, %"{{.*}}/runtime/internal/runtime.iface" } %12, 1
// CHECK-NEXT:   %15 = insertvalue { i64, %"{{.*}}/runtime/internal/runtime.iface" } undef, i64 %13, 0
// CHECK-NEXT:   %16 = insertvalue { i64, %"{{.*}}/runtime/internal/runtime.iface" } %15, %"{{.*}}/runtime/internal/runtime.iface" %14, 1
// CHECK-NEXT:   ret { i64, %"{{.*}}/runtime/internal/runtime.iface" } %16
// CHECK-NEXT: }

// CHECK-LABEL: define %"{{.*}}/runtime/internal/runtime.iface" @"{{.*}}/cl/_testgo/reader.nopCloserWriterTo.Close"(%"{{.*}}/cl/_testgo/reader.nopCloserWriterTo" %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   ret %"{{.*}}/runtime/internal/runtime.iface" zeroinitializer
// CHECK-NEXT: }

// CHECK-LABEL: define { i64, %"{{.*}}/runtime/internal/runtime.iface" } @"{{.*}}/cl/_testgo/reader.nopCloserWriterTo.Read"(%"{{.*}}/cl/_testgo/reader.nopCloserWriterTo" %0, %"{{.*}}/runtime/internal/runtime.Slice" %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = alloca %"{{.*}}/cl/_testgo/reader.nopCloserWriterTo", align 8
// CHECK-NEXT:   call void @llvm.memset(ptr %2, i8 0, i64 16, i1 false)
// CHECK-NEXT:   store %"{{.*}}/cl/_testgo/reader.nopCloserWriterTo" %0, ptr %2, align 8
// CHECK-NEXT:   %3 = getelementptr inbounds %"{{.*}}/cl/_testgo/reader.nopCloserWriterTo", ptr %2, i32 0, i32 0
// CHECK-NEXT:   %4 = load %"{{.*}}/runtime/internal/runtime.iface", ptr %3, align 8
// CHECK-NEXT:   %5 = call ptr @"{{.*}}/runtime/internal/runtime.IfacePtrData"(%"{{.*}}/runtime/internal/runtime.iface" %4)
// CHECK-NEXT:   %6 = extractvalue %"{{.*}}/runtime/internal/runtime.iface" %4, 0
// CHECK-NEXT:   %7 = getelementptr ptr, ptr %6, i64 3
// CHECK-NEXT:   %8 = load ptr, ptr %7, align 8
// CHECK-NEXT:   %9 = insertvalue { ptr, ptr } undef, ptr %8, 0
// CHECK-NEXT:   %10 = insertvalue { ptr, ptr } %9, ptr %5, 1
// CHECK-NEXT:   %11 = extractvalue { ptr, ptr } %10, 1
// CHECK-NEXT:   %12 = extractvalue { ptr, ptr } %10, 0
// CHECK-NEXT:   %13 = call { i64, %"{{.*}}/runtime/internal/runtime.iface" } %12(ptr %11, %"{{.*}}/runtime/internal/runtime.Slice" %1)
// CHECK-NEXT:   %14 = extractvalue { i64, %"{{.*}}/runtime/internal/runtime.iface" } %13, 0
// CHECK-NEXT:   %15 = extractvalue { i64, %"{{.*}}/runtime/internal/runtime.iface" } %13, 1
// CHECK-NEXT:   %16 = insertvalue { i64, %"{{.*}}/runtime/internal/runtime.iface" } undef, i64 %14, 0
// CHECK-NEXT:   %17 = insertvalue { i64, %"{{.*}}/runtime/internal/runtime.iface" } %16, %"{{.*}}/runtime/internal/runtime.iface" %15, 1
// CHECK-NEXT:   ret { i64, %"{{.*}}/runtime/internal/runtime.iface" } %17
// CHECK-NEXT: }

// CHECK-LABEL: define { i64, %"{{.*}}/runtime/internal/runtime.iface" } @"{{.*}}/cl/_testgo/reader.nopCloserWriterTo.WriteTo"(%"{{.*}}/cl/_testgo/reader.nopCloserWriterTo" %0, %"{{.*}}/runtime/internal/runtime.iface" %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = alloca %"{{.*}}/cl/_testgo/reader.nopCloserWriterTo", align 8
// CHECK-NEXT:   call void @llvm.memset(ptr %2, i8 0, i64 16, i1 false)
// CHECK-NEXT:   store %"{{.*}}/cl/_testgo/reader.nopCloserWriterTo" %0, ptr %2, align 8
// CHECK-NEXT:   %3 = getelementptr inbounds %"{{.*}}/cl/_testgo/reader.nopCloserWriterTo", ptr %2, i32 0, i32 0
// CHECK-NEXT:   %4 = load %"{{.*}}/runtime/internal/runtime.iface", ptr %3, align 8
// CHECK-NEXT:   %5 = call ptr @"{{.*}}/runtime/internal/runtime.IfaceType"(%"{{.*}}/runtime/internal/runtime.iface" %4)
// CHECK-NEXT:   %6 = call i1 @"{{.*}}/runtime/internal/runtime.Implements"(ptr @"_llgo_{{.*}}/cl/_testgo/reader.WriterTo", ptr %5)
// CHECK-NEXT:   br i1 %6, label %_llgo_1, label %_llgo_2
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_0
// CHECK-NEXT:   %7 = extractvalue %"{{.*}}/runtime/internal/runtime.iface" %4, 1
// CHECK-NEXT:   %8 = call ptr @"{{.*}}/runtime/internal/runtime.NewItab"(ptr @"_llgo_iface$p5Bo_emI1h8acs1rFbUxZTrpeDbIQ34gFcsbwK9YIgs", ptr %5)
// CHECK-NEXT:   %9 = insertvalue %"{{.*}}/runtime/internal/runtime.iface" undef, ptr %8, 0
// CHECK-NEXT:   %10 = insertvalue %"{{.*}}/runtime/internal/runtime.iface" %9, ptr %7, 1
// CHECK-NEXT:   %11 = call ptr @"{{.*}}/runtime/internal/runtime.IfacePtrData"(%"{{.*}}/runtime/internal/runtime.iface" %10)
// CHECK-NEXT:   %12 = extractvalue %"{{.*}}/runtime/internal/runtime.iface" %10, 0
// CHECK-NEXT:   %13 = getelementptr ptr, ptr %12, i64 3
// CHECK-NEXT:   %14 = load ptr, ptr %13, align 8
// CHECK-NEXT:   %15 = insertvalue { ptr, ptr } undef, ptr %14, 0
// CHECK-NEXT:   %16 = insertvalue { ptr, ptr } %15, ptr %11, 1
// CHECK-NEXT:   %17 = extractvalue { ptr, ptr } %16, 1
// CHECK-NEXT:   %18 = extractvalue { ptr, ptr } %16, 0
// CHECK-NEXT:   %19 = call { i64, %"{{.*}}/runtime/internal/runtime.iface" } %18(ptr %17, %"{{.*}}/runtime/internal/runtime.iface" %1)
// CHECK-NEXT:   %20 = extractvalue { i64, %"{{.*}}/runtime/internal/runtime.iface" } %19, 0
// CHECK-NEXT:   %21 = extractvalue { i64, %"{{.*}}/runtime/internal/runtime.iface" } %19, 1
// CHECK-NEXT:   %22 = insertvalue { i64, %"{{.*}}/runtime/internal/runtime.iface" } undef, i64 %20, 0
// CHECK-NEXT:   %23 = insertvalue { i64, %"{{.*}}/runtime/internal/runtime.iface" } %22, %"{{.*}}/runtime/internal/runtime.iface" %21, 1
// CHECK-NEXT:   ret { i64, %"{{.*}}/runtime/internal/runtime.iface" } %23
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_0
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PanicTypeAssert"(ptr %5, %"{{.*}}/runtime/internal/runtime.String" { ptr @54, i64 49 }, %"{{.*}}/runtime/internal/runtime.String" { ptr @2, i64 7 })
// CHECK-NEXT:   unreachable
// CHECK-NEXT: }

// CHECK-LABEL: define %"{{.*}}/runtime/internal/runtime.iface" @"{{.*}}/cl/_testgo/reader.(*nopCloserWriterTo).Close"(ptr %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = icmp eq ptr %0, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PanicWrapNilPointer"(i1 %1, %"{{.*}}/runtime/internal/runtime.String" { ptr @55, i64 58 }, %"{{.*}}/runtime/internal/runtime.String" { ptr @17, i64 5 })
// CHECK-NEXT:   %2 = load %"{{.*}}/cl/_testgo/reader.nopCloserWriterTo", ptr %0, align 8
// CHECK-NEXT:   %3 = call %"{{.*}}/runtime/internal/runtime.iface" @"{{.*}}/cl/_testgo/reader.nopCloserWriterTo.Close"(%"{{.*}}/cl/_testgo/reader.nopCloserWriterTo" %2)
// CHECK-NEXT:   ret %"{{.*}}/runtime/internal/runtime.iface" %3
// CHECK-NEXT: }

// CHECK-LABEL: define { i64, %"{{.*}}/runtime/internal/runtime.iface" } @"{{.*}}/cl/_testgo/reader.(*nopCloserWriterTo).Read"(ptr %0, %"{{.*}}/runtime/internal/runtime.Slice" %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = getelementptr inbounds %"{{.*}}/cl/_testgo/reader.nopCloserWriterTo", ptr %0, i32 0, i32 0
// CHECK-NEXT:   %3 = load %"{{.*}}/runtime/internal/runtime.iface", ptr %2, align 8
// CHECK-NEXT:   %4 = call ptr @"{{.*}}/runtime/internal/runtime.IfacePtrData"(%"{{.*}}/runtime/internal/runtime.iface" %3)
// CHECK-NEXT:   %5 = extractvalue %"{{.*}}/runtime/internal/runtime.iface" %3, 0
// CHECK-NEXT:   %6 = getelementptr ptr, ptr %5, i64 3
// CHECK-NEXT:   %7 = load ptr, ptr %6, align 8
// CHECK-NEXT:   %8 = insertvalue { ptr, ptr } undef, ptr %7, 0
// CHECK-NEXT:   %9 = insertvalue { ptr, ptr } %8, ptr %4, 1
// CHECK-NEXT:   %10 = extractvalue { ptr, ptr } %9, 1
// CHECK-NEXT:   %11 = extractvalue { ptr, ptr } %9, 0
// CHECK-NEXT:   %12 = call { i64, %"{{.*}}/runtime/internal/runtime.iface" } %11(ptr %10, %"{{.*}}/runtime/internal/runtime.Slice" %1)
// CHECK-NEXT:   %13 = extractvalue { i64, %"{{.*}}/runtime/internal/runtime.iface" } %12, 0
// CHECK-NEXT:   %14 = extractvalue { i64, %"{{.*}}/runtime/internal/runtime.iface" } %12, 1
// CHECK-NEXT:   %15 = insertvalue { i64, %"{{.*}}/runtime/internal/runtime.iface" } undef, i64 %13, 0
// CHECK-NEXT:   %16 = insertvalue { i64, %"{{.*}}/runtime/internal/runtime.iface" } %15, %"{{.*}}/runtime/internal/runtime.iface" %14, 1
// CHECK-NEXT:   ret { i64, %"{{.*}}/runtime/internal/runtime.iface" } %16
// CHECK-NEXT: }

// CHECK-LABEL: define { i64, %"{{.*}}/runtime/internal/runtime.iface" } @"{{.*}}/cl/_testgo/reader.(*nopCloserWriterTo).WriteTo"(ptr %0, %"{{.*}}/runtime/internal/runtime.iface" %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = icmp eq ptr %0, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PanicWrapNilPointer"(i1 %2, %"{{.*}}/runtime/internal/runtime.String" { ptr @55, i64 58 }, %"{{.*}}/runtime/internal/runtime.String" { ptr @2, i64 7 })
// CHECK-NEXT:   %3 = load %"{{.*}}/cl/_testgo/reader.nopCloserWriterTo", ptr %0, align 8
// CHECK-NEXT:   %4 = call { i64, %"{{.*}}/runtime/internal/runtime.iface" } @"{{.*}}/cl/_testgo/reader.nopCloserWriterTo.WriteTo"(%"{{.*}}/cl/_testgo/reader.nopCloserWriterTo" %3, %"{{.*}}/runtime/internal/runtime.iface" %1)
// CHECK-NEXT:   %5 = extractvalue { i64, %"{{.*}}/runtime/internal/runtime.iface" } %4, 0
// CHECK-NEXT:   %6 = extractvalue { i64, %"{{.*}}/runtime/internal/runtime.iface" } %4, 1
// CHECK-NEXT:   %7 = insertvalue { i64, %"{{.*}}/runtime/internal/runtime.iface" } undef, i64 %5, 0
// CHECK-NEXT:   %8 = insertvalue { i64, %"{{.*}}/runtime/internal/runtime.iface" } %7, %"{{.*}}/runtime/internal/runtime.iface" %6, 1
// CHECK-NEXT:   ret { i64, %"{{.*}}/runtime/internal/runtime.iface" } %8
// CHECK-NEXT: }

// CHECK-LABEL: define i64 @"{{.*}}/cl/_testgo/reader.(*stringReader).Len"(ptr %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = getelementptr inbounds %"{{.*}}/cl/_testgo/reader.stringReader", ptr %0, i32 0, i32 1
// CHECK-NEXT:   %2 = load i64, ptr %1, align 8
// CHECK-NEXT:   %3 = getelementptr inbounds %"{{.*}}/cl/_testgo/reader.stringReader", ptr %0, i32 0, i32 0
// CHECK-NEXT:   %4 = load %"{{.*}}/runtime/internal/runtime.String", ptr %3, align 8
// CHECK-NEXT:   %5 = extractvalue %"{{.*}}/runtime/internal/runtime.String" %4, 1
// CHECK-NEXT:   %6 = icmp sge i64 %2, %5
// CHECK-NEXT:   br i1 %6, label %_llgo_1, label %_llgo_2
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_0
// CHECK-NEXT:   ret i64 0
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_0
// CHECK-NEXT:   %7 = getelementptr inbounds %"{{.*}}/cl/_testgo/reader.stringReader", ptr %0, i32 0, i32 0
// CHECK-NEXT:   %8 = load %"{{.*}}/runtime/internal/runtime.String", ptr %7, align 8
// CHECK-NEXT:   %9 = extractvalue %"{{.*}}/runtime/internal/runtime.String" %8, 1
// CHECK-NEXT:   %10 = getelementptr inbounds %"{{.*}}/cl/_testgo/reader.stringReader", ptr %0, i32 0, i32 1
// CHECK-NEXT:   %11 = load i64, ptr %10, align 8
// CHECK-NEXT:   %12 = sub i64 %9, %11
// CHECK-NEXT:   ret i64 %12
// CHECK-NEXT: }

// CHECK-LABEL: define { i64, %"{{.*}}/runtime/internal/runtime.iface" } @"{{.*}}/cl/_testgo/reader.(*stringReader).Read"(ptr %0, %"{{.*}}/runtime/internal/runtime.Slice" %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = getelementptr inbounds %"{{.*}}/cl/_testgo/reader.stringReader", ptr %0, i32 0, i32 1
// CHECK-NEXT:   %3 = load i64, ptr %2, align 8
// CHECK-NEXT:   %4 = getelementptr inbounds %"{{.*}}/cl/_testgo/reader.stringReader", ptr %0, i32 0, i32 0
// CHECK-NEXT:   %5 = load %"{{.*}}/runtime/internal/runtime.String", ptr %4, align 8
// CHECK-NEXT:   %6 = extractvalue %"{{.*}}/runtime/internal/runtime.String" %5, 1
// CHECK-NEXT:   %7 = icmp sge i64 %3, %6
// CHECK-NEXT:   br i1 %7, label %_llgo_1, label %_llgo_2
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_0
// CHECK-NEXT:   %8 = load %"{{.*}}/runtime/internal/runtime.iface", ptr @"{{.*}}/cl/_testgo/reader.EOF", align 8
// CHECK-NEXT:   %9 = insertvalue { i64, %"{{.*}}/runtime/internal/runtime.iface" } { i64 0, %"{{.*}}/runtime/internal/runtime.iface" undef }, %"{{.*}}/runtime/internal/runtime.iface" %8, 1
// CHECK-NEXT:   ret { i64, %"{{.*}}/runtime/internal/runtime.iface" } %9
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_0
// CHECK-NEXT:   %10 = getelementptr inbounds %"{{.*}}/cl/_testgo/reader.stringReader", ptr %0, i32 0, i32 2
// CHECK-NEXT:   store i64 -1, ptr %10, align 8
// CHECK-NEXT:   %11 = getelementptr inbounds %"{{.*}}/cl/_testgo/reader.stringReader", ptr %0, i32 0, i32 0
// CHECK-NEXT:   %12 = load %"{{.*}}/runtime/internal/runtime.String", ptr %11, align 8
// CHECK-NEXT:   %13 = getelementptr inbounds %"{{.*}}/cl/_testgo/reader.stringReader", ptr %0, i32 0, i32 1
// CHECK-NEXT:   %14 = load i64, ptr %13, align 8
// CHECK-NEXT:   %15 = extractvalue %"{{.*}}/runtime/internal/runtime.String" %12, 1
// CHECK-NEXT:   %16 = call %"{{.*}}/runtime/internal/runtime.String" @"{{.*}}/runtime/internal/runtime.StringSlice2"(%"{{.*}}/runtime/internal/runtime.String" %12, i64 %14, i64 %15, i1 true, i1 true)
// CHECK-NEXT:   %17 = extractvalue %"{{.*}}/runtime/internal/runtime.String" %16, 0
// CHECK-NEXT:   %18 = extractvalue %"{{.*}}/runtime/internal/runtime.String" %16, 1
// CHECK-NEXT:   %19 = call i64 @"{{.*}}/runtime/internal/runtime.SliceCopy"(%"{{.*}}/runtime/internal/runtime.Slice" %1, ptr %17, i64 %18, i64 1)
// CHECK-NEXT:   %20 = getelementptr inbounds %"{{.*}}/cl/_testgo/reader.stringReader", ptr %0, i32 0, i32 1
// CHECK-NEXT:   %21 = load i64, ptr %20, align 8
// CHECK-NEXT:   %22 = add i64 %21, %19
// CHECK-NEXT:   %23 = getelementptr inbounds %"{{.*}}/cl/_testgo/reader.stringReader", ptr %0, i32 0, i32 1
// CHECK-NEXT:   store i64 %22, ptr %23, align 8
// CHECK-NEXT:   %24 = insertvalue { i64, %"{{.*}}/runtime/internal/runtime.iface" } undef, i64 %19, 0
// CHECK-NEXT:   %25 = insertvalue { i64, %"{{.*}}/runtime/internal/runtime.iface" } %24, %"{{.*}}/runtime/internal/runtime.iface" zeroinitializer, 1
// CHECK-NEXT:   ret { i64, %"{{.*}}/runtime/internal/runtime.iface" } %25
// CHECK-NEXT: }

// CHECK-LABEL: define { i64, %"{{.*}}/runtime/internal/runtime.iface" } @"{{.*}}/cl/_testgo/reader.(*stringReader).ReadAt"(ptr %0, %"{{.*}}/runtime/internal/runtime.Slice" %1, i64 %2){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %3 = icmp slt i64 %2, 0
// CHECK-NEXT:   br i1 %3, label %_llgo_1, label %_llgo_2
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_0
// CHECK-NEXT:   %4 = call %"{{.*}}/runtime/internal/runtime.iface" @"{{.*}}/cl/_testgo/reader.newError"(%"{{.*}}/runtime/internal/runtime.String" { ptr @56, i64 37 })
// CHECK-NEXT:   %5 = insertvalue { i64, %"{{.*}}/runtime/internal/runtime.iface" } { i64 0, %"{{.*}}/runtime/internal/runtime.iface" undef }, %"{{.*}}/runtime/internal/runtime.iface" %4, 1
// CHECK-NEXT:   ret { i64, %"{{.*}}/runtime/internal/runtime.iface" } %5
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_0
// CHECK-NEXT:   %6 = getelementptr inbounds %"{{.*}}/cl/_testgo/reader.stringReader", ptr %0, i32 0, i32 0
// CHECK-NEXT:   %7 = load %"{{.*}}/runtime/internal/runtime.String", ptr %6, align 8
// CHECK-NEXT:   %8 = extractvalue %"{{.*}}/runtime/internal/runtime.String" %7, 1
// CHECK-NEXT:   %9 = icmp sge i64 %2, %8
// CHECK-NEXT:   br i1 %9, label %_llgo_3, label %_llgo_4
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_3:                                          ; preds = %_llgo_2
// CHECK-NEXT:   %10 = load %"{{.*}}/runtime/internal/runtime.iface", ptr @"{{.*}}/cl/_testgo/reader.EOF", align 8
// CHECK-NEXT:   %11 = insertvalue { i64, %"{{.*}}/runtime/internal/runtime.iface" } { i64 0, %"{{.*}}/runtime/internal/runtime.iface" undef }, %"{{.*}}/runtime/internal/runtime.iface" %10, 1
// CHECK-NEXT:   ret { i64, %"{{.*}}/runtime/internal/runtime.iface" } %11
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_4:                                          ; preds = %_llgo_2
// CHECK-NEXT:   %12 = getelementptr inbounds %"{{.*}}/cl/_testgo/reader.stringReader", ptr %0, i32 0, i32 0
// CHECK-NEXT:   %13 = load %"{{.*}}/runtime/internal/runtime.String", ptr %12, align 8
// CHECK-NEXT:   %14 = extractvalue %"{{.*}}/runtime/internal/runtime.String" %13, 1
// CHECK-NEXT:   %15 = call %"{{.*}}/runtime/internal/runtime.String" @"{{.*}}/runtime/internal/runtime.StringSlice2"(%"{{.*}}/runtime/internal/runtime.String" %13, i64 %2, i64 %14, i1 true, i1 true)
// CHECK-NEXT:   %16 = extractvalue %"{{.*}}/runtime/internal/runtime.String" %15, 0
// CHECK-NEXT:   %17 = extractvalue %"{{.*}}/runtime/internal/runtime.String" %15, 1
// CHECK-NEXT:   %18 = call i64 @"{{.*}}/runtime/internal/runtime.SliceCopy"(%"{{.*}}/runtime/internal/runtime.Slice" %1, ptr %16, i64 %17, i64 1)
// CHECK-NEXT:   %19 = extractvalue %"{{.*}}/runtime/internal/runtime.Slice" %1, 1
// CHECK-NEXT:   %20 = icmp slt i64 %18, %19
// CHECK-NEXT:   br i1 %20, label %_llgo_5, label %_llgo_6
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_5:                                          ; preds = %_llgo_4
// CHECK-NEXT:   %21 = load %"{{.*}}/runtime/internal/runtime.iface", ptr @"{{.*}}/cl/_testgo/reader.EOF", align 8
// CHECK-NEXT:   br label %_llgo_6
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_6:                                          ; preds = %_llgo_5, %_llgo_4
// CHECK-NEXT:   %22 = phi %"{{.*}}/runtime/internal/runtime.iface" [ zeroinitializer, %_llgo_4 ], [ %21, %_llgo_5 ]
// CHECK-NEXT:   %23 = insertvalue { i64, %"{{.*}}/runtime/internal/runtime.iface" } undef, i64 %18, 0
// CHECK-NEXT:   %24 = insertvalue { i64, %"{{.*}}/runtime/internal/runtime.iface" } %23, %"{{.*}}/runtime/internal/runtime.iface" %22, 1
// CHECK-NEXT:   ret { i64, %"{{.*}}/runtime/internal/runtime.iface" } %24
// CHECK-NEXT: }

// CHECK-LABEL: define { i8, %"{{.*}}/runtime/internal/runtime.iface" } @"{{.*}}/cl/_testgo/reader.(*stringReader).ReadByte"(ptr %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = getelementptr inbounds %"{{.*}}/cl/_testgo/reader.stringReader", ptr %0, i32 0, i32 2
// CHECK-NEXT:   store i64 -1, ptr %1, align 8
// CHECK-NEXT:   %2 = getelementptr inbounds %"{{.*}}/cl/_testgo/reader.stringReader", ptr %0, i32 0, i32 1
// CHECK-NEXT:   %3 = load i64, ptr %2, align 8
// CHECK-NEXT:   %4 = getelementptr inbounds %"{{.*}}/cl/_testgo/reader.stringReader", ptr %0, i32 0, i32 0
// CHECK-NEXT:   %5 = load %"{{.*}}/runtime/internal/runtime.String", ptr %4, align 8
// CHECK-NEXT:   %6 = extractvalue %"{{.*}}/runtime/internal/runtime.String" %5, 1
// CHECK-NEXT:   %7 = icmp sge i64 %3, %6
// CHECK-NEXT:   br i1 %7, label %_llgo_1, label %_llgo_2
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_0
// CHECK-NEXT:   %8 = load %"{{.*}}/runtime/internal/runtime.iface", ptr @"{{.*}}/cl/_testgo/reader.EOF", align 8
// CHECK-NEXT:   %9 = insertvalue { i8, %"{{.*}}/runtime/internal/runtime.iface" } { i8 0, %"{{.*}}/runtime/internal/runtime.iface" undef }, %"{{.*}}/runtime/internal/runtime.iface" %8, 1
// CHECK-NEXT:   ret { i8, %"{{.*}}/runtime/internal/runtime.iface" } %9
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_0
// CHECK-NEXT:   %10 = getelementptr inbounds %"{{.*}}/cl/_testgo/reader.stringReader", ptr %0, i32 0, i32 1
// CHECK-NEXT:   %11 = load i64, ptr %10, align 8
// CHECK-NEXT:   %12 = getelementptr inbounds %"{{.*}}/cl/_testgo/reader.stringReader", ptr %0, i32 0, i32 0
// CHECK-NEXT:   %13 = load %"{{.*}}/runtime/internal/runtime.String", ptr %12, align 8
// CHECK-NEXT:   %14 = extractvalue %"{{.*}}/runtime/internal/runtime.String" %13, 0
// CHECK-NEXT:   %15 = extractvalue %"{{.*}}/runtime/internal/runtime.String" %13, 1
// CHECK-NEXT:   %16 = icmp slt i64 %11, 0
// CHECK-NEXT:   %17 = icmp uge i64 %11, %15
// CHECK-NEXT:   %18 = or i1 %17, %16
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.CheckIndexRange"(i1 %18, i64 %11, i1 true, i64 %15)
// CHECK-NEXT:   %19 = getelementptr inbounds i8, ptr %14, i64 %11
// CHECK-NEXT:   %20 = load i8, ptr %19, align 1
// CHECK-NEXT:   %21 = getelementptr inbounds %"{{.*}}/cl/_testgo/reader.stringReader", ptr %0, i32 0, i32 1
// CHECK-NEXT:   %22 = load i64, ptr %21, align 8
// CHECK-NEXT:   %23 = add i64 %22, 1
// CHECK-NEXT:   %24 = getelementptr inbounds %"{{.*}}/cl/_testgo/reader.stringReader", ptr %0, i32 0, i32 1
// CHECK-NEXT:   store i64 %23, ptr %24, align 8
// CHECK-NEXT:   %25 = insertvalue { i8, %"{{.*}}/runtime/internal/runtime.iface" } undef, i8 %20, 0
// CHECK-NEXT:   %26 = insertvalue { i8, %"{{.*}}/runtime/internal/runtime.iface" } %25, %"{{.*}}/runtime/internal/runtime.iface" zeroinitializer, 1
// CHECK-NEXT:   ret { i8, %"{{.*}}/runtime/internal/runtime.iface" } %26
// CHECK-NEXT: }

// CHECK-LABEL: define { i32, i64, %"{{.*}}/runtime/internal/runtime.iface" } @"{{.*}}/cl/_testgo/reader.(*stringReader).ReadRune"(ptr %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = getelementptr inbounds %"{{.*}}/cl/_testgo/reader.stringReader", ptr %0, i32 0, i32 1
// CHECK-NEXT:   %2 = load i64, ptr %1, align 8
// CHECK-NEXT:   %3 = getelementptr inbounds %"{{.*}}/cl/_testgo/reader.stringReader", ptr %0, i32 0, i32 0
// CHECK-NEXT:   %4 = load %"{{.*}}/runtime/internal/runtime.String", ptr %3, align 8
// CHECK-NEXT:   %5 = extractvalue %"{{.*}}/runtime/internal/runtime.String" %4, 1
// CHECK-NEXT:   %6 = icmp sge i64 %2, %5
// CHECK-NEXT:   br i1 %6, label %_llgo_1, label %_llgo_2
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_0
// CHECK-NEXT:   %7 = getelementptr inbounds %"{{.*}}/cl/_testgo/reader.stringReader", ptr %0, i32 0, i32 2
// CHECK-NEXT:   store i64 -1, ptr %7, align 8
// CHECK-NEXT:   %8 = load %"{{.*}}/runtime/internal/runtime.iface", ptr @"{{.*}}/cl/_testgo/reader.EOF", align 8
// CHECK-NEXT:   %9 = insertvalue { i32, i64, %"{{.*}}/runtime/internal/runtime.iface" } { i32 0, i64 0, %"{{.*}}/runtime/internal/runtime.iface" undef }, %"{{.*}}/runtime/internal/runtime.iface" %8, 2
// CHECK-NEXT:   ret { i32, i64, %"{{.*}}/runtime/internal/runtime.iface" } %9
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_0
// CHECK-NEXT:   %10 = getelementptr inbounds %"{{.*}}/cl/_testgo/reader.stringReader", ptr %0, i32 0, i32 1
// CHECK-NEXT:   %11 = load i64, ptr %10, align 8
// CHECK-NEXT:   %12 = getelementptr inbounds %"{{.*}}/cl/_testgo/reader.stringReader", ptr %0, i32 0, i32 2
// CHECK-NEXT:   store i64 %11, ptr %12, align 8
// CHECK-NEXT:   %13 = getelementptr inbounds %"{{.*}}/cl/_testgo/reader.stringReader", ptr %0, i32 0, i32 1
// CHECK-NEXT:   %14 = load i64, ptr %13, align 8
// CHECK-NEXT:   %15 = getelementptr inbounds %"{{.*}}/cl/_testgo/reader.stringReader", ptr %0, i32 0, i32 0
// CHECK-NEXT:   %16 = load %"{{.*}}/runtime/internal/runtime.String", ptr %15, align 8
// CHECK-NEXT:   %17 = extractvalue %"{{.*}}/runtime/internal/runtime.String" %16, 0
// CHECK-NEXT:   %18 = extractvalue %"{{.*}}/runtime/internal/runtime.String" %16, 1
// CHECK-NEXT:   %19 = icmp slt i64 %14, 0
// CHECK-NEXT:   %20 = icmp uge i64 %14, %18
// CHECK-NEXT:   %21 = or i1 %20, %19
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.CheckIndexRange"(i1 %21, i64 %14, i1 true, i64 %18)
// CHECK-NEXT:   %22 = getelementptr inbounds i8, ptr %17, i64 %14
// CHECK-NEXT:   %23 = load i8, ptr %22, align 1
// CHECK-NEXT:   %24 = icmp ult i8 %23, -128
// CHECK-NEXT:   br i1 %24, label %_llgo_3, label %_llgo_4
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_3:                                          ; preds = %_llgo_2
// CHECK-NEXT:   %25 = getelementptr inbounds %"{{.*}}/cl/_testgo/reader.stringReader", ptr %0, i32 0, i32 1
// CHECK-NEXT:   %26 = load i64, ptr %25, align 8
// CHECK-NEXT:   %27 = add i64 %26, 1
// CHECK-NEXT:   %28 = getelementptr inbounds %"{{.*}}/cl/_testgo/reader.stringReader", ptr %0, i32 0, i32 1
// CHECK-NEXT:   store i64 %27, ptr %28, align 8
// CHECK-NEXT:   %29 = zext i8 %23 to i32
// CHECK-NEXT:   %30 = insertvalue { i32, i64, %"{{.*}}/runtime/internal/runtime.iface" } undef, i32 %29, 0
// CHECK-NEXT:   %31 = insertvalue { i32, i64, %"{{.*}}/runtime/internal/runtime.iface" } %30, i64 1, 1
// CHECK-NEXT:   %32 = insertvalue { i32, i64, %"{{.*}}/runtime/internal/runtime.iface" } %31, %"{{.*}}/runtime/internal/runtime.iface" zeroinitializer, 2
// CHECK-NEXT:   ret { i32, i64, %"{{.*}}/runtime/internal/runtime.iface" } %32
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_4:                                          ; preds = %_llgo_2
// CHECK-NEXT:   %33 = getelementptr inbounds %"{{.*}}/cl/_testgo/reader.stringReader", ptr %0, i32 0, i32 0
// CHECK-NEXT:   %34 = load %"{{.*}}/runtime/internal/runtime.String", ptr %33, align 8
// CHECK-NEXT:   %35 = getelementptr inbounds %"{{.*}}/cl/_testgo/reader.stringReader", ptr %0, i32 0, i32 1
// CHECK-NEXT:   %36 = load i64, ptr %35, align 8
// CHECK-NEXT:   %37 = extractvalue %"{{.*}}/runtime/internal/runtime.String" %34, 1
// CHECK-NEXT:   %38 = call %"{{.*}}/runtime/internal/runtime.String" @"{{.*}}/runtime/internal/runtime.StringSlice2"(%"{{.*}}/runtime/internal/runtime.String" %34, i64 %36, i64 %37, i1 true, i1 true)
// CHECK-NEXT:   %39 = call { i32, i64 } @"unicode/utf8.DecodeRuneInString"(%"{{.*}}/runtime/internal/runtime.String" %38)
// CHECK-NEXT:   %40 = extractvalue { i32, i64 } %39, 0
// CHECK-NEXT:   %41 = extractvalue { i32, i64 } %39, 1
// CHECK-NEXT:   %42 = getelementptr inbounds %"{{.*}}/cl/_testgo/reader.stringReader", ptr %0, i32 0, i32 1
// CHECK-NEXT:   %43 = load i64, ptr %42, align 8
// CHECK-NEXT:   %44 = add i64 %43, %41
// CHECK-NEXT:   %45 = getelementptr inbounds %"{{.*}}/cl/_testgo/reader.stringReader", ptr %0, i32 0, i32 1
// CHECK-NEXT:   store i64 %44, ptr %45, align 8
// CHECK-NEXT:   %46 = insertvalue { i32, i64, %"{{.*}}/runtime/internal/runtime.iface" } undef, i32 %40, 0
// CHECK-NEXT:   %47 = insertvalue { i32, i64, %"{{.*}}/runtime/internal/runtime.iface" } %46, i64 %41, 1
// CHECK-NEXT:   %48 = insertvalue { i32, i64, %"{{.*}}/runtime/internal/runtime.iface" } %47, %"{{.*}}/runtime/internal/runtime.iface" zeroinitializer, 2
// CHECK-NEXT:   ret { i32, i64, %"{{.*}}/runtime/internal/runtime.iface" } %48
// CHECK-NEXT: }

// CHECK-LABEL: define { i64, %"{{.*}}/runtime/internal/runtime.iface" } @"{{.*}}/cl/_testgo/reader.(*stringReader).Seek"(ptr %0, i64 %1, i64 %2){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %3 = getelementptr inbounds %"{{.*}}/cl/_testgo/reader.stringReader", ptr %0, i32 0, i32 2
// CHECK-NEXT:   store i64 -1, ptr %3, align 8
// CHECK-NEXT:   %4 = icmp eq i64 %2, 0
// CHECK-NEXT:   br i1 %4, label %_llgo_2, label %_llgo_4
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_5, %_llgo_3, %_llgo_2
// CHECK-NEXT:   %5 = phi i64 [ %1, %_llgo_2 ], [ %9, %_llgo_3 ], [ %14, %_llgo_5 ]
// CHECK-NEXT:   %6 = icmp slt i64 %5, 0
// CHECK-NEXT:   br i1 %6, label %_llgo_8, label %_llgo_9
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_0
// CHECK-NEXT:   br label %_llgo_1
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_3:                                          ; preds = %_llgo_4
// CHECK-NEXT:   %7 = getelementptr inbounds %"{{.*}}/cl/_testgo/reader.stringReader", ptr %0, i32 0, i32 1
// CHECK-NEXT:   %8 = load i64, ptr %7, align 8
// CHECK-NEXT:   %9 = add i64 %8, %1
// CHECK-NEXT:   br label %_llgo_1
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_4:                                          ; preds = %_llgo_0
// CHECK-NEXT:   %10 = icmp eq i64 %2, 1
// CHECK-NEXT:   br i1 %10, label %_llgo_3, label %_llgo_6
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_5:                                          ; preds = %_llgo_6
// CHECK-NEXT:   %11 = getelementptr inbounds %"{{.*}}/cl/_testgo/reader.stringReader", ptr %0, i32 0, i32 0
// CHECK-NEXT:   %12 = load %"{{.*}}/runtime/internal/runtime.String", ptr %11, align 8
// CHECK-NEXT:   %13 = extractvalue %"{{.*}}/runtime/internal/runtime.String" %12, 1
// CHECK-NEXT:   %14 = add i64 %13, %1
// CHECK-NEXT:   br label %_llgo_1
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_6:                                          ; preds = %_llgo_4
// CHECK-NEXT:   %15 = icmp eq i64 %2, 2
// CHECK-NEXT:   br i1 %15, label %_llgo_5, label %_llgo_7
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_7:                                          ; preds = %_llgo_6
// CHECK-NEXT:   %16 = call %"{{.*}}/runtime/internal/runtime.iface" @"{{.*}}/cl/_testgo/reader.newError"(%"{{.*}}/runtime/internal/runtime.String" { ptr @57, i64 34 })
// CHECK-NEXT:   %17 = insertvalue { i64, %"{{.*}}/runtime/internal/runtime.iface" } { i64 0, %"{{.*}}/runtime/internal/runtime.iface" undef }, %"{{.*}}/runtime/internal/runtime.iface" %16, 1
// CHECK-NEXT:   ret { i64, %"{{.*}}/runtime/internal/runtime.iface" } %17
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_8:                                          ; preds = %_llgo_1
// CHECK-NEXT:   %18 = call %"{{.*}}/runtime/internal/runtime.iface" @"{{.*}}/cl/_testgo/reader.newError"(%"{{.*}}/runtime/internal/runtime.String" { ptr @58, i64 37 })
// CHECK-NEXT:   %19 = insertvalue { i64, %"{{.*}}/runtime/internal/runtime.iface" } { i64 0, %"{{.*}}/runtime/internal/runtime.iface" undef }, %"{{.*}}/runtime/internal/runtime.iface" %18, 1
// CHECK-NEXT:   ret { i64, %"{{.*}}/runtime/internal/runtime.iface" } %19
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_9:                                          ; preds = %_llgo_1
// CHECK-NEXT:   %20 = getelementptr inbounds %"{{.*}}/cl/_testgo/reader.stringReader", ptr %0, i32 0, i32 1
// CHECK-NEXT:   store i64 %5, ptr %20, align 8
// CHECK-NEXT:   %21 = insertvalue { i64, %"{{.*}}/runtime/internal/runtime.iface" } undef, i64 %5, 0
// CHECK-NEXT:   %22 = insertvalue { i64, %"{{.*}}/runtime/internal/runtime.iface" } %21, %"{{.*}}/runtime/internal/runtime.iface" zeroinitializer, 1
// CHECK-NEXT:   ret { i64, %"{{.*}}/runtime/internal/runtime.iface" } %22
// CHECK-NEXT: }

// CHECK-LABEL: define i64 @"{{.*}}/cl/_testgo/reader.(*stringReader).Size"(ptr %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = getelementptr inbounds %"{{.*}}/cl/_testgo/reader.stringReader", ptr %0, i32 0, i32 0
// CHECK-NEXT:   %2 = load %"{{.*}}/runtime/internal/runtime.String", ptr %1, align 8
// CHECK-NEXT:   %3 = extractvalue %"{{.*}}/runtime/internal/runtime.String" %2, 1
// CHECK-NEXT:   ret i64 %3
// CHECK-NEXT: }

// CHECK-LABEL: define %"{{.*}}/runtime/internal/runtime.iface" @"{{.*}}/cl/_testgo/reader.(*stringReader).UnreadByte"(ptr %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = getelementptr inbounds %"{{.*}}/cl/_testgo/reader.stringReader", ptr %0, i32 0, i32 1
// CHECK-NEXT:   %2 = load i64, ptr %1, align 8
// CHECK-NEXT:   %3 = icmp sle i64 %2, 0
// CHECK-NEXT:   br i1 %3, label %_llgo_1, label %_llgo_2
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_0
// CHECK-NEXT:   %4 = call %"{{.*}}/runtime/internal/runtime.iface" @"{{.*}}/cl/_testgo/reader.newError"(%"{{.*}}/runtime/internal/runtime.String" { ptr @59, i64 48 })
// CHECK-NEXT:   ret %"{{.*}}/runtime/internal/runtime.iface" %4
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_0
// CHECK-NEXT:   %5 = getelementptr inbounds %"{{.*}}/cl/_testgo/reader.stringReader", ptr %0, i32 0, i32 2
// CHECK-NEXT:   store i64 -1, ptr %5, align 8
// CHECK-NEXT:   %6 = getelementptr inbounds %"{{.*}}/cl/_testgo/reader.stringReader", ptr %0, i32 0, i32 1
// CHECK-NEXT:   %7 = load i64, ptr %6, align 8
// CHECK-NEXT:   %8 = sub i64 %7, 1
// CHECK-NEXT:   %9 = getelementptr inbounds %"{{.*}}/cl/_testgo/reader.stringReader", ptr %0, i32 0, i32 1
// CHECK-NEXT:   store i64 %8, ptr %9, align 8
// CHECK-NEXT:   ret %"{{.*}}/runtime/internal/runtime.iface" zeroinitializer
// CHECK-NEXT: }

// CHECK-LABEL: define %"{{.*}}/runtime/internal/runtime.iface" @"{{.*}}/cl/_testgo/reader.(*stringReader).UnreadRune"(ptr %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = getelementptr inbounds %"{{.*}}/cl/_testgo/reader.stringReader", ptr %0, i32 0, i32 1
// CHECK-NEXT:   %2 = load i64, ptr %1, align 8
// CHECK-NEXT:   %3 = icmp sle i64 %2, 0
// CHECK-NEXT:   br i1 %3, label %_llgo_1, label %_llgo_2
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_0
// CHECK-NEXT:   %4 = call %"{{.*}}/runtime/internal/runtime.iface" @"{{.*}}/cl/_testgo/reader.newError"(%"{{.*}}/runtime/internal/runtime.String" { ptr @60, i64 49 })
// CHECK-NEXT:   ret %"{{.*}}/runtime/internal/runtime.iface" %4
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_0
// CHECK-NEXT:   %5 = getelementptr inbounds %"{{.*}}/cl/_testgo/reader.stringReader", ptr %0, i32 0, i32 2
// CHECK-NEXT:   %6 = load i64, ptr %5, align 8
// CHECK-NEXT:   %7 = icmp slt i64 %6, 0
// CHECK-NEXT:   br i1 %7, label %_llgo_3, label %_llgo_4
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_3:                                          ; preds = %_llgo_2
// CHECK-NEXT:   %8 = call %"{{.*}}/runtime/internal/runtime.iface" @"{{.*}}/cl/_testgo/reader.newError"(%"{{.*}}/runtime/internal/runtime.String" { ptr @61, i64 62 })
// CHECK-NEXT:   ret %"{{.*}}/runtime/internal/runtime.iface" %8
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_4:                                          ; preds = %_llgo_2
// CHECK-NEXT:   %9 = getelementptr inbounds %"{{.*}}/cl/_testgo/reader.stringReader", ptr %0, i32 0, i32 2
// CHECK-NEXT:   %10 = load i64, ptr %9, align 8
// CHECK-NEXT:   %11 = getelementptr inbounds %"{{.*}}/cl/_testgo/reader.stringReader", ptr %0, i32 0, i32 1
// CHECK-NEXT:   store i64 %10, ptr %11, align 8
// CHECK-NEXT:   %12 = getelementptr inbounds %"{{.*}}/cl/_testgo/reader.stringReader", ptr %0, i32 0, i32 2
// CHECK-NEXT:   store i64 -1, ptr %12, align 8
// CHECK-NEXT:   ret %"{{.*}}/runtime/internal/runtime.iface" zeroinitializer
// CHECK-NEXT: }

// CHECK-LABEL: define { i64, %"{{.*}}/runtime/internal/runtime.iface" } @"{{.*}}/cl/_testgo/reader.(*stringReader).WriteTo"(ptr %0, %"{{.*}}/runtime/internal/runtime.iface" %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = getelementptr inbounds %"{{.*}}/cl/_testgo/reader.stringReader", ptr %0, i32 0, i32 2
// CHECK-NEXT:   store i64 -1, ptr %2, align 8
// CHECK-NEXT:   %3 = getelementptr inbounds %"{{.*}}/cl/_testgo/reader.stringReader", ptr %0, i32 0, i32 1
// CHECK-NEXT:   %4 = load i64, ptr %3, align 8
// CHECK-NEXT:   %5 = getelementptr inbounds %"{{.*}}/cl/_testgo/reader.stringReader", ptr %0, i32 0, i32 0
// CHECK-NEXT:   %6 = load %"{{.*}}/runtime/internal/runtime.String", ptr %5, align 8
// CHECK-NEXT:   %7 = extractvalue %"{{.*}}/runtime/internal/runtime.String" %6, 1
// CHECK-NEXT:   %8 = icmp sge i64 %4, %7
// CHECK-NEXT:   br i1 %8, label %_llgo_1, label %_llgo_2
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_0
// CHECK-NEXT:   ret { i64, %"{{.*}}/runtime/internal/runtime.iface" } zeroinitializer
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_0
// CHECK-NEXT:   %9 = getelementptr inbounds %"{{.*}}/cl/_testgo/reader.stringReader", ptr %0, i32 0, i32 0
// CHECK-NEXT:   %10 = load %"{{.*}}/runtime/internal/runtime.String", ptr %9, align 8
// CHECK-NEXT:   %11 = getelementptr inbounds %"{{.*}}/cl/_testgo/reader.stringReader", ptr %0, i32 0, i32 1
// CHECK-NEXT:   %12 = load i64, ptr %11, align 8
// CHECK-NEXT:   %13 = extractvalue %"{{.*}}/runtime/internal/runtime.String" %10, 1
// CHECK-NEXT:   %14 = call %"{{.*}}/runtime/internal/runtime.String" @"{{.*}}/runtime/internal/runtime.StringSlice2"(%"{{.*}}/runtime/internal/runtime.String" %10, i64 %12, i64 %13, i1 true, i1 true)
// CHECK-NEXT:   %15 = call { i64, %"{{.*}}/runtime/internal/runtime.iface" } @"{{.*}}/cl/_testgo/reader.WriteString"(%"{{.*}}/runtime/internal/runtime.iface" %1, %"{{.*}}/runtime/internal/runtime.String" %14)
// CHECK-NEXT:   %16 = extractvalue { i64, %"{{.*}}/runtime/internal/runtime.iface" } %15, 0
// CHECK-NEXT:   %17 = extractvalue { i64, %"{{.*}}/runtime/internal/runtime.iface" } %15, 1
// CHECK-NEXT:   %18 = extractvalue %"{{.*}}/runtime/internal/runtime.String" %14, 1
// CHECK-NEXT:   %19 = icmp sgt i64 %16, %18
// CHECK-NEXT:   br i1 %19, label %_llgo_3, label %_llgo_4
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_3:                                          ; preds = %_llgo_2
// CHECK-NEXT:   %20 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @62, i64 48 }, ptr %20, align 8
// CHECK-NEXT:   %21 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_string, ptr undef }, ptr %20, 1
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.Panic"(%"{{.*}}/runtime/internal/runtime.eface" %21)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_4:                                          ; preds = %_llgo_2
// CHECK-NEXT:   %22 = getelementptr inbounds %"{{.*}}/cl/_testgo/reader.stringReader", ptr %0, i32 0, i32 1
// CHECK-NEXT:   %23 = load i64, ptr %22, align 8
// CHECK-NEXT:   %24 = add i64 %23, %16
// CHECK-NEXT:   %25 = getelementptr inbounds %"{{.*}}/cl/_testgo/reader.stringReader", ptr %0, i32 0, i32 1
// CHECK-NEXT:   store i64 %24, ptr %25, align 8
// CHECK-NEXT:   %26 = extractvalue %"{{.*}}/runtime/internal/runtime.String" %14, 1
// CHECK-NEXT:   %27 = icmp ne i64 %16, %26
// CHECK-NEXT:   br i1 %27, label %_llgo_7, label %_llgo_6
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_5:                                          ; preds = %_llgo_7
// CHECK-NEXT:   %28 = load %"{{.*}}/runtime/internal/runtime.iface", ptr @"{{.*}}/cl/_testgo/reader.ErrShortWrite", align 8
// CHECK-NEXT:   br label %_llgo_6
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_6:                                          ; preds = %_llgo_5, %_llgo_7, %_llgo_4
// CHECK-NEXT:   %29 = phi %"{{.*}}/runtime/internal/runtime.iface" [ %17, %_llgo_4 ], [ %17, %_llgo_7 ], [ %28, %_llgo_5 ]
// CHECK-NEXT:   %30 = insertvalue { i64, %"{{.*}}/runtime/internal/runtime.iface" } undef, i64 %16, 0
// CHECK-NEXT:   %31 = insertvalue { i64, %"{{.*}}/runtime/internal/runtime.iface" } %30, %"{{.*}}/runtime/internal/runtime.iface" %29, 1
// CHECK-NEXT:   ret { i64, %"{{.*}}/runtime/internal/runtime.iface" } %31
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_7:                                          ; preds = %_llgo_4
// CHECK-NEXT:   %32 = call ptr @"{{.*}}/runtime/internal/runtime.IfaceType"(%"{{.*}}/runtime/internal/runtime.iface" %17)
// CHECK-NEXT:   %33 = extractvalue %"{{.*}}/runtime/internal/runtime.iface" %17, 1
// CHECK-NEXT:   %34 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" undef, ptr %32, 0
// CHECK-NEXT:   %35 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" %34, ptr %33, 1
// CHECK-NEXT:   %36 = call ptr @"{{.*}}/runtime/internal/runtime.IfaceType"(%"{{.*}}/runtime/internal/runtime.iface" zeroinitializer)
// CHECK-NEXT:   %37 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" undef, ptr %36, 0
// CHECK-NEXT:   %38 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" %37, ptr null, 1
// CHECK-NEXT:   %39 = call i1 @"{{.*}}/runtime/internal/runtime.EfaceEqual"(%"{{.*}}/runtime/internal/runtime.eface" %35, %"{{.*}}/runtime/internal/runtime.eface" %38)
// CHECK-NEXT:   br i1 %39, label %_llgo_5, label %_llgo_6
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce i1 @"__llgo_stub.{{.*}}/runtime/internal/runtime.interequal"(ptr %0, ptr %1, ptr %2){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %3 = tail call i1 @"{{.*}}/runtime/internal/runtime.interequal"(ptr %1, ptr %2)
// CHECK-NEXT:   ret i1 %3
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce i1 @"__llgo_stub.{{.*}}/runtime/internal/runtime.memequal8"(ptr %0, ptr %1, ptr %2){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %3 = tail call i1 @"{{.*}}/runtime/internal/runtime.memequal8"(ptr %1, ptr %2)
// CHECK-NEXT:   ret i1 %3
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce i1 @"__llgo_stub.{{.*}}/runtime/internal/runtime.memequal64"(ptr %0, ptr %1, ptr %2){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %3 = tail call i1 @"{{.*}}/runtime/internal/runtime.memequal64"(ptr %1, ptr %2)
// CHECK-NEXT:   ret i1 %3
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce %"{{.*}}/runtime/internal/runtime.iface" @"__llgo_stub.{{.*}}/cl/_testgo/reader.(*nopCloserWriterTo).Close"(ptr %0, ptr %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = tail call %"{{.*}}/runtime/internal/runtime.iface" @"{{.*}}/cl/_testgo/reader.(*nopCloserWriterTo).Close"(ptr %1)
// CHECK-NEXT:   ret %"{{.*}}/runtime/internal/runtime.iface" %2
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce { i64, %"{{.*}}/runtime/internal/runtime.iface" } @"__llgo_stub.{{.*}}/cl/_testgo/reader.(*nopCloserWriterTo).Read"(ptr %0, ptr %1, %"{{.*}}/runtime/internal/runtime.Slice" %2){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %3 = tail call { i64, %"{{.*}}/runtime/internal/runtime.iface" } @"{{.*}}/cl/_testgo/reader.(*nopCloserWriterTo).Read"(ptr %1, %"{{.*}}/runtime/internal/runtime.Slice" %2)
// CHECK-NEXT:   ret { i64, %"{{.*}}/runtime/internal/runtime.iface" } %3
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce { i64, %"{{.*}}/runtime/internal/runtime.iface" } @"__llgo_stub.{{.*}}/cl/_testgo/reader.(*nopCloserWriterTo).WriteTo"(ptr %0, ptr %1, %"{{.*}}/runtime/internal/runtime.iface" %2){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %3 = tail call { i64, %"{{.*}}/runtime/internal/runtime.iface" } @"{{.*}}/cl/_testgo/reader.(*nopCloserWriterTo).WriteTo"(ptr %1, %"{{.*}}/runtime/internal/runtime.iface" %2)
// CHECK-NEXT:   ret { i64, %"{{.*}}/runtime/internal/runtime.iface" } %3
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce %"{{.*}}/runtime/internal/runtime.iface" @"__llgo_stub.{{.*}}/cl/_testgo/reader.nopCloserWriterTo.Close"(ptr %0, %"{{.*}}/cl/_testgo/reader.nopCloserWriterTo" %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = tail call %"{{.*}}/runtime/internal/runtime.iface" @"{{.*}}/cl/_testgo/reader.nopCloserWriterTo.Close"(%"{{.*}}/cl/_testgo/reader.nopCloserWriterTo" %1)
// CHECK-NEXT:   ret %"{{.*}}/runtime/internal/runtime.iface" %2
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce { i64, %"{{.*}}/runtime/internal/runtime.iface" } @"__llgo_stub.{{.*}}/cl/_testgo/reader.nopCloserWriterTo.Read"(ptr %0, %"{{.*}}/cl/_testgo/reader.nopCloserWriterTo" %1, %"{{.*}}/runtime/internal/runtime.Slice" %2){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %3 = tail call { i64, %"{{.*}}/runtime/internal/runtime.iface" } @"{{.*}}/cl/_testgo/reader.nopCloserWriterTo.Read"(%"{{.*}}/cl/_testgo/reader.nopCloserWriterTo" %1, %"{{.*}}/runtime/internal/runtime.Slice" %2)
// CHECK-NEXT:   ret { i64, %"{{.*}}/runtime/internal/runtime.iface" } %3
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce { i64, %"{{.*}}/runtime/internal/runtime.iface" } @"__llgo_stub.{{.*}}/cl/_testgo/reader.nopCloserWriterTo.WriteTo"(ptr %0, %"{{.*}}/cl/_testgo/reader.nopCloserWriterTo" %1, %"{{.*}}/runtime/internal/runtime.iface" %2){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %3 = tail call { i64, %"{{.*}}/runtime/internal/runtime.iface" } @"{{.*}}/cl/_testgo/reader.nopCloserWriterTo.WriteTo"(%"{{.*}}/cl/_testgo/reader.nopCloserWriterTo" %1, %"{{.*}}/runtime/internal/runtime.iface" %2)
// CHECK-NEXT:   ret { i64, %"{{.*}}/runtime/internal/runtime.iface" } %3
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce %"{{.*}}/runtime/internal/runtime.iface" @"__llgo_stub.{{.*}}/cl/_testgo/reader.(*nopCloser).Close"(ptr %0, ptr %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = tail call %"{{.*}}/runtime/internal/runtime.iface" @"{{.*}}/cl/_testgo/reader.(*nopCloser).Close"(ptr %1)
// CHECK-NEXT:   ret %"{{.*}}/runtime/internal/runtime.iface" %2
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce { i64, %"{{.*}}/runtime/internal/runtime.iface" } @"__llgo_stub.{{.*}}/cl/_testgo/reader.(*nopCloser).Read"(ptr %0, ptr %1, %"{{.*}}/runtime/internal/runtime.Slice" %2){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %3 = tail call { i64, %"{{.*}}/runtime/internal/runtime.iface" } @"{{.*}}/cl/_testgo/reader.(*nopCloser).Read"(ptr %1, %"{{.*}}/runtime/internal/runtime.Slice" %2)
// CHECK-NEXT:   ret { i64, %"{{.*}}/runtime/internal/runtime.iface" } %3
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce %"{{.*}}/runtime/internal/runtime.iface" @"__llgo_stub.{{.*}}/cl/_testgo/reader.nopCloser.Close"(ptr %0, %"{{.*}}/cl/_testgo/reader.nopCloser" %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = tail call %"{{.*}}/runtime/internal/runtime.iface" @"{{.*}}/cl/_testgo/reader.nopCloser.Close"(%"{{.*}}/cl/_testgo/reader.nopCloser" %1)
// CHECK-NEXT:   ret %"{{.*}}/runtime/internal/runtime.iface" %2
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce { i64, %"{{.*}}/runtime/internal/runtime.iface" } @"__llgo_stub.{{.*}}/cl/_testgo/reader.nopCloser.Read"(ptr %0, %"{{.*}}/cl/_testgo/reader.nopCloser" %1, %"{{.*}}/runtime/internal/runtime.Slice" %2){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %3 = tail call { i64, %"{{.*}}/runtime/internal/runtime.iface" } @"{{.*}}/cl/_testgo/reader.nopCloser.Read"(%"{{.*}}/cl/_testgo/reader.nopCloser" %1, %"{{.*}}/runtime/internal/runtime.Slice" %2)
// CHECK-NEXT:   ret { i64, %"{{.*}}/runtime/internal/runtime.iface" } %3
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce i64 @"__llgo_stub.{{.*}}/cl/_testgo/reader.(*stringReader).Len"(ptr %0, ptr %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = tail call i64 @"{{.*}}/cl/_testgo/reader.(*stringReader).Len"(ptr %1)
// CHECK-NEXT:   ret i64 %2
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce { i64, %"{{.*}}/runtime/internal/runtime.iface" } @"__llgo_stub.{{.*}}/cl/_testgo/reader.(*stringReader).Read"(ptr %0, ptr %1, %"{{.*}}/runtime/internal/runtime.Slice" %2){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %3 = tail call { i64, %"{{.*}}/runtime/internal/runtime.iface" } @"{{.*}}/cl/_testgo/reader.(*stringReader).Read"(ptr %1, %"{{.*}}/runtime/internal/runtime.Slice" %2)
// CHECK-NEXT:   ret { i64, %"{{.*}}/runtime/internal/runtime.iface" } %3
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce { i64, %"{{.*}}/runtime/internal/runtime.iface" } @"__llgo_stub.{{.*}}/cl/_testgo/reader.(*stringReader).ReadAt"(ptr %0, ptr %1, %"{{.*}}/runtime/internal/runtime.Slice" %2, i64 %3){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %4 = tail call { i64, %"{{.*}}/runtime/internal/runtime.iface" } @"{{.*}}/cl/_testgo/reader.(*stringReader).ReadAt"(ptr %1, %"{{.*}}/runtime/internal/runtime.Slice" %2, i64 %3)
// CHECK-NEXT:   ret { i64, %"{{.*}}/runtime/internal/runtime.iface" } %4
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce { i8, %"{{.*}}/runtime/internal/runtime.iface" } @"__llgo_stub.{{.*}}/cl/_testgo/reader.(*stringReader).ReadByte"(ptr %0, ptr %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = tail call { i8, %"{{.*}}/runtime/internal/runtime.iface" } @"{{.*}}/cl/_testgo/reader.(*stringReader).ReadByte"(ptr %1)
// CHECK-NEXT:   ret { i8, %"{{.*}}/runtime/internal/runtime.iface" } %2
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce { i32, i64, %"{{.*}}/runtime/internal/runtime.iface" } @"__llgo_stub.{{.*}}/cl/_testgo/reader.(*stringReader).ReadRune"(ptr %0, ptr %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = tail call { i32, i64, %"{{.*}}/runtime/internal/runtime.iface" } @"{{.*}}/cl/_testgo/reader.(*stringReader).ReadRune"(ptr %1)
// CHECK-NEXT:   ret { i32, i64, %"{{.*}}/runtime/internal/runtime.iface" } %2
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce { i64, %"{{.*}}/runtime/internal/runtime.iface" } @"__llgo_stub.{{.*}}/cl/_testgo/reader.(*stringReader).Seek"(ptr %0, ptr %1, i64 %2, i64 %3){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %4 = tail call { i64, %"{{.*}}/runtime/internal/runtime.iface" } @"{{.*}}/cl/_testgo/reader.(*stringReader).Seek"(ptr %1, i64 %2, i64 %3)
// CHECK-NEXT:   ret { i64, %"{{.*}}/runtime/internal/runtime.iface" } %4
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce i64 @"__llgo_stub.{{.*}}/cl/_testgo/reader.(*stringReader).Size"(ptr %0, ptr %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = tail call i64 @"{{.*}}/cl/_testgo/reader.(*stringReader).Size"(ptr %1)
// CHECK-NEXT:   ret i64 %2
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce %"{{.*}}/runtime/internal/runtime.iface" @"__llgo_stub.{{.*}}/cl/_testgo/reader.(*stringReader).UnreadByte"(ptr %0, ptr %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = tail call %"{{.*}}/runtime/internal/runtime.iface" @"{{.*}}/cl/_testgo/reader.(*stringReader).UnreadByte"(ptr %1)
// CHECK-NEXT:   ret %"{{.*}}/runtime/internal/runtime.iface" %2
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce %"{{.*}}/runtime/internal/runtime.iface" @"__llgo_stub.{{.*}}/cl/_testgo/reader.(*stringReader).UnreadRune"(ptr %0, ptr %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = tail call %"{{.*}}/runtime/internal/runtime.iface" @"{{.*}}/cl/_testgo/reader.(*stringReader).UnreadRune"(ptr %1)
// CHECK-NEXT:   ret %"{{.*}}/runtime/internal/runtime.iface" %2
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce { i64, %"{{.*}}/runtime/internal/runtime.iface" } @"__llgo_stub.{{.*}}/cl/_testgo/reader.(*stringReader).WriteTo"(ptr %0, ptr %1, %"{{.*}}/runtime/internal/runtime.iface" %2){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %3 = tail call { i64, %"{{.*}}/runtime/internal/runtime.iface" } @"{{.*}}/cl/_testgo/reader.(*stringReader).WriteTo"(ptr %1, %"{{.*}}/runtime/internal/runtime.iface" %2)
// CHECK-NEXT:   ret { i64, %"{{.*}}/runtime/internal/runtime.iface" } %3
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce %"{{.*}}/runtime/internal/runtime.String" @"__llgo_stub.{{.*}}/cl/_testgo/reader.(*errorString).Error"(ptr %0, ptr %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = tail call %"{{.*}}/runtime/internal/runtime.String" @"{{.*}}/cl/_testgo/reader.(*errorString).Error"(ptr %1)
// CHECK-NEXT:   ret %"{{.*}}/runtime/internal/runtime.String" %2
// CHECK-NEXT: }
