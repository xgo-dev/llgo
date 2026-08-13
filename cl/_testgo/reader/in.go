// LITTEST
package main

import (
	"unicode/utf8"
)

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

func (e *errorString) Error() string {
	return e.s
}

var (
	EOF           = newError("EOF")
	ErrShortWrite = newError("short write")
)

func main() {
	r := &stringReader{s: "hello world"}
	data, err := ReadAll(r)
	println(string(data), err)
}

// NopCloser preserves the Reader payload and chooses the WriterTo-capable wrapper.
// CHECK-LABEL: define %"{{.*}}/runtime/internal/runtime.iface" @main.NopCloser(%"{{.*}}/runtime/internal/runtime.iface" %0){{.*}} {
// CHECK: [[NC_TYPE:%[0-9]+]] = call ptr @"{{.*}}/runtime/internal/runtime.IfaceType"(%"{{.*}}/runtime/internal/runtime.iface" %0)
// CHECK: [[NC_WRITER_TO:%[0-9]+]] = call i1 @"{{.*}}/runtime/internal/runtime.Implements"(ptr @_llgo_main.WriterTo, ptr [[NC_TYPE]])
// CHECK: br i1 [[NC_WRITER_TO]],
// CHECK: store %"{{.*}}/runtime/internal/runtime.iface" %0, ptr %{{[0-9]+}}
// CHECK: [[NC_WT_ITAB:%[0-9]+]] = call ptr @"{{.*}}/runtime/internal/runtime.NewItab"({{.*}}ptr @_llgo_main.nopCloserWriterTo)
// CHECK: ret %"{{.*}}/runtime/internal/runtime.iface" %{{[0-9]+}}
// CHECK: store %"{{.*}}/runtime/internal/runtime.iface" %0, ptr %{{[0-9]+}}
// CHECK: [[NC_ITAB:%[0-9]+]] = call ptr @"{{.*}}/runtime/internal/runtime.NewItab"({{.*}}ptr @_llgo_main.nopCloser)
// CHECK: ret %"{{.*}}/runtime/internal/runtime.iface" %{{[0-9]+}}

// ReadAll passes only the unused capacity to Reader.Read, extends by n, clears
// EOF, and grows only when len == cap.
// CHECK-LABEL: define { %"{{.*}}/runtime/internal/runtime.Slice", %"{{.*}}/runtime/internal/runtime.iface" } @main.ReadAll(%"{{.*}}/runtime/internal/runtime.iface" %0){{.*}} {
// CHECK: [[READ_BUF:%[0-9]+]] = call %"{{.*}}/runtime/internal/runtime.Slice" @"{{.*}}/runtime/internal/runtime.NewSlice2"({{.*}}i64 512, i64 0, i64 0,
// CHECK: [[READ_TAIL:%[0-9]+]] = call %"{{.*}}/runtime/internal/runtime.Slice" @"{{.*}}/runtime/internal/runtime.NewSlice2"({{.*}}i64 %{{[0-9]+}}, i64 %{{[0-9]+}},
// CHECK: [[READ_DATA:%[0-9]+]] = call ptr @"{{.*}}/runtime/internal/runtime.IfacePtrData"(%"{{.*}}/runtime/internal/runtime.iface" %0)
// CHECK-NEXT: [[READ_TABLE:%[0-9]+]] = extractvalue %"{{.*}}/runtime/internal/runtime.iface" %0, 0
// CHECK-NEXT: [[READ_SLOT:%[0-9]+]] = getelementptr ptr, ptr [[READ_TABLE]], i64 3
// CHECK-NEXT: [[READ_METHOD:%[0-9]+]] = load ptr, ptr [[READ_SLOT]]
// CHECK: [[READ_PAIR:%[0-9]+]] = call { i64, %"{{.*}}/runtime/internal/runtime.iface" } %{{[0-9]+}}(ptr %{{[0-9]+}}, %"{{.*}}/runtime/internal/runtime.Slice" [[READ_TAIL]])
// CHECK: [[READ_N:%[0-9]+]] = extractvalue { i64, %"{{.*}}/runtime/internal/runtime.iface" } [[READ_PAIR]], 0
// CHECK: [[READ_ERR:%[0-9]+]] = extractvalue { i64, %"{{.*}}/runtime/internal/runtime.iface" } [[READ_PAIR]], 1
// CHECK: [[READ_NEW_LEN:%[0-9]+]] = add i64 %{{[0-9]+}}, [[READ_N]]
// CHECK: [[READ_EXTENDED:%[0-9]+]] = call %"{{.*}}/runtime/internal/runtime.Slice" @"{{.*}}/runtime/internal/runtime.NewSlice2"({{.*}}i64 [[READ_NEW_LEN]],
// CHECK: [[READ_HAS_ERR:%[0-9]+]] = xor i1 %{{[0-9]+}}, true
// CHECK: br i1 [[READ_HAS_ERR]],
// CHECK: [[READ_EOF:%[0-9]+]] = load %"{{.*}}/runtime/internal/runtime.iface", ptr @main.EOF
// CHECK: [[READ_IS_EOF:%[0-9]+]] = call i1 @"{{.*}}/runtime/internal/runtime.EfaceEqual"
// CHECK: [[READ_FULL:%[0-9]+]] = icmp eq i64 %{{[0-9]+}}, %{{[0-9]+}}
// CHECK: br i1 [[READ_FULL]],
// CHECK: [[READ_GROWN:%[0-9]+]] = call %"{{.*}}/runtime/internal/runtime.Slice" @"{{.*}}/runtime/internal/runtime.SliceAppend"(%"{{.*}}/runtime/internal/runtime.Slice" [[READ_EXTENDED]], ptr %{{[0-9]+}}, i64 %{{[0-9]+}}, i64 1)

// WriteString dispatches StringWriter.WriteString when available and otherwise
// converts the same string to bytes for Writer.Write.
// CHECK-LABEL: define { i64, %"{{.*}}/runtime/internal/runtime.iface" } @main.WriteString(%"{{.*}}/runtime/internal/runtime.iface" %0, %"{{.*}}/runtime/internal/runtime.String" %1){{.*}} {
// CHECK: [[WS_TYPE:%[0-9]+]] = call ptr @"{{.*}}/runtime/internal/runtime.IfaceType"(%"{{.*}}/runtime/internal/runtime.iface" %0)
// CHECK: [[WS_FAST:%[0-9]+]] = call i1 @"{{.*}}/runtime/internal/runtime.Implements"(ptr @_llgo_main.StringWriter, ptr [[WS_TYPE]])
// CHECK: br i1 [[WS_FAST]],
// CHECK: [[WS_FAST_RESULT:%[0-9]+]] = call { i64, %"{{.*}}/runtime/internal/runtime.iface" } %{{[0-9]+}}(ptr %{{[0-9]+}}, %"{{.*}}/runtime/internal/runtime.String" %1)
// CHECK: [[WS_BYTES:%[0-9]+]] = call %"{{.*}}/runtime/internal/runtime.Slice" @"{{.*}}/runtime/internal/runtime.StringToBytes"(%"{{.*}}/runtime/internal/runtime.String" %1)
// CHECK: [[WS_FALLBACK:%[0-9]+]] = call { i64, %"{{.*}}/runtime/internal/runtime.iface" } %{{[0-9]+}}(ptr %{{[0-9]+}}, %"{{.*}}/runtime/internal/runtime.Slice" [[WS_BYTES]])

// The runtime driver constructs *stringReader as Reader and feeds both ReadAll results to print.
// CHECK-LABEL: define void @main.main(){{.*}} {
// CHECK: [[MAIN_READER_ITAB:%[0-9]+]] = call ptr @"{{.*}}/runtime/internal/runtime.NewItab"({{.*}}ptr @"*_llgo_main.stringReader")
// CHECK: [[MAIN_READER:%[0-9]+]] = insertvalue %"{{.*}}/runtime/internal/runtime.iface" %{{[0-9]+}}, ptr %{{[0-9]+}}, 1
// CHECK: [[MAIN_READ:%[0-9]+]] = call { %"{{.*}}/runtime/internal/runtime.Slice", %"{{.*}}/runtime/internal/runtime.iface" } @main.ReadAll(%"{{.*}}/runtime/internal/runtime.iface" [[MAIN_READER]])
// CHECK: [[MAIN_BYTES:%[0-9]+]] = extractvalue { %"{{.*}}/runtime/internal/runtime.Slice", %"{{.*}}/runtime/internal/runtime.iface" } [[MAIN_READ]], 0
// CHECK: [[MAIN_ERR:%[0-9]+]] = extractvalue { %"{{.*}}/runtime/internal/runtime.Slice", %"{{.*}}/runtime/internal/runtime.iface" } [[MAIN_READ]], 1
// CHECK: [[MAIN_STRING:%[0-9]+]] = call %"{{.*}}/runtime/internal/runtime.String" @"{{.*}}/runtime/internal/runtime.StringFromBytes"(%"{{.*}}/runtime/internal/runtime.Slice" [[MAIN_BYTES]])
// CHECK: call void @"{{.*}}/runtime/internal/runtime.PrintString"(%"{{.*}}/runtime/internal/runtime.String" [[MAIN_STRING]])
// CHECK: call void @"{{.*}}/runtime/internal/runtime.PrintIface"(%"{{.*}}/runtime/internal/runtime.iface" [[MAIN_ERR]])

// CHECK-LABEL: define %"{{.*}}/runtime/internal/runtime.iface" @main.newError(%"{{.*}}/runtime/internal/runtime.String" %0){{.*}} {
// CHECK: store %"{{.*}}/runtime/internal/runtime.String" %0, ptr %{{[0-9]+}}
// CHECK: [[ERROR_ITAB:%[0-9]+]] = call ptr @"{{.*}}/runtime/internal/runtime.NewItab"({{.*}}ptr @"*_llgo_main.errorString")
// CHECK: ret %"{{.*}}/runtime/internal/runtime.iface" %{{[0-9]+}}

// The WriterTo wrapper asserts its embedded Reader, dispatches WriteTo, and returns both results.
// CHECK-LABEL: define { i64, %"{{.*}}/runtime/internal/runtime.iface" } @main.nopCloserWriterTo.WriteTo(%main.nopCloserWriterTo %0, %"{{.*}}/runtime/internal/runtime.iface" %1){{.*}} {
// CHECK: store %main.nopCloserWriterTo %0, ptr [[WT_RECEIVER:%[0-9]+]]
// CHECK: [[WT_READER_FIELD:%[0-9]+]] = getelementptr inbounds %main.nopCloserWriterTo, ptr [[WT_RECEIVER]], i32 0, i32 0
// CHECK-NEXT: [[WT_READER:%[0-9]+]] = load %"{{.*}}/runtime/internal/runtime.iface", ptr [[WT_READER_FIELD]]
// CHECK: [[WT_TYPE:%[0-9]+]] = call ptr @"{{.*}}/runtime/internal/runtime.IfaceType"(%"{{.*}}/runtime/internal/runtime.iface" [[WT_READER]])
// CHECK: [[WT_OK:%[0-9]+]] = call i1 @"{{.*}}/runtime/internal/runtime.Implements"(ptr @_llgo_main.WriterTo, ptr [[WT_TYPE]])
// CHECK: br i1 [[WT_OK]],
// CHECK: [[WT_RESULT:%[0-9]+]] = call { i64, %"{{.*}}/runtime/internal/runtime.iface" } %{{[0-9]+}}(ptr %{{[0-9]+}}, %"{{.*}}/runtime/internal/runtime.iface" %1)
// CHECK: [[WT_N:%[0-9]+]] = extractvalue { i64, %"{{.*}}/runtime/internal/runtime.iface" } [[WT_RESULT]], 0
// CHECK: [[WT_ERR:%[0-9]+]] = extractvalue { i64, %"{{.*}}/runtime/internal/runtime.iface" } [[WT_RESULT]], 1

// CHECK-LABEL: define { i64, %"{{.*}}/runtime/internal/runtime.iface" } @"main.(*nopCloserWriterTo).WriteTo"(ptr %0, %"{{.*}}/runtime/internal/runtime.iface" %1){{.*}} {
// CHECK: [[WT_VALUE:%[0-9]+]] = load %main.nopCloserWriterTo, ptr %0
// CHECK: [[WT_WRAPPED:%[0-9]+]] = call { i64, %"{{.*}}/runtime/internal/runtime.iface" } @main.nopCloserWriterTo.WriteTo(%main.nopCloserWriterTo [[WT_VALUE]], %"{{.*}}/runtime/internal/runtime.iface" %1)

// Len computes max(len(s)-i, 0).
// CHECK-LABEL: define i64 @"main.(*stringReader).Len"(ptr %0){{.*}} {
// CHECK: [[LEN_I_FIELD:%[0-9]+]] = getelementptr inbounds %main.stringReader, ptr %0, i32 0, i32 1
// CHECK-NEXT: [[LEN_I:%[0-9]+]] = load i64, ptr [[LEN_I_FIELD]]
// CHECK: [[LEN_S_FIELD:%[0-9]+]] = getelementptr inbounds %main.stringReader, ptr %0, i32 0, i32 0
// CHECK-NEXT: [[LEN_S_VALUE:%[0-9]+]] = load %"{{.*}}/runtime/internal/runtime.String", ptr [[LEN_S_FIELD]]
// CHECK-NEXT: [[LEN_S:%[0-9]+]] = extractvalue %"{{.*}}/runtime/internal/runtime.String" [[LEN_S_VALUE]], 1
// CHECK: [[LEN_DONE:%[0-9]+]] = icmp sge i64 [[LEN_I]], [[LEN_S]]
// CHECK: [[LEN_REMAIN:%[0-9]+]] = sub i64 %{{[0-9]+}}, %{{[0-9]+}}
// CHECK: ret i64 [[LEN_REMAIN]]

// Read slices s at i, copies into b, advances i by n, and returns EOF at the end.
// CHECK-LABEL: define { i64, %"{{.*}}/runtime/internal/runtime.iface" } @"main.(*stringReader).Read"(ptr %0, %"{{.*}}/runtime/internal/runtime.Slice" %1){{.*}} {
// CHECK: [[R_I_FIELD:%[0-9]+]] = getelementptr inbounds %main.stringReader, ptr %0, i32 0, i32 1
// CHECK-NEXT: [[R_I:%[0-9]+]] = load i64, ptr [[R_I_FIELD]]
// CHECK: [[R_S_FIELD:%[0-9]+]] = getelementptr inbounds %main.stringReader, ptr %0, i32 0, i32 0
// CHECK-NEXT: [[R_S:%[0-9]+]] = load %"{{.*}}/runtime/internal/runtime.String", ptr [[R_S_FIELD]]
// CHECK-NEXT: [[R_LEN:%[0-9]+]] = extractvalue %"{{.*}}/runtime/internal/runtime.String" [[R_S]], 1
// CHECK: [[R_EOF:%[0-9]+]] = icmp sge i64 [[R_I]], [[R_LEN]]
// CHECK: load %"{{.*}}/runtime/internal/runtime.iface", ptr @main.EOF
// CHECK: store i64 -1, ptr %{{[0-9]+}}
// CHECK: [[R_SUFFIX:%[0-9]+]] = call %"{{.*}}/runtime/internal/runtime.String" @"{{.*}}/runtime/internal/runtime.StringSlice2"({{.*}}i64 [[R_SLICE_I:%[0-9]+]],
// CHECK: [[R_N:%[0-9]+]] = call i64 @"{{.*}}/runtime/internal/runtime.SliceCopy"(%"{{.*}}/runtime/internal/runtime.Slice" %1,
// CHECK: [[R_NEXT:%[0-9]+]] = add i64 %{{[0-9]+}}, [[R_N]]
// CHECK: store i64 [[R_NEXT]], ptr %{{[0-9]+}}

// ReadAt rejects negative/past-end offsets and reports EOF for a short copy.
// CHECK-LABEL: define { i64, %"{{.*}}/runtime/internal/runtime.iface" } @"main.(*stringReader).ReadAt"(ptr %0, %"{{.*}}/runtime/internal/runtime.Slice" %1, i64 %2){{.*}} {
// CHECK: [[RA_NEG:%[0-9]+]] = icmp slt i64 %2, 0
// CHECK: call %"{{.*}}/runtime/internal/runtime.iface" @main.newError(
// CHECK: [[RA_PAST:%[0-9]+]] = icmp sge i64 %2, %{{[0-9]+}}
// CHECK: load %"{{.*}}/runtime/internal/runtime.iface", ptr @main.EOF
// CHECK: [[RA_SUFFIX:%[0-9]+]] = call %"{{.*}}/runtime/internal/runtime.String" @"{{.*}}/runtime/internal/runtime.StringSlice2"({{.*}}i64 %2,
// CHECK: [[RA_N:%[0-9]+]] = call i64 @"{{.*}}/runtime/internal/runtime.SliceCopy"(%"{{.*}}/runtime/internal/runtime.Slice" %1,
// CHECK: [[RA_TARGET_LEN:%[0-9]+]] = extractvalue %"{{.*}}/runtime/internal/runtime.Slice" %1, 1
// CHECK-NEXT: [[RA_SHORT:%[0-9]+]] = icmp slt i64 [[RA_N]], [[RA_TARGET_LEN]]

// ReadByte invalidates prevRune, checks EOF, returns s[i], and increments i once.
// CHECK-LABEL: define { i8, %"{{.*}}/runtime/internal/runtime.iface" } @"main.(*stringReader).ReadByte"(ptr %0){{.*}} {
// CHECK: [[RB_PREV_FIELD:%[0-9]+]] = getelementptr inbounds %main.stringReader, ptr %0, i32 0, i32 2
// CHECK-NEXT: store i64 -1, ptr [[RB_PREV_FIELD]]
// CHECK: [[RB_I_FIELD:%[0-9]+]] = getelementptr inbounds %main.stringReader, ptr %0, i32 0, i32 1
// CHECK-NEXT: [[RB_I:%[0-9]+]] = load i64, ptr [[RB_I_FIELD]]
// CHECK: [[RB_LEN:%[0-9]+]] = extractvalue %"{{.*}}/runtime/internal/runtime.String" %{{[0-9]+}}, 1
// CHECK: [[RB_EOF:%[0-9]+]] = icmp sge i64 [[RB_I]], [[RB_LEN]]
// CHECK: [[RB_PTR:%[0-9]+]] = getelementptr inbounds i8, ptr %{{[0-9]+}}, i64 [[RB_INDEX:%[0-9]+]]
// CHECK: [[RB_BYTE:%[0-9]+]] = load i8, ptr [[RB_PTR]]
// CHECK: [[RB_NEXT:%[0-9]+]] = add i64 %{{[0-9]+}}, 1
// CHECK: store i64 [[RB_NEXT]], ptr %{{[0-9]+}}

// ReadRune records prevRune, has an ASCII fast path, and advances by the decoded UTF-8 width.
// CHECK-LABEL: define { i32, i64, %"{{.*}}/runtime/internal/runtime.iface" } @"main.(*stringReader).ReadRune"(ptr %0){{.*}} {
// CHECK: [[RR_I:%[0-9]+]] = load i64, ptr %{{[0-9]+}}
// CHECK: [[RR_LEN:%[0-9]+]] = extractvalue %"{{.*}}/runtime/internal/runtime.String" %{{[0-9]+}}, 1
// CHECK: [[RR_EOF:%[0-9]+]] = icmp sge i64 [[RR_I]], [[RR_LEN]]
// CHECK: store i64 [[RR_PREV_I:%[0-9]+]], ptr %{{[0-9]+}}
// CHECK: [[RR_BYTE:%[0-9]+]] = load i8, ptr %{{[0-9]+}}
// CHECK: [[RR_ASCII:%[0-9]+]] = icmp ult i8 [[RR_BYTE]], -128
// CHECK: [[RR_ASCII_NEXT:%[0-9]+]] = add i64 %{{[0-9]+}}, 1
// CHECK: [[RR_RUNE:%[0-9]+]] = zext i8 [[RR_BYTE]] to i32
// CHECK: [[RR_DECODE:%[0-9]+]] = call { i32, i64 } @"unicode/utf8.DecodeRuneInString"
// CHECK: [[RR_WIDTH:%[0-9]+]] = extractvalue { i32, i64 } [[RR_DECODE]], 1
// CHECK: [[RR_NEXT:%[0-9]+]] = add i64 %{{[0-9]+}}, [[RR_WIDTH]]
// CHECK: store i64 [[RR_NEXT]], ptr %{{[0-9]+}}

// Seek handles start/current/end, rejects invalid/negative results, and stores abs.
// CHECK-LABEL: define { i64, %"{{.*}}/runtime/internal/runtime.iface" } @"main.(*stringReader).Seek"(ptr %0, i64 %1, i64 %2){{.*}} {
// CHECK: store i64 -1, ptr %{{[0-9]+}}
// CHECK: icmp eq i64 %2, 0
// CHECK: [[SEEK_ABS:%[0-9]+]] = phi i64
// CHECK: [[SEEK_NEG:%[0-9]+]] = icmp slt i64 [[SEEK_ABS]], 0
// CHECK: [[SEEK_CURRENT:%[0-9]+]] = add i64 %{{[0-9]+}}, %1
// CHECK: icmp eq i64 %2, 1
// CHECK: [[SEEK_END:%[0-9]+]] = add i64 %{{[0-9]+}}, %1
// CHECK: icmp eq i64 %2, 2
// CHECK: call %"{{.*}}/runtime/internal/runtime.iface" @main.newError(
// CHECK: call %"{{.*}}/runtime/internal/runtime.iface" @main.newError(
// CHECK: store i64 [[SEEK_ABS]], ptr %{{[0-9]+}}

// CHECK-LABEL: define i64 @"main.(*stringReader).Size"(ptr %0){{.*}} {
// CHECK: [[SIZE_FIELD:%[0-9]+]] = getelementptr inbounds %main.stringReader, ptr %0, i32 0, i32 0
// CHECK-NEXT: [[SIZE_STRING:%[0-9]+]] = load %"{{.*}}/runtime/internal/runtime.String", ptr [[SIZE_FIELD]]
// CHECK-NEXT: [[SIZE:%[0-9]+]] = extractvalue %"{{.*}}/runtime/internal/runtime.String" [[SIZE_STRING]], 1
// CHECK: ret i64 [[SIZE]]

// UnreadByte decrements i only after the beginning check and invalidates prevRune.
// CHECK-LABEL: define %"{{.*}}/runtime/internal/runtime.iface" @"main.(*stringReader).UnreadByte"(ptr %0){{.*}} {
// CHECK: [[UB_I_FIELD:%[0-9]+]] = getelementptr inbounds %main.stringReader, ptr %0, i32 0, i32 1
// CHECK-NEXT: [[UB_I:%[0-9]+]] = load i64, ptr [[UB_I_FIELD]]
// CHECK: [[UB_BAD:%[0-9]+]] = icmp sle i64 [[UB_I]], 0
// CHECK: call %"{{.*}}/runtime/internal/runtime.iface" @main.newError(
// CHECK: store i64 -1, ptr %{{[0-9]+}}
// CHECK: [[UB_PREV:%[0-9]+]] = sub i64 %{{[0-9]+}}, 1
// CHECK: store i64 [[UB_PREV]], ptr %{{[0-9]+}}

// UnreadRune requires both i > 0 and a recorded rune, then restores i and clears prevRune.
// CHECK-LABEL: define %"{{.*}}/runtime/internal/runtime.iface" @"main.(*stringReader).UnreadRune"(ptr %0){{.*}} {
// CHECK: [[UR_I_FIELD:%[0-9]+]] = getelementptr inbounds %main.stringReader, ptr %0, i32 0, i32 1
// CHECK-NEXT: [[UR_I:%[0-9]+]] = load i64, ptr [[UR_I_FIELD]]
// CHECK: [[UR_BEGIN:%[0-9]+]] = icmp sle i64 [[UR_I]], 0
// CHECK: call %"{{.*}}/runtime/internal/runtime.iface" @main.newError(
// CHECK: [[UR_PREV:%[0-9]+]] = load i64, ptr %{{[0-9]+}}
// CHECK: [[UR_INVALID:%[0-9]+]] = icmp slt i64 [[UR_PREV]], 0
// CHECK: call %"{{.*}}/runtime/internal/runtime.iface" @main.newError(
// CHECK: store i64 [[UR_RESTORE:%[0-9]+]], ptr %{{[0-9]+}}
// CHECK: store i64 -1, ptr %{{[0-9]+}}

// WriteTo writes exactly s[i:], advances by m, rejects over-reporting, and substitutes ErrShortWrite only for a nil short-write error.
// CHECK-LABEL: define { i64, %"{{.*}}/runtime/internal/runtime.iface" } @"main.(*stringReader).WriteTo"(ptr %0, %"{{.*}}/runtime/internal/runtime.iface" %1){{.*}} {
// CHECK: store i64 -1, ptr %{{[0-9]+}}
// CHECK: [[WTO_I:%[0-9]+]] = load i64, ptr %{{[0-9]+}}
// CHECK: [[WTO_LEN:%[0-9]+]] = extractvalue %"{{.*}}/runtime/internal/runtime.String" %{{[0-9]+}}, 1
// CHECK: [[WTO_DONE:%[0-9]+]] = icmp sge i64 [[WTO_I]], [[WTO_LEN]]
// CHECK: [[WTO_SUFFIX:%[0-9]+]] = call %"{{.*}}/runtime/internal/runtime.String" @"{{.*}}/runtime/internal/runtime.StringSlice2"({{.*}}i64 [[WTO_SLICE_I:%[0-9]+]],
// CHECK: [[WTO_RESULT:%[0-9]+]] = call { i64, %"{{.*}}/runtime/internal/runtime.iface" } @main.WriteString(%"{{.*}}/runtime/internal/runtime.iface" %1, %"{{.*}}/runtime/internal/runtime.String" [[WTO_SUFFIX]])
// CHECK: [[WTO_N:%[0-9]+]] = extractvalue { i64, %"{{.*}}/runtime/internal/runtime.iface" } [[WTO_RESULT]], 0
// CHECK: [[WTO_ERR:%[0-9]+]] = extractvalue { i64, %"{{.*}}/runtime/internal/runtime.iface" } [[WTO_RESULT]], 1
// CHECK: [[WTO_OVER:%[0-9]+]] = icmp sgt i64 [[WTO_N]], %{{[0-9]+}}
// CHECK: call void @"{{.*}}/runtime/internal/runtime.Panic"
// CHECK: [[WTO_NEXT:%[0-9]+]] = add i64 %{{[0-9]+}}, [[WTO_N]]
// CHECK: store i64 [[WTO_NEXT]], ptr %{{[0-9]+}}
// CHECK: [[WTO_SHORT:%[0-9]+]] = icmp ne i64 [[WTO_N]], %{{[0-9]+}}
// CHECK: load %"{{.*}}/runtime/internal/runtime.iface", ptr @main.ErrShortWrite
