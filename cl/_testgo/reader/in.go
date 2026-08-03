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

// CHECK-LABEL: define %"{{.*}}/runtime/internal/runtime.iface" @main.NopCloser(%"{{.*}}/runtime/internal/runtime.iface" %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = call ptr @"{{.*}}/runtime/internal/runtime.IfaceType"(%"{{.*}}/runtime/internal/runtime.iface" %0)
// CHECK-NEXT:   %2 = call i1 @"{{.*}}/runtime/internal/runtime.Implements"(ptr @_llgo_main.WriterTo, ptr %1)
// CHECK-NEXT:   br i1 %2, label %_llgo_3, label %_llgo_4
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_5
// CHECK-NEXT:   %3 = alloca %main.nopCloserWriterTo, align 8
// CHECK-NEXT:   call void @llvm.memset.p0.i64(ptr %3, i8 0, i64 16, i1 false)
// CHECK-NEXT:   %4 = getelementptr inbounds %main.nopCloserWriterTo, ptr %3, i32 0, i32 0
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.iface" %0, ptr %4, align 8
// CHECK-NEXT:   %5 = load %main.nopCloserWriterTo, ptr %3, align 8
// CHECK-NEXT:   %6 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %main.nopCloserWriterTo %5, ptr %6, align 8
// CHECK-NEXT:   %7 = call ptr @"{{.*}}/runtime/internal/runtime.NewItab"(ptr @"_llgo_iface${{[-A-Za-z0-9_]+}}", ptr @_llgo_main.nopCloserWriterTo)
// CHECK-NEXT:   %8 = insertvalue %"{{.*}}/runtime/internal/runtime.iface" undef, ptr %7, 0
// CHECK-NEXT:   %9 = insertvalue %"{{.*}}/runtime/internal/runtime.iface" %8, ptr %6, 1
// CHECK-NEXT:   ret %"{{.*}}/runtime/internal/runtime.iface" %9
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_5
// CHECK-NEXT:   %10 = alloca %main.nopCloser, align 8
// CHECK-NEXT:   call void @llvm.memset.p0.i64(ptr %10, i8 0, i64 16, i1 false)
// CHECK-NEXT:   %11 = getelementptr inbounds %main.nopCloser, ptr %10, i32 0, i32 0
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.iface" %0, ptr %11, align 8
// CHECK-NEXT:   %12 = load %main.nopCloser, ptr %10, align 8
// CHECK-NEXT:   %13 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %main.nopCloser %12, ptr %13, align 8
// CHECK-NEXT:   %14 = call ptr @"{{.*}}/runtime/internal/runtime.NewItab"(ptr @"_llgo_iface${{[-A-Za-z0-9_]+}}", ptr @_llgo_main.nopCloser)
// CHECK-NEXT:   %15 = insertvalue %"{{.*}}/runtime/internal/runtime.iface" undef, ptr %14, 0
// CHECK-NEXT:   %16 = insertvalue %"{{.*}}/runtime/internal/runtime.iface" %15, ptr %13, 1
// CHECK-NEXT:   ret %"{{.*}}/runtime/internal/runtime.iface" %16
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_3:                                          ; preds = %_llgo_0
// CHECK-NEXT:   %17 = extractvalue %"{{.*}}/runtime/internal/runtime.iface" %0, 1
// CHECK-NEXT:   %18 = call ptr @"{{.*}}/runtime/internal/runtime.NewItab"(ptr @"_llgo_iface${{[-A-Za-z0-9_]+}}", ptr %1)
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

// CHECK-LABEL: define { %"{{.*}}/runtime/internal/runtime.Slice", %"{{.*}}/runtime/internal/runtime.iface" } @main.ReadAll(%"{{.*}}/runtime/internal/runtime.iface" %0){{.*}} {
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
// CHECK-NEXT:   %34 = load %"{{.*}}/runtime/internal/runtime.iface", ptr @main.EOF, align 8
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

// CHECK-LABEL: define { i64, %"{{.*}}/runtime/internal/runtime.iface" } @main.WriteString(%"{{.*}}/runtime/internal/runtime.iface" %0, %"{{.*}}/runtime/internal/runtime.String" %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = call ptr @"{{.*}}/runtime/internal/runtime.IfaceType"(%"{{.*}}/runtime/internal/runtime.iface" %0)
// CHECK-NEXT:   %3 = call i1 @"{{.*}}/runtime/internal/runtime.Implements"(ptr @_llgo_main.StringWriter, ptr %2)
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
// CHECK-NEXT:   %32 = call ptr @"{{.*}}/runtime/internal/runtime.NewItab"(ptr @"_llgo_iface${{[-A-Za-z0-9_]+}}", ptr %2)
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

// CHECK-LABEL: define %"{{.*}}/runtime/internal/runtime.String" @"main.(*errorString).Error"(ptr %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = getelementptr inbounds %main.errorString, ptr %0, i32 0, i32 0
// CHECK-NEXT:   %2 = icmp eq ptr %0, null
// CHECK-NEXT:   br i1 %2, label %3, label %4
// CHECK-EMPTY:
// CHECK-NEXT: 3:                                                ; preds = %_llgo_0
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 true)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: 4:                                                ; preds = %_llgo_0
// CHECK-NEXT:   %5 = load %"{{.*}}/runtime/internal/runtime.String", ptr %1, align 8
// CHECK-NEXT:   ret %"{{.*}}/runtime/internal/runtime.String" %5
// CHECK-NEXT: }

// CHECK-LABEL: define void @main.init(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = load i1, ptr @"main.init$guard", align 1
// CHECK-NEXT:   br i1 %0, label %_llgo_2, label %_llgo_1
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_0
// CHECK-NEXT:   store i1 true, ptr @"main.init$guard", align 1
// CHECK-NEXT:   call void @"unicode/utf8.init"()
// CHECK-NEXT:   %1 = call %"{{.*}}/runtime/internal/runtime.iface" @main.newError(%"{{.*}}/runtime/internal/runtime.String" { ptr @28, i64 3 })
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.iface" %1, ptr @main.EOF, align 8
// CHECK-NEXT:   %2 = call %"{{.*}}/runtime/internal/runtime.iface" @main.newError(%"{{.*}}/runtime/internal/runtime.String" { ptr @29, i64 11 })
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.iface" %2, ptr @main.ErrShortWrite, align 8
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

// CHECK-LABEL: define void @main.main(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 32)
// CHECK-NEXT:   %1 = getelementptr inbounds %main.stringReader, ptr %0, i32 0, i32 0
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @30, i64 11 }, ptr %1, align 8
// CHECK-NEXT:   %2 = call ptr @"{{.*}}/runtime/internal/runtime.NewItab"(ptr @"_llgo_iface${{[-A-Za-z0-9_]+}}", ptr @"*_llgo_main.stringReader")
// CHECK-NEXT:   %3 = insertvalue %"{{.*}}/runtime/internal/runtime.iface" undef, ptr %2, 0
// CHECK-NEXT:   %4 = insertvalue %"{{.*}}/runtime/internal/runtime.iface" %3, ptr %0, 1
// CHECK-NEXT:   %5 = call { %"{{.*}}/runtime/internal/runtime.Slice", %"{{.*}}/runtime/internal/runtime.iface" } @main.ReadAll(%"{{.*}}/runtime/internal/runtime.iface" %4)
// CHECK-NEXT:   %6 = extractvalue { %"{{.*}}/runtime/internal/runtime.Slice", %"{{.*}}/runtime/internal/runtime.iface" } %5, 0
// CHECK-NEXT:   %7 = extractvalue { %"{{.*}}/runtime/internal/runtime.Slice", %"{{.*}}/runtime/internal/runtime.iface" } %5, 1
// CHECK-NEXT:   %8 = call %"{{.*}}/runtime/internal/runtime.String" @"{{.*}}/runtime/internal/runtime.StringFromBytes"(%"{{.*}}/runtime/internal/runtime.Slice" %6)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintString"(%"{{.*}}/runtime/internal/runtime.String" %8)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintIface"(%"{{.*}}/runtime/internal/runtime.iface" %7)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   ret void
// CHECK-NEXT: }

// CHECK-LABEL: define %"{{.*}}/runtime/internal/runtime.iface" @main.newError(%"{{.*}}/runtime/internal/runtime.String" %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 16)
// CHECK-NEXT:   %2 = getelementptr inbounds %main.errorString, ptr %1, i32 0, i32 0
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" %0, ptr %2, align 8
// CHECK-NEXT:   %3 = call ptr @"{{.*}}/runtime/internal/runtime.NewItab"(ptr @"_llgo_iface${{[-A-Za-z0-9_]+}}", ptr @"*_llgo_main.errorString")
// CHECK-NEXT:   %4 = insertvalue %"{{.*}}/runtime/internal/runtime.iface" undef, ptr %3, 0
// CHECK-NEXT:   %5 = insertvalue %"{{.*}}/runtime/internal/runtime.iface" %4, ptr %1, 1
// CHECK-NEXT:   ret %"{{.*}}/runtime/internal/runtime.iface" %5
// CHECK-NEXT: }

// CHECK-LABEL: define %"{{.*}}/runtime/internal/runtime.iface" @main.nopCloser.Close(%main.nopCloser %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   ret %"{{.*}}/runtime/internal/runtime.iface" zeroinitializer
// CHECK-NEXT: }

func main() {
	r := &stringReader{s: "hello world"}
	data, err := ReadAll(r)
	println(string(data), err)
}

// CHECK-LABEL: define { i64, %"{{.*}}/runtime/internal/runtime.iface" } @main.nopCloser.Read(%main.nopCloser %0, %"{{.*}}/runtime/internal/runtime.Slice" %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = alloca %main.nopCloser, align 8
// CHECK-NEXT:   call void @llvm.memset.p0.i64(ptr %2, i8 0, i64 16, i1 false)
// CHECK-NEXT:   store %main.nopCloser %0, ptr %2, align 8
// CHECK-NEXT:   %3 = getelementptr inbounds %main.nopCloser, ptr %2, i32 0, i32 0
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

// CHECK-LABEL: define %"{{.*}}/runtime/internal/runtime.iface" @"main.(*nopCloser).Close"(ptr %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = icmp eq ptr %0, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PanicWrapNilPointer"(i1 %1, %"{{.*}}/runtime/internal/runtime.String" { ptr @53, i64 50 }, %"{{.*}}/runtime/internal/runtime.String" { ptr @17, i64 5 })
// CHECK-NEXT:   %2 = load %main.nopCloser, ptr %0, align 8
// CHECK-NEXT:   %3 = call %"{{.*}}/runtime/internal/runtime.iface" @main.nopCloser.Close(%main.nopCloser %2)
// CHECK-NEXT:   ret %"{{.*}}/runtime/internal/runtime.iface" %3
// CHECK-NEXT: }

// CHECK-LABEL: define { i64, %"{{.*}}/runtime/internal/runtime.iface" } @"main.(*nopCloser).Read"(ptr %0, %"{{.*}}/runtime/internal/runtime.Slice" %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = getelementptr inbounds %main.nopCloser, ptr %0, i32 0, i32 0
// CHECK-NEXT:   %3 = icmp eq ptr %0, null
// CHECK-NEXT:   br i1 %3, label %4, label %5
// CHECK-EMPTY:
// CHECK-NEXT: 4:                                                ; preds = %_llgo_0
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 true)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: 5:                                                ; preds = %_llgo_0
// CHECK-NEXT:   %6 = load %"{{.*}}/runtime/internal/runtime.iface", ptr %2, align 8
// CHECK-NEXT:   %7 = call ptr @"{{.*}}/runtime/internal/runtime.IfacePtrData"(%"{{.*}}/runtime/internal/runtime.iface" %6)
// CHECK-NEXT:   %8 = extractvalue %"{{.*}}/runtime/internal/runtime.iface" %6, 0
// CHECK-NEXT:   %9 = getelementptr ptr, ptr %8, i64 3
// CHECK-NEXT:   %10 = load ptr, ptr %9, align 8
// CHECK-NEXT:   %11 = insertvalue { ptr, ptr } undef, ptr %10, 0
// CHECK-NEXT:   %12 = insertvalue { ptr, ptr } %11, ptr %7, 1
// CHECK-NEXT:   %13 = extractvalue { ptr, ptr } %12, 1
// CHECK-NEXT:   %14 = extractvalue { ptr, ptr } %12, 0
// CHECK-NEXT:   %15 = call { i64, %"{{.*}}/runtime/internal/runtime.iface" } %14(ptr %13, %"{{.*}}/runtime/internal/runtime.Slice" %1)
// CHECK-NEXT:   %16 = extractvalue { i64, %"{{.*}}/runtime/internal/runtime.iface" } %15, 0
// CHECK-NEXT:   %17 = extractvalue { i64, %"{{.*}}/runtime/internal/runtime.iface" } %15, 1
// CHECK-NEXT:   %18 = insertvalue { i64, %"{{.*}}/runtime/internal/runtime.iface" } undef, i64 %16, 0
// CHECK-NEXT:   %19 = insertvalue { i64, %"{{.*}}/runtime/internal/runtime.iface" } %18, %"{{.*}}/runtime/internal/runtime.iface" %17, 1
// CHECK-NEXT:   ret { i64, %"{{.*}}/runtime/internal/runtime.iface" } %19
// CHECK-NEXT: }

// CHECK-LABEL: define %"{{.*}}/runtime/internal/runtime.iface" @main.nopCloserWriterTo.Close(%main.nopCloserWriterTo %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   ret %"{{.*}}/runtime/internal/runtime.iface" zeroinitializer
// CHECK-NEXT: }

// CHECK-LABEL: define { i64, %"{{.*}}/runtime/internal/runtime.iface" } @main.nopCloserWriterTo.Read(%main.nopCloserWriterTo %0, %"{{.*}}/runtime/internal/runtime.Slice" %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = alloca %main.nopCloserWriterTo, align 8
// CHECK-NEXT:   call void @llvm.memset.p0.i64(ptr %2, i8 0, i64 16, i1 false)
// CHECK-NEXT:   store %main.nopCloserWriterTo %0, ptr %2, align 8
// CHECK-NEXT:   %3 = getelementptr inbounds %main.nopCloserWriterTo, ptr %2, i32 0, i32 0
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

// CHECK-LABEL: define { i64, %"{{.*}}/runtime/internal/runtime.iface" } @main.nopCloserWriterTo.WriteTo(%main.nopCloserWriterTo %0, %"{{.*}}/runtime/internal/runtime.iface" %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = alloca %main.nopCloserWriterTo, align 8
// CHECK-NEXT:   call void @llvm.memset.p0.i64(ptr %2, i8 0, i64 16, i1 false)
// CHECK-NEXT:   store %main.nopCloserWriterTo %0, ptr %2, align 8
// CHECK-NEXT:   %3 = getelementptr inbounds %main.nopCloserWriterTo, ptr %2, i32 0, i32 0
// CHECK-NEXT:   %4 = load %"{{.*}}/runtime/internal/runtime.iface", ptr %3, align 8
// CHECK-NEXT:   %5 = call ptr @"{{.*}}/runtime/internal/runtime.IfaceType"(%"{{.*}}/runtime/internal/runtime.iface" %4)
// CHECK-NEXT:   %6 = call i1 @"{{.*}}/runtime/internal/runtime.Implements"(ptr @_llgo_main.WriterTo, ptr %5)
// CHECK-NEXT:   br i1 %6, label %_llgo_1, label %_llgo_2
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_0
// CHECK-NEXT:   %7 = extractvalue %"{{.*}}/runtime/internal/runtime.iface" %4, 1
// CHECK-NEXT:   %8 = call ptr @"{{.*}}/runtime/internal/runtime.NewItab"(ptr @"_llgo_iface${{[-A-Za-z0-9_]+}}", ptr %5)
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

// CHECK-LABEL: define %"{{.*}}/runtime/internal/runtime.iface" @"main.(*nopCloserWriterTo).Close"(ptr %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = icmp eq ptr %0, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PanicWrapNilPointer"(i1 %1, %"{{.*}}/runtime/internal/runtime.String" { ptr @55, i64 58 }, %"{{.*}}/runtime/internal/runtime.String" { ptr @17, i64 5 })
// CHECK-NEXT:   %2 = load %main.nopCloserWriterTo, ptr %0, align 8
// CHECK-NEXT:   %3 = call %"{{.*}}/runtime/internal/runtime.iface" @main.nopCloserWriterTo.Close(%main.nopCloserWriterTo %2)
// CHECK-NEXT:   ret %"{{.*}}/runtime/internal/runtime.iface" %3
// CHECK-NEXT: }

// CHECK-LABEL: define { i64, %"{{.*}}/runtime/internal/runtime.iface" } @"main.(*nopCloserWriterTo).Read"(ptr %0, %"{{.*}}/runtime/internal/runtime.Slice" %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = getelementptr inbounds %main.nopCloserWriterTo, ptr %0, i32 0, i32 0
// CHECK-NEXT:   %3 = icmp eq ptr %0, null
// CHECK-NEXT:   br i1 %3, label %4, label %5
// CHECK-EMPTY:
// CHECK-NEXT: 4:                                                ; preds = %_llgo_0
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 true)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: 5:                                                ; preds = %_llgo_0
// CHECK-NEXT:   %6 = load %"{{.*}}/runtime/internal/runtime.iface", ptr %2, align 8
// CHECK-NEXT:   %7 = call ptr @"{{.*}}/runtime/internal/runtime.IfacePtrData"(%"{{.*}}/runtime/internal/runtime.iface" %6)
// CHECK-NEXT:   %8 = extractvalue %"{{.*}}/runtime/internal/runtime.iface" %6, 0
// CHECK-NEXT:   %9 = getelementptr ptr, ptr %8, i64 3
// CHECK-NEXT:   %10 = load ptr, ptr %9, align 8
// CHECK-NEXT:   %11 = insertvalue { ptr, ptr } undef, ptr %10, 0
// CHECK-NEXT:   %12 = insertvalue { ptr, ptr } %11, ptr %7, 1
// CHECK-NEXT:   %13 = extractvalue { ptr, ptr } %12, 1
// CHECK-NEXT:   %14 = extractvalue { ptr, ptr } %12, 0
// CHECK-NEXT:   %15 = call { i64, %"{{.*}}/runtime/internal/runtime.iface" } %14(ptr %13, %"{{.*}}/runtime/internal/runtime.Slice" %1)
// CHECK-NEXT:   %16 = extractvalue { i64, %"{{.*}}/runtime/internal/runtime.iface" } %15, 0
// CHECK-NEXT:   %17 = extractvalue { i64, %"{{.*}}/runtime/internal/runtime.iface" } %15, 1
// CHECK-NEXT:   %18 = insertvalue { i64, %"{{.*}}/runtime/internal/runtime.iface" } undef, i64 %16, 0
// CHECK-NEXT:   %19 = insertvalue { i64, %"{{.*}}/runtime/internal/runtime.iface" } %18, %"{{.*}}/runtime/internal/runtime.iface" %17, 1
// CHECK-NEXT:   ret { i64, %"{{.*}}/runtime/internal/runtime.iface" } %19
// CHECK-NEXT: }

// CHECK-LABEL: define { i64, %"{{.*}}/runtime/internal/runtime.iface" } @"main.(*nopCloserWriterTo).WriteTo"(ptr %0, %"{{.*}}/runtime/internal/runtime.iface" %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = icmp eq ptr %0, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PanicWrapNilPointer"(i1 %2, %"{{.*}}/runtime/internal/runtime.String" { ptr @55, i64 58 }, %"{{.*}}/runtime/internal/runtime.String" { ptr @2, i64 7 })
// CHECK-NEXT:   %3 = load %main.nopCloserWriterTo, ptr %0, align 8
// CHECK-NEXT:   %4 = call { i64, %"{{.*}}/runtime/internal/runtime.iface" } @main.nopCloserWriterTo.WriteTo(%main.nopCloserWriterTo %3, %"{{.*}}/runtime/internal/runtime.iface" %1)
// CHECK-NEXT:   %5 = extractvalue { i64, %"{{.*}}/runtime/internal/runtime.iface" } %4, 0
// CHECK-NEXT:   %6 = extractvalue { i64, %"{{.*}}/runtime/internal/runtime.iface" } %4, 1
// CHECK-NEXT:   %7 = insertvalue { i64, %"{{.*}}/runtime/internal/runtime.iface" } undef, i64 %5, 0
// CHECK-NEXT:   %8 = insertvalue { i64, %"{{.*}}/runtime/internal/runtime.iface" } %7, %"{{.*}}/runtime/internal/runtime.iface" %6, 1
// CHECK-NEXT:   ret { i64, %"{{.*}}/runtime/internal/runtime.iface" } %8
// CHECK-NEXT: }

// CHECK-LABEL: define i64 @"main.(*stringReader).Len"(ptr %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = getelementptr inbounds %main.stringReader, ptr %0, i32 0, i32 1
// CHECK-NEXT:   %2 = icmp eq ptr %0, null
// CHECK-NEXT:   br i1 %2, label %5, label %6
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %11
// CHECK-NEXT:   ret i64 0
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %11
// CHECK-NEXT:   %3 = getelementptr inbounds %main.stringReader, ptr %0, i32 0, i32 0
// CHECK-NEXT:   %4 = icmp eq ptr %0, null
// CHECK-NEXT:   br i1 %4, label %15, label %16
// CHECK-EMPTY:
// CHECK-NEXT: 5:                                                ; preds = %_llgo_0
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 true)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: 6:                                                ; preds = %_llgo_0
// CHECK-NEXT:   %7 = load i64, ptr %1, align 8
// CHECK-NEXT:   %8 = getelementptr inbounds %main.stringReader, ptr %0, i32 0, i32 0
// CHECK-NEXT:   %9 = icmp eq ptr %0, null
// CHECK-NEXT:   br i1 %9, label %10, label %11
// CHECK-EMPTY:
// CHECK-NEXT: 10:                                               ; preds = %6
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 true)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: 11:                                               ; preds = %6
// CHECK-NEXT:   %12 = load %"{{.*}}/runtime/internal/runtime.String", ptr %8, align 8
// CHECK-NEXT:   %13 = extractvalue %"{{.*}}/runtime/internal/runtime.String" %12, 1
// CHECK-NEXT:   %14 = icmp sge i64 %7, %13
// CHECK-NEXT:   br i1 %14, label %_llgo_1, label %_llgo_2
// CHECK-EMPTY:
// CHECK-NEXT: 15:                                               ; preds = %_llgo_2
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 true)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: 16:                                               ; preds = %_llgo_2
// CHECK-NEXT:   %17 = load %"{{.*}}/runtime/internal/runtime.String", ptr %3, align 8
// CHECK-NEXT:   %18 = extractvalue %"{{.*}}/runtime/internal/runtime.String" %17, 1
// CHECK-NEXT:   %19 = getelementptr inbounds %main.stringReader, ptr %0, i32 0, i32 1
// CHECK-NEXT:   %20 = icmp eq ptr %0, null
// CHECK-NEXT:   br i1 %20, label %21, label %22
// CHECK-EMPTY:
// CHECK-NEXT: 21:                                               ; preds = %16
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 true)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: 22:                                               ; preds = %16
// CHECK-NEXT:   %23 = load i64, ptr %19, align 8
// CHECK-NEXT:   %24 = sub i64 %18, %23
// CHECK-NEXT:   ret i64 %24
// CHECK-NEXT: }

// CHECK-LABEL: define { i64, %"{{.*}}/runtime/internal/runtime.iface" } @"main.(*stringReader).Read"(ptr %0, %"{{.*}}/runtime/internal/runtime.Slice" %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = getelementptr inbounds %main.stringReader, ptr %0, i32 0, i32 1
// CHECK-NEXT:   %3 = icmp eq ptr %0, null
// CHECK-NEXT:   br i1 %3, label %9, label %10
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %15
// CHECK-NEXT:   %4 = load %"{{.*}}/runtime/internal/runtime.iface", ptr @main.EOF, align 8
// CHECK-NEXT:   %5 = insertvalue { i64, %"{{.*}}/runtime/internal/runtime.iface" } { i64 0, %"{{.*}}/runtime/internal/runtime.iface" undef }, %"{{.*}}/runtime/internal/runtime.iface" %4, 1
// CHECK-NEXT:   ret { i64, %"{{.*}}/runtime/internal/runtime.iface" } %5
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %15
// CHECK-NEXT:   %6 = getelementptr inbounds %main.stringReader, ptr %0, i32 0, i32 2
// CHECK-NEXT:   store i64 -1, ptr %6, align 8
// CHECK-NEXT:   %7 = getelementptr inbounds %main.stringReader, ptr %0, i32 0, i32 0
// CHECK-NEXT:   %8 = icmp eq ptr %0, null
// CHECK-NEXT:   br i1 %8, label %19, label %20
// CHECK-EMPTY:
// CHECK-NEXT: 9:                                                ; preds = %_llgo_0
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 true)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: 10:                                               ; preds = %_llgo_0
// CHECK-NEXT:   %11 = load i64, ptr %2, align 8
// CHECK-NEXT:   %12 = getelementptr inbounds %main.stringReader, ptr %0, i32 0, i32 0
// CHECK-NEXT:   %13 = icmp eq ptr %0, null
// CHECK-NEXT:   br i1 %13, label %14, label %15
// CHECK-EMPTY:
// CHECK-NEXT: 14:                                               ; preds = %10
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 true)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: 15:                                               ; preds = %10
// CHECK-NEXT:   %16 = load %"{{.*}}/runtime/internal/runtime.String", ptr %12, align 8
// CHECK-NEXT:   %17 = extractvalue %"{{.*}}/runtime/internal/runtime.String" %16, 1
// CHECK-NEXT:   %18 = icmp sge i64 %11, %17
// CHECK-NEXT:   br i1 %18, label %_llgo_1, label %_llgo_2
// CHECK-EMPTY:
// CHECK-NEXT: 19:                                               ; preds = %_llgo_2
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 true)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: 20:                                               ; preds = %_llgo_2
// CHECK-NEXT:   %21 = load %"{{.*}}/runtime/internal/runtime.String", ptr %7, align 8
// CHECK-NEXT:   %22 = getelementptr inbounds %main.stringReader, ptr %0, i32 0, i32 1
// CHECK-NEXT:   %23 = icmp eq ptr %0, null
// CHECK-NEXT:   br i1 %23, label %24, label %25
// CHECK-EMPTY:
// CHECK-NEXT: 24:                                               ; preds = %20
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 true)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: 25:                                               ; preds = %20
// CHECK-NEXT:   %26 = load i64, ptr %22, align 8
// CHECK-NEXT:   %27 = extractvalue %"{{.*}}/runtime/internal/runtime.String" %21, 1
// CHECK-NEXT:   %28 = call %"{{.*}}/runtime/internal/runtime.String" @"{{.*}}/runtime/internal/runtime.StringSlice2"(%"{{.*}}/runtime/internal/runtime.String" %21, i64 %26, i64 %27, i1 true, i1 true)
// CHECK-NEXT:   %29 = extractvalue %"{{.*}}/runtime/internal/runtime.String" %28, 0
// CHECK-NEXT:   %30 = extractvalue %"{{.*}}/runtime/internal/runtime.String" %28, 1
// CHECK-NEXT:   %31 = call i64 @"{{.*}}/runtime/internal/runtime.SliceCopy"(%"{{.*}}/runtime/internal/runtime.Slice" %1, ptr %29, i64 %30, i64 1)
// CHECK-NEXT:   %32 = getelementptr inbounds %main.stringReader, ptr %0, i32 0, i32 1
// CHECK-NEXT:   %33 = icmp eq ptr %0, null
// CHECK-NEXT:   br i1 %33, label %34, label %35
// CHECK-EMPTY:
// CHECK-NEXT: 34:                                               ; preds = %25
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 true)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: 35:                                               ; preds = %25
// CHECK-NEXT:   %36 = load i64, ptr %32, align 8
// CHECK-NEXT:   %37 = add i64 %36, %31
// CHECK-NEXT:   %38 = getelementptr inbounds %main.stringReader, ptr %0, i32 0, i32 1
// CHECK-NEXT:   store i64 %37, ptr %38, align 8
// CHECK-NEXT:   %39 = insertvalue { i64, %"{{.*}}/runtime/internal/runtime.iface" } undef, i64 %31, 0
// CHECK-NEXT:   %40 = insertvalue { i64, %"{{.*}}/runtime/internal/runtime.iface" } %39, %"{{.*}}/runtime/internal/runtime.iface" zeroinitializer, 1
// CHECK-NEXT:   ret { i64, %"{{.*}}/runtime/internal/runtime.iface" } %40
// CHECK-NEXT: }

// CHECK-LABEL: define { i64, %"{{.*}}/runtime/internal/runtime.iface" } @"main.(*stringReader).ReadAt"(ptr %0, %"{{.*}}/runtime/internal/runtime.Slice" %1, i64 %2){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %3 = icmp slt i64 %2, 0
// CHECK-NEXT:   br i1 %3, label %_llgo_1, label %_llgo_2
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_0
// CHECK-NEXT:   %4 = call %"{{.*}}/runtime/internal/runtime.iface" @main.newError(%"{{.*}}/runtime/internal/runtime.String" { ptr @56, i64 37 })
// CHECK-NEXT:   %5 = insertvalue { i64, %"{{.*}}/runtime/internal/runtime.iface" } { i64 0, %"{{.*}}/runtime/internal/runtime.iface" undef }, %"{{.*}}/runtime/internal/runtime.iface" %4, 1
// CHECK-NEXT:   ret { i64, %"{{.*}}/runtime/internal/runtime.iface" } %5
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_0
// CHECK-NEXT:   %6 = getelementptr inbounds %main.stringReader, ptr %0, i32 0, i32 0
// CHECK-NEXT:   %7 = icmp eq ptr %0, null
// CHECK-NEXT:   br i1 %7, label %16, label %17
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_3:                                          ; preds = %17
// CHECK-NEXT:   %8 = load %"{{.*}}/runtime/internal/runtime.iface", ptr @main.EOF, align 8
// CHECK-NEXT:   %9 = insertvalue { i64, %"{{.*}}/runtime/internal/runtime.iface" } { i64 0, %"{{.*}}/runtime/internal/runtime.iface" undef }, %"{{.*}}/runtime/internal/runtime.iface" %8, 1
// CHECK-NEXT:   ret { i64, %"{{.*}}/runtime/internal/runtime.iface" } %9
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_4:                                          ; preds = %17
// CHECK-NEXT:   %10 = getelementptr inbounds %main.stringReader, ptr %0, i32 0, i32 0
// CHECK-NEXT:   %11 = icmp eq ptr %0, null
// CHECK-NEXT:   br i1 %11, label %21, label %22
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_5:                                          ; preds = %22
// CHECK-NEXT:   %12 = load %"{{.*}}/runtime/internal/runtime.iface", ptr @main.EOF, align 8
// CHECK-NEXT:   br label %_llgo_6
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_6:                                          ; preds = %_llgo_5, %22
// CHECK-NEXT:   %13 = phi %"{{.*}}/runtime/internal/runtime.iface" [ zeroinitializer, %22 ], [ %12, %_llgo_5 ]
// CHECK-NEXT:   %14 = insertvalue { i64, %"{{.*}}/runtime/internal/runtime.iface" } undef, i64 %28, 0
// CHECK-NEXT:   %15 = insertvalue { i64, %"{{.*}}/runtime/internal/runtime.iface" } %14, %"{{.*}}/runtime/internal/runtime.iface" %13, 1
// CHECK-NEXT:   ret { i64, %"{{.*}}/runtime/internal/runtime.iface" } %15
// CHECK-EMPTY:
// CHECK-NEXT: 16:                                               ; preds = %_llgo_2
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 true)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: 17:                                               ; preds = %_llgo_2
// CHECK-NEXT:   %18 = load %"{{.*}}/runtime/internal/runtime.String", ptr %6, align 8
// CHECK-NEXT:   %19 = extractvalue %"{{.*}}/runtime/internal/runtime.String" %18, 1
// CHECK-NEXT:   %20 = icmp sge i64 %2, %19
// CHECK-NEXT:   br i1 %20, label %_llgo_3, label %_llgo_4
// CHECK-EMPTY:
// CHECK-NEXT: 21:                                               ; preds = %_llgo_4
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 true)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: 22:                                               ; preds = %_llgo_4
// CHECK-NEXT:   %23 = load %"{{.*}}/runtime/internal/runtime.String", ptr %10, align 8
// CHECK-NEXT:   %24 = extractvalue %"{{.*}}/runtime/internal/runtime.String" %23, 1
// CHECK-NEXT:   %25 = call %"{{.*}}/runtime/internal/runtime.String" @"{{.*}}/runtime/internal/runtime.StringSlice2"(%"{{.*}}/runtime/internal/runtime.String" %23, i64 %2, i64 %24, i1 true, i1 true)
// CHECK-NEXT:   %26 = extractvalue %"{{.*}}/runtime/internal/runtime.String" %25, 0
// CHECK-NEXT:   %27 = extractvalue %"{{.*}}/runtime/internal/runtime.String" %25, 1
// CHECK-NEXT:   %28 = call i64 @"{{.*}}/runtime/internal/runtime.SliceCopy"(%"{{.*}}/runtime/internal/runtime.Slice" %1, ptr %26, i64 %27, i64 1)
// CHECK-NEXT:   %29 = extractvalue %"{{.*}}/runtime/internal/runtime.Slice" %1, 1
// CHECK-NEXT:   %30 = icmp slt i64 %28, %29
// CHECK-NEXT:   br i1 %30, label %_llgo_5, label %_llgo_6
// CHECK-NEXT: }

// CHECK-LABEL: define { i8, %"{{.*}}/runtime/internal/runtime.iface" } @"main.(*stringReader).ReadByte"(ptr %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = getelementptr inbounds %main.stringReader, ptr %0, i32 0, i32 2
// CHECK-NEXT:   store i64 -1, ptr %1, align 8
// CHECK-NEXT:   %2 = getelementptr inbounds %main.stringReader, ptr %0, i32 0, i32 1
// CHECK-NEXT:   %3 = icmp eq ptr %0, null
// CHECK-NEXT:   br i1 %3, label %8, label %9
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %14
// CHECK-NEXT:   %4 = load %"{{.*}}/runtime/internal/runtime.iface", ptr @main.EOF, align 8
// CHECK-NEXT:   %5 = insertvalue { i8, %"{{.*}}/runtime/internal/runtime.iface" } { i8 0, %"{{.*}}/runtime/internal/runtime.iface" undef }, %"{{.*}}/runtime/internal/runtime.iface" %4, 1
// CHECK-NEXT:   ret { i8, %"{{.*}}/runtime/internal/runtime.iface" } %5
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %14
// CHECK-NEXT:   %6 = getelementptr inbounds %main.stringReader, ptr %0, i32 0, i32 1
// CHECK-NEXT:   %7 = icmp eq ptr %0, null
// CHECK-NEXT:   br i1 %7, label %18, label %19
// CHECK-EMPTY:
// CHECK-NEXT: 8:                                                ; preds = %_llgo_0
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 true)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: 9:                                                ; preds = %_llgo_0
// CHECK-NEXT:   %10 = load i64, ptr %2, align 8
// CHECK-NEXT:   %11 = getelementptr inbounds %main.stringReader, ptr %0, i32 0, i32 0
// CHECK-NEXT:   %12 = icmp eq ptr %0, null
// CHECK-NEXT:   br i1 %12, label %13, label %14
// CHECK-EMPTY:
// CHECK-NEXT: 13:                                               ; preds = %9
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 true)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: 14:                                               ; preds = %9
// CHECK-NEXT:   %15 = load %"{{.*}}/runtime/internal/runtime.String", ptr %11, align 8
// CHECK-NEXT:   %16 = extractvalue %"{{.*}}/runtime/internal/runtime.String" %15, 1
// CHECK-NEXT:   %17 = icmp sge i64 %10, %16
// CHECK-NEXT:   br i1 %17, label %_llgo_1, label %_llgo_2
// CHECK-EMPTY:
// CHECK-NEXT: 18:                                               ; preds = %_llgo_2
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 true)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: 19:                                               ; preds = %_llgo_2
// CHECK-NEXT:   %20 = load i64, ptr %6, align 8
// CHECK-NEXT:   %21 = getelementptr inbounds %main.stringReader, ptr %0, i32 0, i32 0
// CHECK-NEXT:   %22 = icmp eq ptr %0, null
// CHECK-NEXT:   br i1 %22, label %23, label %24
// CHECK-EMPTY:
// CHECK-NEXT: 23:                                               ; preds = %19
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 true)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: 24:                                               ; preds = %19
// CHECK-NEXT:   %25 = load %"{{.*}}/runtime/internal/runtime.String", ptr %21, align 8
// CHECK-NEXT:   %26 = extractvalue %"{{.*}}/runtime/internal/runtime.String" %25, 0
// CHECK-NEXT:   %27 = extractvalue %"{{.*}}/runtime/internal/runtime.String" %25, 1
// CHECK-NEXT:   %28 = icmp slt i64 %20, 0
// CHECK-NEXT:   %29 = icmp uge i64 %20, %27
// CHECK-NEXT:   %30 = or i1 %29, %28
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.CheckIndexRange"(i1 %30, i64 %20, i1 true, i64 %27)
// CHECK-NEXT:   %31 = getelementptr inbounds i8, ptr %26, i64 %20
// CHECK-NEXT:   %32 = load i8, ptr %31, align 1
// CHECK-NEXT:   %33 = getelementptr inbounds %main.stringReader, ptr %0, i32 0, i32 1
// CHECK-NEXT:   %34 = icmp eq ptr %0, null
// CHECK-NEXT:   br i1 %34, label %35, label %36
// CHECK-EMPTY:
// CHECK-NEXT: 35:                                               ; preds = %24
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 true)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: 36:                                               ; preds = %24
// CHECK-NEXT:   %37 = load i64, ptr %33, align 8
// CHECK-NEXT:   %38 = add i64 %37, 1
// CHECK-NEXT:   %39 = getelementptr inbounds %main.stringReader, ptr %0, i32 0, i32 1
// CHECK-NEXT:   store i64 %38, ptr %39, align 8
// CHECK-NEXT:   %40 = insertvalue { i8, %"{{.*}}/runtime/internal/runtime.iface" } undef, i8 %32, 0
// CHECK-NEXT:   %41 = insertvalue { i8, %"{{.*}}/runtime/internal/runtime.iface" } %40, %"{{.*}}/runtime/internal/runtime.iface" zeroinitializer, 1
// CHECK-NEXT:   ret { i8, %"{{.*}}/runtime/internal/runtime.iface" } %41
// CHECK-NEXT: }

// CHECK-LABEL: define { i32, i64, %"{{.*}}/runtime/internal/runtime.iface" } @"main.(*stringReader).ReadRune"(ptr %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = getelementptr inbounds %main.stringReader, ptr %0, i32 0, i32 1
// CHECK-NEXT:   %2 = icmp eq ptr %0, null
// CHECK-NEXT:   br i1 %2, label %12, label %13
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %18
// CHECK-NEXT:   %3 = getelementptr inbounds %main.stringReader, ptr %0, i32 0, i32 2
// CHECK-NEXT:   store i64 -1, ptr %3, align 8
// CHECK-NEXT:   %4 = load %"{{.*}}/runtime/internal/runtime.iface", ptr @main.EOF, align 8
// CHECK-NEXT:   %5 = insertvalue { i32, i64, %"{{.*}}/runtime/internal/runtime.iface" } { i32 0, i64 0, %"{{.*}}/runtime/internal/runtime.iface" undef }, %"{{.*}}/runtime/internal/runtime.iface" %4, 2
// CHECK-NEXT:   ret { i32, i64, %"{{.*}}/runtime/internal/runtime.iface" } %5
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %18
// CHECK-NEXT:   %6 = getelementptr inbounds %main.stringReader, ptr %0, i32 0, i32 1
// CHECK-NEXT:   %7 = icmp eq ptr %0, null
// CHECK-NEXT:   br i1 %7, label %22, label %23
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_3:                                          ; preds = %34
// CHECK-NEXT:   %8 = getelementptr inbounds %main.stringReader, ptr %0, i32 0, i32 1
// CHECK-NEXT:   %9 = icmp eq ptr %0, null
// CHECK-NEXT:   br i1 %9, label %44, label %45
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_4:                                          ; preds = %34
// CHECK-NEXT:   %10 = getelementptr inbounds %main.stringReader, ptr %0, i32 0, i32 0
// CHECK-NEXT:   %11 = icmp eq ptr %0, null
// CHECK-NEXT:   br i1 %11, label %53, label %54
// CHECK-EMPTY:
// CHECK-NEXT: 12:                                               ; preds = %_llgo_0
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 true)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: 13:                                               ; preds = %_llgo_0
// CHECK-NEXT:   %14 = load i64, ptr %1, align 8
// CHECK-NEXT:   %15 = getelementptr inbounds %main.stringReader, ptr %0, i32 0, i32 0
// CHECK-NEXT:   %16 = icmp eq ptr %0, null
// CHECK-NEXT:   br i1 %16, label %17, label %18
// CHECK-EMPTY:
// CHECK-NEXT: 17:                                               ; preds = %13
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 true)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: 18:                                               ; preds = %13
// CHECK-NEXT:   %19 = load %"{{.*}}/runtime/internal/runtime.String", ptr %15, align 8
// CHECK-NEXT:   %20 = extractvalue %"{{.*}}/runtime/internal/runtime.String" %19, 1
// CHECK-NEXT:   %21 = icmp sge i64 %14, %20
// CHECK-NEXT:   br i1 %21, label %_llgo_1, label %_llgo_2
// CHECK-EMPTY:
// CHECK-NEXT: 22:                                               ; preds = %_llgo_2
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 true)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: 23:                                               ; preds = %_llgo_2
// CHECK-NEXT:   %24 = load i64, ptr %6, align 8
// CHECK-NEXT:   %25 = getelementptr inbounds %main.stringReader, ptr %0, i32 0, i32 2
// CHECK-NEXT:   store i64 %24, ptr %25, align 8
// CHECK-NEXT:   %26 = getelementptr inbounds %main.stringReader, ptr %0, i32 0, i32 1
// CHECK-NEXT:   %27 = icmp eq ptr %0, null
// CHECK-NEXT:   br i1 %27, label %28, label %29
// CHECK-EMPTY:
// CHECK-NEXT: 28:                                               ; preds = %23
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 true)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: 29:                                               ; preds = %23
// CHECK-NEXT:   %30 = load i64, ptr %26, align 8
// CHECK-NEXT:   %31 = getelementptr inbounds %main.stringReader, ptr %0, i32 0, i32 0
// CHECK-NEXT:   %32 = icmp eq ptr %0, null
// CHECK-NEXT:   br i1 %32, label %33, label %34
// CHECK-EMPTY:
// CHECK-NEXT: 33:                                               ; preds = %29
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 true)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: 34:                                               ; preds = %29
// CHECK-NEXT:   %35 = load %"{{.*}}/runtime/internal/runtime.String", ptr %31, align 8
// CHECK-NEXT:   %36 = extractvalue %"{{.*}}/runtime/internal/runtime.String" %35, 0
// CHECK-NEXT:   %37 = extractvalue %"{{.*}}/runtime/internal/runtime.String" %35, 1
// CHECK-NEXT:   %38 = icmp slt i64 %30, 0
// CHECK-NEXT:   %39 = icmp uge i64 %30, %37
// CHECK-NEXT:   %40 = or i1 %39, %38
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.CheckIndexRange"(i1 %40, i64 %30, i1 true, i64 %37)
// CHECK-NEXT:   %41 = getelementptr inbounds i8, ptr %36, i64 %30
// CHECK-NEXT:   %42 = load i8, ptr %41, align 1
// CHECK-NEXT:   %43 = icmp ult i8 %42, -128
// CHECK-NEXT:   br i1 %43, label %_llgo_3, label %_llgo_4
// CHECK-EMPTY:
// CHECK-NEXT: 44:                                               ; preds = %_llgo_3
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 true)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: 45:                                               ; preds = %_llgo_3
// CHECK-NEXT:   %46 = load i64, ptr %8, align 8
// CHECK-NEXT:   %47 = add i64 %46, 1
// CHECK-NEXT:   %48 = getelementptr inbounds %main.stringReader, ptr %0, i32 0, i32 1
// CHECK-NEXT:   store i64 %47, ptr %48, align 8
// CHECK-NEXT:   %49 = zext i8 %42 to i32
// CHECK-NEXT:   %50 = insertvalue { i32, i64, %"{{.*}}/runtime/internal/runtime.iface" } undef, i32 %49, 0
// CHECK-NEXT:   %51 = insertvalue { i32, i64, %"{{.*}}/runtime/internal/runtime.iface" } %50, i64 1, 1
// CHECK-NEXT:   %52 = insertvalue { i32, i64, %"{{.*}}/runtime/internal/runtime.iface" } %51, %"{{.*}}/runtime/internal/runtime.iface" zeroinitializer, 2
// CHECK-NEXT:   ret { i32, i64, %"{{.*}}/runtime/internal/runtime.iface" } %52
// CHECK-EMPTY:
// CHECK-NEXT: 53:                                               ; preds = %_llgo_4
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 true)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: 54:                                               ; preds = %_llgo_4
// CHECK-NEXT:   %55 = load %"{{.*}}/runtime/internal/runtime.String", ptr %10, align 8
// CHECK-NEXT:   %56 = getelementptr inbounds %main.stringReader, ptr %0, i32 0, i32 1
// CHECK-NEXT:   %57 = icmp eq ptr %0, null
// CHECK-NEXT:   br i1 %57, label %58, label %59
// CHECK-EMPTY:
// CHECK-NEXT: 58:                                               ; preds = %54
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 true)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: 59:                                               ; preds = %54
// CHECK-NEXT:   %60 = load i64, ptr %56, align 8
// CHECK-NEXT:   %61 = extractvalue %"{{.*}}/runtime/internal/runtime.String" %55, 1
// CHECK-NEXT:   %62 = call %"{{.*}}/runtime/internal/runtime.String" @"{{.*}}/runtime/internal/runtime.StringSlice2"(%"{{.*}}/runtime/internal/runtime.String" %55, i64 %60, i64 %61, i1 true, i1 true)
// CHECK-NEXT:   %63 = call { i32, i64 } @"unicode/utf8.DecodeRuneInString"(%"{{.*}}/runtime/internal/runtime.String" %62)
// CHECK-NEXT:   %64 = extractvalue { i32, i64 } %63, 0
// CHECK-NEXT:   %65 = extractvalue { i32, i64 } %63, 1
// CHECK-NEXT:   %66 = getelementptr inbounds %main.stringReader, ptr %0, i32 0, i32 1
// CHECK-NEXT:   %67 = icmp eq ptr %0, null
// CHECK-NEXT:   br i1 %67, label %68, label %69
// CHECK-EMPTY:
// CHECK-NEXT: 68:                                               ; preds = %59
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 true)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: 69:                                               ; preds = %59
// CHECK-NEXT:   %70 = load i64, ptr %66, align 8
// CHECK-NEXT:   %71 = add i64 %70, %65
// CHECK-NEXT:   %72 = getelementptr inbounds %main.stringReader, ptr %0, i32 0, i32 1
// CHECK-NEXT:   store i64 %71, ptr %72, align 8
// CHECK-NEXT:   %73 = insertvalue { i32, i64, %"{{.*}}/runtime/internal/runtime.iface" } undef, i32 %64, 0
// CHECK-NEXT:   %74 = insertvalue { i32, i64, %"{{.*}}/runtime/internal/runtime.iface" } %73, i64 %65, 1
// CHECK-NEXT:   %75 = insertvalue { i32, i64, %"{{.*}}/runtime/internal/runtime.iface" } %74, %"{{.*}}/runtime/internal/runtime.iface" zeroinitializer, 2
// CHECK-NEXT:   ret { i32, i64, %"{{.*}}/runtime/internal/runtime.iface" } %75
// CHECK-NEXT: }

// CHECK-LABEL: define { i64, %"{{.*}}/runtime/internal/runtime.iface" } @"main.(*stringReader).Seek"(ptr %0, i64 %1, i64 %2){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %3 = getelementptr inbounds %main.stringReader, ptr %0, i32 0, i32 2
// CHECK-NEXT:   store i64 -1, ptr %3, align 8
// CHECK-NEXT:   %4 = icmp eq i64 %2, 0
// CHECK-NEXT:   br i1 %4, label %_llgo_2, label %_llgo_4
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %25, %21, %_llgo_2
// CHECK-NEXT:   %5 = phi i64 [ %1, %_llgo_2 ], [ %23, %21 ], [ %28, %25 ]
// CHECK-NEXT:   %6 = icmp slt i64 %5, 0
// CHECK-NEXT:   br i1 %6, label %_llgo_8, label %_llgo_9
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_0
// CHECK-NEXT:   br label %_llgo_1
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_3:                                          ; preds = %_llgo_4
// CHECK-NEXT:   %7 = getelementptr inbounds %main.stringReader, ptr %0, i32 0, i32 1
// CHECK-NEXT:   %8 = icmp eq ptr %0, null
// CHECK-NEXT:   br i1 %8, label %20, label %21
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_4:                                          ; preds = %_llgo_0
// CHECK-NEXT:   %9 = icmp eq i64 %2, 1
// CHECK-NEXT:   br i1 %9, label %_llgo_3, label %_llgo_6
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_5:                                          ; preds = %_llgo_6
// CHECK-NEXT:   %10 = getelementptr inbounds %main.stringReader, ptr %0, i32 0, i32 0
// CHECK-NEXT:   %11 = icmp eq ptr %0, null
// CHECK-NEXT:   br i1 %11, label %24, label %25
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_6:                                          ; preds = %_llgo_4
// CHECK-NEXT:   %12 = icmp eq i64 %2, 2
// CHECK-NEXT:   br i1 %12, label %_llgo_5, label %_llgo_7
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_7:                                          ; preds = %_llgo_6
// CHECK-NEXT:   %13 = call %"{{.*}}/runtime/internal/runtime.iface" @main.newError(%"{{.*}}/runtime/internal/runtime.String" { ptr @57, i64 34 })
// CHECK-NEXT:   %14 = insertvalue { i64, %"{{.*}}/runtime/internal/runtime.iface" } { i64 0, %"{{.*}}/runtime/internal/runtime.iface" undef }, %"{{.*}}/runtime/internal/runtime.iface" %13, 1
// CHECK-NEXT:   ret { i64, %"{{.*}}/runtime/internal/runtime.iface" } %14
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_8:                                          ; preds = %_llgo_1
// CHECK-NEXT:   %15 = call %"{{.*}}/runtime/internal/runtime.iface" @main.newError(%"{{.*}}/runtime/internal/runtime.String" { ptr @58, i64 37 })
// CHECK-NEXT:   %16 = insertvalue { i64, %"{{.*}}/runtime/internal/runtime.iface" } { i64 0, %"{{.*}}/runtime/internal/runtime.iface" undef }, %"{{.*}}/runtime/internal/runtime.iface" %15, 1
// CHECK-NEXT:   ret { i64, %"{{.*}}/runtime/internal/runtime.iface" } %16
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_9:                                          ; preds = %_llgo_1
// CHECK-NEXT:   %17 = getelementptr inbounds %main.stringReader, ptr %0, i32 0, i32 1
// CHECK-NEXT:   store i64 %5, ptr %17, align 8
// CHECK-NEXT:   %18 = insertvalue { i64, %"{{.*}}/runtime/internal/runtime.iface" } undef, i64 %5, 0
// CHECK-NEXT:   %19 = insertvalue { i64, %"{{.*}}/runtime/internal/runtime.iface" } %18, %"{{.*}}/runtime/internal/runtime.iface" zeroinitializer, 1
// CHECK-NEXT:   ret { i64, %"{{.*}}/runtime/internal/runtime.iface" } %19
// CHECK-EMPTY:
// CHECK-NEXT: 20:                                               ; preds = %_llgo_3
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 true)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: 21:                                               ; preds = %_llgo_3
// CHECK-NEXT:   %22 = load i64, ptr %7, align 8
// CHECK-NEXT:   %23 = add i64 %22, %1
// CHECK-NEXT:   br label %_llgo_1
// CHECK-EMPTY:
// CHECK-NEXT: 24:                                               ; preds = %_llgo_5
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 true)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: 25:                                               ; preds = %_llgo_5
// CHECK-NEXT:   %26 = load %"{{.*}}/runtime/internal/runtime.String", ptr %10, align 8
// CHECK-NEXT:   %27 = extractvalue %"{{.*}}/runtime/internal/runtime.String" %26, 1
// CHECK-NEXT:   %28 = add i64 %27, %1
// CHECK-NEXT:   br label %_llgo_1
// CHECK-NEXT: }

// CHECK-LABEL: define i64 @"main.(*stringReader).Size"(ptr %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = getelementptr inbounds %main.stringReader, ptr %0, i32 0, i32 0
// CHECK-NEXT:   %2 = icmp eq ptr %0, null
// CHECK-NEXT:   br i1 %2, label %3, label %4
// CHECK-EMPTY:
// CHECK-NEXT: 3:                                                ; preds = %_llgo_0
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 true)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: 4:                                                ; preds = %_llgo_0
// CHECK-NEXT:   %5 = load %"{{.*}}/runtime/internal/runtime.String", ptr %1, align 8
// CHECK-NEXT:   %6 = extractvalue %"{{.*}}/runtime/internal/runtime.String" %5, 1
// CHECK-NEXT:   ret i64 %6
// CHECK-NEXT: }

// CHECK-LABEL: define %"{{.*}}/runtime/internal/runtime.iface" @"main.(*stringReader).UnreadByte"(ptr %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = getelementptr inbounds %main.stringReader, ptr %0, i32 0, i32 1
// CHECK-NEXT:   %2 = icmp eq ptr %0, null
// CHECK-NEXT:   br i1 %2, label %7, label %8
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %8
// CHECK-NEXT:   %3 = call %"{{.*}}/runtime/internal/runtime.iface" @main.newError(%"{{.*}}/runtime/internal/runtime.String" { ptr @59, i64 48 })
// CHECK-NEXT:   ret %"{{.*}}/runtime/internal/runtime.iface" %3
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %8
// CHECK-NEXT:   %4 = getelementptr inbounds %main.stringReader, ptr %0, i32 0, i32 2
// CHECK-NEXT:   store i64 -1, ptr %4, align 8
// CHECK-NEXT:   %5 = getelementptr inbounds %main.stringReader, ptr %0, i32 0, i32 1
// CHECK-NEXT:   %6 = icmp eq ptr %0, null
// CHECK-NEXT:   br i1 %6, label %11, label %12
// CHECK-EMPTY:
// CHECK-NEXT: 7:                                                ; preds = %_llgo_0
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 true)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: 8:                                                ; preds = %_llgo_0
// CHECK-NEXT:   %9 = load i64, ptr %1, align 8
// CHECK-NEXT:   %10 = icmp sle i64 %9, 0
// CHECK-NEXT:   br i1 %10, label %_llgo_1, label %_llgo_2
// CHECK-EMPTY:
// CHECK-NEXT: 11:                                               ; preds = %_llgo_2
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 true)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: 12:                                               ; preds = %_llgo_2
// CHECK-NEXT:   %13 = load i64, ptr %5, align 8
// CHECK-NEXT:   %14 = sub i64 %13, 1
// CHECK-NEXT:   %15 = getelementptr inbounds %main.stringReader, ptr %0, i32 0, i32 1
// CHECK-NEXT:   store i64 %14, ptr %15, align 8
// CHECK-NEXT:   ret %"{{.*}}/runtime/internal/runtime.iface" zeroinitializer
// CHECK-NEXT: }

// CHECK-LABEL: define %"{{.*}}/runtime/internal/runtime.iface" @"main.(*stringReader).UnreadRune"(ptr %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = getelementptr inbounds %main.stringReader, ptr %0, i32 0, i32 1
// CHECK-NEXT:   %2 = icmp eq ptr %0, null
// CHECK-NEXT:   br i1 %2, label %9, label %10
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %10
// CHECK-NEXT:   %3 = call %"{{.*}}/runtime/internal/runtime.iface" @main.newError(%"{{.*}}/runtime/internal/runtime.String" { ptr @60, i64 49 })
// CHECK-NEXT:   ret %"{{.*}}/runtime/internal/runtime.iface" %3
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %10
// CHECK-NEXT:   %4 = getelementptr inbounds %main.stringReader, ptr %0, i32 0, i32 2
// CHECK-NEXT:   %5 = icmp eq ptr %0, null
// CHECK-NEXT:   br i1 %5, label %13, label %14
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_3:                                          ; preds = %14
// CHECK-NEXT:   %6 = call %"{{.*}}/runtime/internal/runtime.iface" @main.newError(%"{{.*}}/runtime/internal/runtime.String" { ptr @61, i64 62 })
// CHECK-NEXT:   ret %"{{.*}}/runtime/internal/runtime.iface" %6
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_4:                                          ; preds = %14
// CHECK-NEXT:   %7 = getelementptr inbounds %main.stringReader, ptr %0, i32 0, i32 2
// CHECK-NEXT:   %8 = icmp eq ptr %0, null
// CHECK-NEXT:   br i1 %8, label %17, label %18
// CHECK-EMPTY:
// CHECK-NEXT: 9:                                                ; preds = %_llgo_0
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 true)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: 10:                                               ; preds = %_llgo_0
// CHECK-NEXT:   %11 = load i64, ptr %1, align 8
// CHECK-NEXT:   %12 = icmp sle i64 %11, 0
// CHECK-NEXT:   br i1 %12, label %_llgo_1, label %_llgo_2
// CHECK-EMPTY:
// CHECK-NEXT: 13:                                               ; preds = %_llgo_2
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 true)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: 14:                                               ; preds = %_llgo_2
// CHECK-NEXT:   %15 = load i64, ptr %4, align 8
// CHECK-NEXT:   %16 = icmp slt i64 %15, 0
// CHECK-NEXT:   br i1 %16, label %_llgo_3, label %_llgo_4
// CHECK-EMPTY:
// CHECK-NEXT: 17:                                               ; preds = %_llgo_4
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 true)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: 18:                                               ; preds = %_llgo_4
// CHECK-NEXT:   %19 = load i64, ptr %7, align 8
// CHECK-NEXT:   %20 = getelementptr inbounds %main.stringReader, ptr %0, i32 0, i32 1
// CHECK-NEXT:   store i64 %19, ptr %20, align 8
// CHECK-NEXT:   %21 = getelementptr inbounds %main.stringReader, ptr %0, i32 0, i32 2
// CHECK-NEXT:   store i64 -1, ptr %21, align 8
// CHECK-NEXT:   ret %"{{.*}}/runtime/internal/runtime.iface" zeroinitializer
// CHECK-NEXT: }

// CHECK-LABEL: define { i64, %"{{.*}}/runtime/internal/runtime.iface" } @"main.(*stringReader).WriteTo"(ptr %0, %"{{.*}}/runtime/internal/runtime.iface" %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = getelementptr inbounds %main.stringReader, ptr %0, i32 0, i32 2
// CHECK-NEXT:   store i64 -1, ptr %2, align 8
// CHECK-NEXT:   %3 = getelementptr inbounds %main.stringReader, ptr %0, i32 0, i32 1
// CHECK-NEXT:   %4 = icmp eq ptr %0, null
// CHECK-NEXT:   br i1 %4, label %23, label %24
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %29
// CHECK-NEXT:   ret { i64, %"{{.*}}/runtime/internal/runtime.iface" } zeroinitializer
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %29
// CHECK-NEXT:   %5 = getelementptr inbounds %main.stringReader, ptr %0, i32 0, i32 0
// CHECK-NEXT:   %6 = icmp eq ptr %0, null
// CHECK-NEXT:   br i1 %6, label %33, label %34
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_3:                                          ; preds = %39
// CHECK-NEXT:   %7 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @62, i64 48 }, ptr %7, align 8
// CHECK-NEXT:   %8 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_string, ptr undef }, ptr %7, 1
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.Panic"(%"{{.*}}/runtime/internal/runtime.eface" %8)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_4:                                          ; preds = %39
// CHECK-NEXT:   %9 = getelementptr inbounds %main.stringReader, ptr %0, i32 0, i32 1
// CHECK-NEXT:   %10 = icmp eq ptr %0, null
// CHECK-NEXT:   br i1 %10, label %48, label %49
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_5:                                          ; preds = %_llgo_7
// CHECK-NEXT:   %11 = load %"{{.*}}/runtime/internal/runtime.iface", ptr @main.ErrShortWrite, align 8
// CHECK-NEXT:   br label %_llgo_6
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_6:                                          ; preds = %_llgo_5, %_llgo_7, %49
// CHECK-NEXT:   %12 = phi %"{{.*}}/runtime/internal/runtime.iface" [ %45, %49 ], [ %45, %_llgo_7 ], [ %11, %_llgo_5 ]
// CHECK-NEXT:   %13 = insertvalue { i64, %"{{.*}}/runtime/internal/runtime.iface" } undef, i64 %44, 0
// CHECK-NEXT:   %14 = insertvalue { i64, %"{{.*}}/runtime/internal/runtime.iface" } %13, %"{{.*}}/runtime/internal/runtime.iface" %12, 1
// CHECK-NEXT:   ret { i64, %"{{.*}}/runtime/internal/runtime.iface" } %14
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_7:                                          ; preds = %49
// CHECK-NEXT:   %15 = call ptr @"{{.*}}/runtime/internal/runtime.IfaceType"(%"{{.*}}/runtime/internal/runtime.iface" %45)
// CHECK-NEXT:   %16 = extractvalue %"{{.*}}/runtime/internal/runtime.iface" %45, 1
// CHECK-NEXT:   %17 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" undef, ptr %15, 0
// CHECK-NEXT:   %18 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" %17, ptr %16, 1
// CHECK-NEXT:   %19 = call ptr @"{{.*}}/runtime/internal/runtime.IfaceType"(%"{{.*}}/runtime/internal/runtime.iface" zeroinitializer)
// CHECK-NEXT:   %20 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" undef, ptr %19, 0
// CHECK-NEXT:   %21 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" %20, ptr null, 1
// CHECK-NEXT:   %22 = call i1 @"{{.*}}/runtime/internal/runtime.EfaceEqual"(%"{{.*}}/runtime/internal/runtime.eface" %18, %"{{.*}}/runtime/internal/runtime.eface" %21)
// CHECK-NEXT:   br i1 %22, label %_llgo_5, label %_llgo_6
// CHECK-EMPTY:
// CHECK-NEXT: 23:                                               ; preds = %_llgo_0
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 true)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: 24:                                               ; preds = %_llgo_0
// CHECK-NEXT:   %25 = load i64, ptr %3, align 8
// CHECK-NEXT:   %26 = getelementptr inbounds %main.stringReader, ptr %0, i32 0, i32 0
// CHECK-NEXT:   %27 = icmp eq ptr %0, null
// CHECK-NEXT:   br i1 %27, label %28, label %29
// CHECK-EMPTY:
// CHECK-NEXT: 28:                                               ; preds = %24
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 true)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: 29:                                               ; preds = %24
// CHECK-NEXT:   %30 = load %"{{.*}}/runtime/internal/runtime.String", ptr %26, align 8
// CHECK-NEXT:   %31 = extractvalue %"{{.*}}/runtime/internal/runtime.String" %30, 1
// CHECK-NEXT:   %32 = icmp sge i64 %25, %31
// CHECK-NEXT:   br i1 %32, label %_llgo_1, label %_llgo_2
// CHECK-EMPTY:
// CHECK-NEXT: 33:                                               ; preds = %_llgo_2
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 true)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: 34:                                               ; preds = %_llgo_2
// CHECK-NEXT:   %35 = load %"{{.*}}/runtime/internal/runtime.String", ptr %5, align 8
// CHECK-NEXT:   %36 = getelementptr inbounds %main.stringReader, ptr %0, i32 0, i32 1
// CHECK-NEXT:   %37 = icmp eq ptr %0, null
// CHECK-NEXT:   br i1 %37, label %38, label %39
// CHECK-EMPTY:
// CHECK-NEXT: 38:                                               ; preds = %34
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 true)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: 39:                                               ; preds = %34
// CHECK-NEXT:   %40 = load i64, ptr %36, align 8
// CHECK-NEXT:   %41 = extractvalue %"{{.*}}/runtime/internal/runtime.String" %35, 1
// CHECK-NEXT:   %42 = call %"{{.*}}/runtime/internal/runtime.String" @"{{.*}}/runtime/internal/runtime.StringSlice2"(%"{{.*}}/runtime/internal/runtime.String" %35, i64 %40, i64 %41, i1 true, i1 true)
// CHECK-NEXT:   %43 = call { i64, %"{{.*}}/runtime/internal/runtime.iface" } @main.WriteString(%"{{.*}}/runtime/internal/runtime.iface" %1, %"{{.*}}/runtime/internal/runtime.String" %42)
// CHECK-NEXT:   %44 = extractvalue { i64, %"{{.*}}/runtime/internal/runtime.iface" } %43, 0
// CHECK-NEXT:   %45 = extractvalue { i64, %"{{.*}}/runtime/internal/runtime.iface" } %43, 1
// CHECK-NEXT:   %46 = extractvalue %"{{.*}}/runtime/internal/runtime.String" %42, 1
// CHECK-NEXT:   %47 = icmp sgt i64 %44, %46
// CHECK-NEXT:   br i1 %47, label %_llgo_3, label %_llgo_4
// CHECK-EMPTY:
// CHECK-NEXT: 48:                                               ; preds = %_llgo_4
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 true)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: 49:                                               ; preds = %_llgo_4
// CHECK-NEXT:   %50 = load i64, ptr %9, align 8
// CHECK-NEXT:   %51 = add i64 %50, %44
// CHECK-NEXT:   %52 = getelementptr inbounds %main.stringReader, ptr %0, i32 0, i32 1
// CHECK-NEXT:   store i64 %51, ptr %52, align 8
// CHECK-NEXT:   %53 = extractvalue %"{{.*}}/runtime/internal/runtime.String" %42, 1
// CHECK-NEXT:   %54 = icmp ne i64 %44, %53
// CHECK-NEXT:   br i1 %54, label %_llgo_7, label %_llgo_6
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

// CHECK-LABEL: define linkonce %"{{.*}}/runtime/internal/runtime.iface" @"__llgo_stub.main.(*nopCloserWriterTo).Close"(ptr %0, ptr %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = tail call %"{{.*}}/runtime/internal/runtime.iface" @"main.(*nopCloserWriterTo).Close"(ptr %1)
// CHECK-NEXT:   ret %"{{.*}}/runtime/internal/runtime.iface" %2
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce { i64, %"{{.*}}/runtime/internal/runtime.iface" } @"__llgo_stub.main.(*nopCloserWriterTo).Read"(ptr %0, ptr %1, %"{{.*}}/runtime/internal/runtime.Slice" %2){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %3 = tail call { i64, %"{{.*}}/runtime/internal/runtime.iface" } @"main.(*nopCloserWriterTo).Read"(ptr %1, %"{{.*}}/runtime/internal/runtime.Slice" %2)
// CHECK-NEXT:   ret { i64, %"{{.*}}/runtime/internal/runtime.iface" } %3
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce { i64, %"{{.*}}/runtime/internal/runtime.iface" } @"__llgo_stub.main.(*nopCloserWriterTo).WriteTo"(ptr %0, ptr %1, %"{{.*}}/runtime/internal/runtime.iface" %2){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %3 = tail call { i64, %"{{.*}}/runtime/internal/runtime.iface" } @"main.(*nopCloserWriterTo).WriteTo"(ptr %1, %"{{.*}}/runtime/internal/runtime.iface" %2)
// CHECK-NEXT:   ret { i64, %"{{.*}}/runtime/internal/runtime.iface" } %3
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce %"{{.*}}/runtime/internal/runtime.iface" @__llgo_stub.main.nopCloserWriterTo.Close(ptr %0, %main.nopCloserWriterTo %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = tail call %"{{.*}}/runtime/internal/runtime.iface" @main.nopCloserWriterTo.Close(%main.nopCloserWriterTo %1)
// CHECK-NEXT:   ret %"{{.*}}/runtime/internal/runtime.iface" %2
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce { i64, %"{{.*}}/runtime/internal/runtime.iface" } @__llgo_stub.main.nopCloserWriterTo.Read(ptr %0, %main.nopCloserWriterTo %1, %"{{.*}}/runtime/internal/runtime.Slice" %2){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %3 = tail call { i64, %"{{.*}}/runtime/internal/runtime.iface" } @main.nopCloserWriterTo.Read(%main.nopCloserWriterTo %1, %"{{.*}}/runtime/internal/runtime.Slice" %2)
// CHECK-NEXT:   ret { i64, %"{{.*}}/runtime/internal/runtime.iface" } %3
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce { i64, %"{{.*}}/runtime/internal/runtime.iface" } @__llgo_stub.main.nopCloserWriterTo.WriteTo(ptr %0, %main.nopCloserWriterTo %1, %"{{.*}}/runtime/internal/runtime.iface" %2){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %3 = tail call { i64, %"{{.*}}/runtime/internal/runtime.iface" } @main.nopCloserWriterTo.WriteTo(%main.nopCloserWriterTo %1, %"{{.*}}/runtime/internal/runtime.iface" %2)
// CHECK-NEXT:   ret { i64, %"{{.*}}/runtime/internal/runtime.iface" } %3
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce %"{{.*}}/runtime/internal/runtime.iface" @"__llgo_stub.main.(*nopCloser).Close"(ptr %0, ptr %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = tail call %"{{.*}}/runtime/internal/runtime.iface" @"main.(*nopCloser).Close"(ptr %1)
// CHECK-NEXT:   ret %"{{.*}}/runtime/internal/runtime.iface" %2
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce { i64, %"{{.*}}/runtime/internal/runtime.iface" } @"__llgo_stub.main.(*nopCloser).Read"(ptr %0, ptr %1, %"{{.*}}/runtime/internal/runtime.Slice" %2){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %3 = tail call { i64, %"{{.*}}/runtime/internal/runtime.iface" } @"main.(*nopCloser).Read"(ptr %1, %"{{.*}}/runtime/internal/runtime.Slice" %2)
// CHECK-NEXT:   ret { i64, %"{{.*}}/runtime/internal/runtime.iface" } %3
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce %"{{.*}}/runtime/internal/runtime.iface" @__llgo_stub.main.nopCloser.Close(ptr %0, %main.nopCloser %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = tail call %"{{.*}}/runtime/internal/runtime.iface" @main.nopCloser.Close(%main.nopCloser %1)
// CHECK-NEXT:   ret %"{{.*}}/runtime/internal/runtime.iface" %2
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce { i64, %"{{.*}}/runtime/internal/runtime.iface" } @__llgo_stub.main.nopCloser.Read(ptr %0, %main.nopCloser %1, %"{{.*}}/runtime/internal/runtime.Slice" %2){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %3 = tail call { i64, %"{{.*}}/runtime/internal/runtime.iface" } @main.nopCloser.Read(%main.nopCloser %1, %"{{.*}}/runtime/internal/runtime.Slice" %2)
// CHECK-NEXT:   ret { i64, %"{{.*}}/runtime/internal/runtime.iface" } %3
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce i64 @"__llgo_stub.main.(*stringReader).Len"(ptr %0, ptr %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = tail call i64 @"main.(*stringReader).Len"(ptr %1)
// CHECK-NEXT:   ret i64 %2
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce { i64, %"{{.*}}/runtime/internal/runtime.iface" } @"__llgo_stub.main.(*stringReader).Read"(ptr %0, ptr %1, %"{{.*}}/runtime/internal/runtime.Slice" %2){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %3 = tail call { i64, %"{{.*}}/runtime/internal/runtime.iface" } @"main.(*stringReader).Read"(ptr %1, %"{{.*}}/runtime/internal/runtime.Slice" %2)
// CHECK-NEXT:   ret { i64, %"{{.*}}/runtime/internal/runtime.iface" } %3
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce { i64, %"{{.*}}/runtime/internal/runtime.iface" } @"__llgo_stub.main.(*stringReader).ReadAt"(ptr %0, ptr %1, %"{{.*}}/runtime/internal/runtime.Slice" %2, i64 %3){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %4 = tail call { i64, %"{{.*}}/runtime/internal/runtime.iface" } @"main.(*stringReader).ReadAt"(ptr %1, %"{{.*}}/runtime/internal/runtime.Slice" %2, i64 %3)
// CHECK-NEXT:   ret { i64, %"{{.*}}/runtime/internal/runtime.iface" } %4
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce { i8, %"{{.*}}/runtime/internal/runtime.iface" } @"__llgo_stub.main.(*stringReader).ReadByte"(ptr %0, ptr %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = tail call { i8, %"{{.*}}/runtime/internal/runtime.iface" } @"main.(*stringReader).ReadByte"(ptr %1)
// CHECK-NEXT:   ret { i8, %"{{.*}}/runtime/internal/runtime.iface" } %2
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce { i32, i64, %"{{.*}}/runtime/internal/runtime.iface" } @"__llgo_stub.main.(*stringReader).ReadRune"(ptr %0, ptr %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = tail call { i32, i64, %"{{.*}}/runtime/internal/runtime.iface" } @"main.(*stringReader).ReadRune"(ptr %1)
// CHECK-NEXT:   ret { i32, i64, %"{{.*}}/runtime/internal/runtime.iface" } %2
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce { i64, %"{{.*}}/runtime/internal/runtime.iface" } @"__llgo_stub.main.(*stringReader).Seek"(ptr %0, ptr %1, i64 %2, i64 %3){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %4 = tail call { i64, %"{{.*}}/runtime/internal/runtime.iface" } @"main.(*stringReader).Seek"(ptr %1, i64 %2, i64 %3)
// CHECK-NEXT:   ret { i64, %"{{.*}}/runtime/internal/runtime.iface" } %4
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce i64 @"__llgo_stub.main.(*stringReader).Size"(ptr %0, ptr %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = tail call i64 @"main.(*stringReader).Size"(ptr %1)
// CHECK-NEXT:   ret i64 %2
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce %"{{.*}}/runtime/internal/runtime.iface" @"__llgo_stub.main.(*stringReader).UnreadByte"(ptr %0, ptr %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = tail call %"{{.*}}/runtime/internal/runtime.iface" @"main.(*stringReader).UnreadByte"(ptr %1)
// CHECK-NEXT:   ret %"{{.*}}/runtime/internal/runtime.iface" %2
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce %"{{.*}}/runtime/internal/runtime.iface" @"__llgo_stub.main.(*stringReader).UnreadRune"(ptr %0, ptr %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = tail call %"{{.*}}/runtime/internal/runtime.iface" @"main.(*stringReader).UnreadRune"(ptr %1)
// CHECK-NEXT:   ret %"{{.*}}/runtime/internal/runtime.iface" %2
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce { i64, %"{{.*}}/runtime/internal/runtime.iface" } @"__llgo_stub.main.(*stringReader).WriteTo"(ptr %0, ptr %1, %"{{.*}}/runtime/internal/runtime.iface" %2){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %3 = tail call { i64, %"{{.*}}/runtime/internal/runtime.iface" } @"main.(*stringReader).WriteTo"(ptr %1, %"{{.*}}/runtime/internal/runtime.iface" %2)
// CHECK-NEXT:   ret { i64, %"{{.*}}/runtime/internal/runtime.iface" } %3
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce %"{{.*}}/runtime/internal/runtime.String" @"__llgo_stub.main.(*errorString).Error"(ptr %0, ptr %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = tail call %"{{.*}}/runtime/internal/runtime.String" @"main.(*errorString).Error"(ptr %1)
// CHECK-NEXT:   ret %"{{.*}}/runtime/internal/runtime.String" %2
// CHECK-NEXT: }
