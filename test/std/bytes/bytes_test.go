package bytes_test

import (
	"bytes"
	"io"
	"strings"
	"testing"
	"unicode"
	"unsafe"
)

func equalByteSlices(a, b [][]byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if !bytes.Equal(a[i], b[i]) {
			return false
		}
	}
	return true
}

func TestBytesBasicFunctions(t *testing.T) {
	src := []byte("go+plus")
	cloned := bytes.Clone(src)
	if !bytes.Equal(cloned, src) {
		t.Fatalf("Clone mismatch: %q", cloned)
	}
	if len(src) > 0 && unsafe.SliceData(cloned) == unsafe.SliceData(src) {
		t.Fatal("Clone should allocate a distinct backing array")
	}

	if bytes.Compare([]byte("a"), []byte("b")) >= 0 {
		t.Fatal("Compare should report a<b")
	}
	if bytes.Compare([]byte("a"), []byte("a")) != 0 {
		t.Fatal("Compare should report equality")
	}

	if !bytes.Contains([]byte("alphabet"), []byte("pha")) {
		t.Fatal("Contains should find subsequence")
	}
	if bytes.ContainsAny([]byte("hello"), "xyz") {
		t.Fatal("ContainsAny should not find characters")
	}
	if !bytes.ContainsAny([]byte("hello"), "xyzlo") {
		t.Fatal("ContainsAny should find shared character")
	}
	if !bytes.ContainsRune([]byte("caf\u00e9"), 'é') {
		t.Fatal("ContainsRune should find rune")
	}

	if bytes.Count([]byte("aaaa"), []byte("aa")) != 2 {
		t.Fatal("Count should consider overlaps correctly")
	}
	if bytes.Count([]byte("abc"), []byte("")) != 1+len("abc") {
		t.Fatal("Count of empty slice should be len+1")
	}

	before, after, found := bytes.Cut([]byte("go+plus"), []byte("+"))
	if !found || !bytes.Equal(before, []byte("go")) || !bytes.Equal(after, []byte("plus")) {
		t.Fatalf("Cut mismatch: %q %q %v", before, after, found)
	}
	trimmed, ok := bytes.CutPrefix([]byte("foobar"), []byte("foo"))
	if !ok || !bytes.Equal(trimmed, []byte("bar")) {
		t.Fatalf("CutPrefix mismatch: %q %v", trimmed, ok)
	}
	chopped, ok := bytes.CutSuffix([]byte("foobar"), []byte("bar"))
	if !ok || !bytes.Equal(chopped, []byte("foo")) {
		t.Fatalf("CutSuffix mismatch: %q %v", chopped, ok)
	}

	if !bytes.EqualFold([]byte("Go"), []byte("gO")) {
		t.Fatal("EqualFold should treat ASCII case-insensitive match")
	}

	fields := bytes.Fields([]byte("  jump over  spaces  "))
	if want := [][]byte{[]byte("jump"), []byte("over"), []byte("spaces")}; !equalByteSlices(fields, want) {
		t.Fatalf("Fields mismatch: %v", fields)
	}

	fieldsFunc := bytes.FieldsFunc([]byte("a|b||c"), func(r rune) bool { return r == '|' })
	if want := [][]byte{[]byte("a"), []byte("b"), []byte("c")}; !equalByteSlices(fieldsFunc, want) {
		t.Fatalf("FieldsFunc mismatch: %v", fieldsFunc)
	}

	if !bytes.HasPrefix([]byte("goplus"), []byte("go")) || !bytes.HasSuffix([]byte("goplus"), []byte("plus")) {
		t.Fatal("HasPrefix/HasSuffix mismatch")
	}

	joined := bytes.Join([][]byte{[]byte("go"), []byte("plus")}, []byte("+"))
	if !bytes.Equal(joined, []byte("go+plus")) {
		t.Fatalf("Join mismatch: %q", joined)
	}

	if bytes.Index([]byte("abracadabra"), []byte("cad")) != 4 {
		t.Fatal("Index mismatch")
	}
	if bytes.IndexAny([]byte("hello"), "xyz") != -1 {
		t.Fatal("IndexAny should miss")
	}
	if bytes.IndexAny([]byte("hello"), "oy") != 4 {
		t.Fatal("IndexAny should find y")
	}
	if bytes.IndexByte([]byte("gopher"), 'p') != 2 {
		t.Fatal("IndexByte mismatch")
	}
	if bytes.IndexFunc([]byte("Hello123"), unicode.IsDigit) != 5 {
		t.Fatal("IndexFunc mismatch")
	}
	if bytes.IndexRune([]byte("caf\u00e9téria"), 'é') != 3 {
		t.Fatal("IndexRune mismatch")
	}

	if bytes.LastIndex([]byte("gophers"), []byte("go")) != 0 {
		t.Fatal("LastIndex mismatch")
	}
	if bytes.LastIndexAny([]byte("hello"), "xyz") != -1 {
		t.Fatal("LastIndexAny should miss")
	}
	if bytes.LastIndexAny([]byte("hello"), "ox") != 4 {
		t.Fatal("LastIndexAny mismatch")
	}
	if bytes.LastIndexByte([]byte("gopher"), 'h') != 3 {
		t.Fatal("LastIndexByte mismatch")
	}
	if bytes.LastIndexFunc([]byte("Hello123"), unicode.IsDigit) != 7 {
		t.Fatal("LastIndexFunc mismatch")
	}

	mapped := bytes.Map(func(r rune) rune {
		if unicode.IsLower(r) {
			return unicode.ToUpper(r)
		}
		return r
	}, []byte("GoPlus"))
	if !bytes.Equal(mapped, []byte("GOPLUS")) {
		t.Fatalf("Map mismatch: %q", mapped)
	}

	if !bytes.Equal(bytes.Repeat([]byte("go"), 3), []byte("gogogo")) {
		t.Fatal("Repeat mismatch")
	}

	if !bytes.Equal(bytes.Replace([]byte("go go"), []byte("go"), []byte("GO"), 1), []byte("GO go")) {
		t.Fatal("Replace mismatch")
	}
	if !bytes.Equal(bytes.ReplaceAll([]byte("go go"), []byte("go"), []byte("GO")), []byte("GO GO")) {
		t.Fatal("ReplaceAll mismatch")
	}

	if !bytes.Equal(bytes.Title([]byte("hello world")), []byte("Hello World")) {
		t.Fatal("Title mismatch")
	}

	if !bytes.Equal(bytes.ToLower([]byte("GoPLUS")), []byte("goplus")) {
		t.Fatal("ToLower mismatch")
	}
	if !bytes.Equal(bytes.ToUpper([]byte("GoPLUS")), []byte("GOPLUS")) {
		t.Fatal("ToUpper mismatch")
	}
	if !bytes.Equal(bytes.ToTitle([]byte("go")), []byte("GO")) {
		t.Fatal("ToTitle mismatch")
	}

	turkish := unicode.TurkishCase
	if !bytes.Equal(bytes.ToLowerSpecial(turkish, []byte("I")), []byte("ı")) {
		t.Fatal("ToLowerSpecial mismatch")
	}
	if !bytes.Equal(bytes.ToUpperSpecial(turkish, []byte("i")), []byte("İ")) {
		t.Fatal("ToUpperSpecial mismatch")
	}
	if !bytes.Equal(bytes.ToTitleSpecial(turkish, []byte("i")), []byte("İ")) {
		t.Fatal("ToTitleSpecial mismatch")
	}

	if got := bytes.ToValidUTF8([]byte("go\xffplus"), []byte("?")); !bytes.Equal(got, []byte("go?plus")) {
		t.Fatalf("ToValidUTF8 mismatch: %q", got)
	}

	if !bytes.Equal(bytes.Trim([]byte("--go--"), "-"), []byte("go")) {
		t.Fatal("Trim mismatch")
	}
	if !bytes.Equal(bytes.TrimFunc([]byte("123go123"), unicode.IsDigit), []byte("go")) {
		t.Fatal("TrimFunc mismatch")
	}
	if !bytes.Equal(bytes.TrimLeft([]byte("..go.."), "."), []byte("go..")) {
		t.Fatal("TrimLeft mismatch")
	}
	if !bytes.Equal(bytes.TrimLeftFunc([]byte("  go"), unicode.IsSpace), []byte("go")) {
		t.Fatal("TrimLeftFunc mismatch")
	}
	if !bytes.Equal(bytes.TrimPrefix([]byte("gopher"), []byte("go")), []byte("pher")) {
		t.Fatal("TrimPrefix mismatch")
	}
	if !bytes.Equal(bytes.TrimRight([]byte("..go.."), "."), []byte("..go")) {
		t.Fatal("TrimRight mismatch")
	}
	if !bytes.Equal(bytes.TrimRightFunc([]byte("go  "), unicode.IsSpace), []byte("go")) {
		t.Fatal("TrimRightFunc mismatch")
	}
	if !bytes.Equal(bytes.TrimSpace([]byte("  go  ")), []byte("go")) {
		t.Fatal("TrimSpace mismatch")
	}
	if !bytes.Equal(bytes.TrimSuffix([]byte("gopher"), []byte("her")), []byte("gop")) {
		t.Fatal("TrimSuffix mismatch")
	}

	if runes := bytes.Runes([]byte("πλα")); string(runes) != "πλα" {
		t.Fatalf("Runes mismatch: %v", runes)
	}

	if bytes.MinRead <= 0 {
		t.Fatalf("MinRead should be positive, got %d", bytes.MinRead)
	}

	if bytes.ErrTooLarge == nil || bytes.ErrTooLarge.Error() == "" {
		t.Fatal("ErrTooLarge should provide a descriptive error")
	}
}

func TestBytesSplitFunctions(t *testing.T) {
	splits := bytes.Split([]byte("a,b,c"), []byte(","))
	if want := [][]byte{[]byte("a"), []byte("b"), []byte("c")}; !equalByteSlices(splits, want) {
		t.Fatalf("Split mismatch: %v", splits)
	}

	splitsAfter := bytes.SplitAfter([]byte("a,b"), []byte(","))
	if want := [][]byte{[]byte("a,"), []byte("b")}; !equalByteSlices(splitsAfter, want) {
		t.Fatalf("SplitAfter mismatch: %v", splitsAfter)
	}

	splitsAfterN := bytes.SplitAfterN([]byte("a,b,c"), []byte(","), 2)
	if want := [][]byte{[]byte("a,"), []byte("b,c")}; !equalByteSlices(splitsAfterN, want) {
		t.Fatalf("SplitAfterN mismatch: %v", splitsAfterN)
	}

	splitsN := bytes.SplitN([]byte("a,b,c"), []byte(","), 2)
	if want := [][]byte{[]byte("a"), []byte("b,c")}; !equalByteSlices(splitsN, want) {
		t.Fatalf("SplitN mismatch: %v", splitsN)
	}

	lines := bytes.Split([]byte("a\nb\n"), []byte("\n"))
	if want := [][]byte{[]byte("a"), []byte("b"), []byte("")}; !equalByteSlices(lines, want) {
		t.Fatalf("Split lines mismatch: %v", lines)
	}
}

func TestBytesReader(t *testing.T) {
	data := []byte("hello")
	r := bytes.NewReader(data)
	if r == nil {
		t.Fatal("NewReader returned nil")
	}
	if r.Size() != int64(len(data)) {
		t.Fatal("Size mismatch")
	}

	buf := make([]byte, 2)
	if n, err := r.Read(buf); err != nil || n != 2 || !bytes.Equal(buf[:n], []byte("he")) {
		t.Fatalf("Read mismatch: n=%d err=%v buf=%q", n, err, buf[:n])
	}
	if r.Len() != len(data)-2 {
		t.Fatal("Len after read mismatch")
	}

	b, err := r.ReadByte()
	if err != nil || b != 'l' {
		t.Fatalf("ReadByte mismatch: %v %q", err, b)
	}
	if err := r.UnreadByte(); err != nil {
		t.Fatalf("UnreadByte error: %v", err)
	}

	runeVal, size, err := r.ReadRune()
	if err != nil || runeVal != 'l' || size != 1 {
		t.Fatalf("ReadRune mismatch: %v %q %d", err, runeVal, size)
	}
	if err := r.UnreadRune(); err != nil {
		t.Fatalf("UnreadRune error: %v", err)
	}
	if _, _, err := r.ReadRune(); err != nil {
		t.Fatalf("ReadRune after UnreadRune error: %v", err)
	}

	if pos, err := r.Seek(0, io.SeekStart); err != nil || pos != 0 {
		t.Fatalf("Seek mismatch: %v %d", err, pos)
	}

	readAtBuf := make([]byte, len(data))
	n, err := r.ReadAt(readAtBuf, 0)
	if n != len(data) || (err != nil && err != io.EOF) || !bytes.Equal(readAtBuf, data) {
		t.Fatalf("ReadAt mismatch: n=%d err=%v buf=%q", n, err, readAtBuf)
	}

	r.Reset(data)
	if r.Len() != len(data) {
		t.Fatal("Len after Reset mismatch")
	}

	var out bytes.Buffer
	if written, err := r.WriteTo(&out); err != nil || written != int64(len(data)) {
		t.Fatalf("WriteTo mismatch: written=%d err=%v", written, err)
	}
	if !bytes.Equal(out.Bytes(), data) {
		t.Fatalf("WriteTo output mismatch: %q", out.Bytes())
	}
}

func TestBytesBuffer(t *testing.T) {
	var buf bytes.Buffer
	if buf.Len() != 0 {
		t.Fatal("zero Buffer should have length 0")
	}

	buf.Grow(16)
	if buf.Cap() < 16 {
		t.Fatalf("Grow should ensure capacity, got %d", buf.Cap())
	}

	if n, err := buf.Write([]byte("Go")); err != nil || n != 2 {
		t.Fatalf("Write mismatch: n=%d err=%v", n, err)
	}
	if err := buf.WriteByte('+'); err != nil {
		t.Fatalf("WriteByte error: %v", err)
	}
	if _, err := buf.WriteRune('π'); err != nil {
		t.Fatalf("WriteRune error: %v", err)
	}
	if _, err := buf.WriteString("Plus"); err != nil {
		t.Fatalf("WriteString error: %v", err)
	}

	if buf.Len() != len("Go+πPlus") {
		t.Fatal("Buffer should contain expected length")
	}
	if !bytes.Equal(buf.Bytes(), []byte("Go+πPlus")) {
		t.Fatalf("Buffer Bytes mismatch: %q", buf.Bytes())
	}

	if got := buf.Next(2); !bytes.Equal(got, []byte("Go")) {
		t.Fatalf("Next mismatch: %q", got)
	}
	if c, err := buf.ReadByte(); err != nil || c != '+' {
		t.Fatalf("ReadByte mismatch: %v %q", err, c)
	}
	if err := buf.UnreadByte(); err != nil {
		t.Fatalf("UnreadByte error: %v", err)
	}
	if c, err := buf.ReadByte(); err != nil || c != '+' {
		t.Fatalf("ReadByte after UnreadByte mismatch: %v %q", err, c)
	}
	if r, size, err := buf.ReadRune(); err != nil || r != 'π' || size != 2 {
		t.Fatalf("ReadRune mismatch: %v %q %d", err, r, size)
	}
	if err := buf.UnreadRune(); err != nil {
		t.Fatalf("UnreadRune error: %v", err)
	}
	if r, size, err := buf.ReadRune(); err != nil || r != 'π' || size != 2 {
		t.Fatalf("ReadRune after UnreadRune mismatch: %v %q %d", err, r, size)
	}
	if s, err := buf.ReadString('s'); err != nil || s != "Plus" {
		t.Fatalf("ReadString mismatch: %v %q", err, s)
	}
	if buf.Len() != 0 {
		t.Fatal("Buffer should be empty after consuming data")
	}

	buf.WriteString("read")
	readBuf := make([]byte, 2)
	if n, err := buf.Read(readBuf); n != 2 || err != nil || !bytes.Equal(readBuf[:n], []byte("re")) {
		t.Fatalf("Read mismatch: n=%d err=%v buf=%q", n, err, readBuf[:n])
	}
	if n, err := buf.Read(readBuf); n != 2 || err != nil || !bytes.Equal(readBuf[:n], []byte("ad")) {
		t.Fatalf("Read second mismatch: n=%d err=%v buf=%q", n, err, readBuf[:n])
	}
	if n, err := buf.Read(readBuf); n != 0 || err != io.EOF {
		t.Fatalf("Read EOF mismatch: n=%d err=%v", n, err)
	}

	buf.WriteString("hello|world|")
	if line, err := buf.ReadBytes('|'); err != nil || !bytes.Equal(line, []byte("hello|")) {
		t.Fatalf("ReadBytes mismatch: %v %q", err, line)
	}
	if line, err := buf.ReadBytes('|'); err != nil || !bytes.Equal(line, []byte("world|")) {
		t.Fatalf("ReadBytes second mismatch: %v %q", err, line)
	}

	buf.Reset()
	if buf.Len() != 0 {
		t.Fatal("Reset should clear buffer")
	}

	if _, err := buf.ReadFrom(strings.NewReader("abc")); err != nil {
		t.Fatalf("ReadFrom error: %v", err)
	}
	if buf.String() != "abc" {
		t.Fatalf("ReadFrom output mismatch: %q", buf.String())
	}

	var out bytes.Buffer
	if n, err := buf.WriteTo(&out); err != nil || int(n) != 3 {
		t.Fatalf("WriteTo mismatch: %v %d", err, n)
	}
	if out.String() != "abc" {
		t.Fatalf("WriteTo output mismatch: %q", out.String())
	}
	if buf.Len() != 0 {
		t.Fatal("Buffer should be empty after WriteTo")
	}

	b := bytes.NewBufferString("xyz")
	if b.Len() != 3 {
		t.Fatalf("NewBufferString Len mismatch: %d", b.Len())
	}
	if b.Cap() < 3 {
		t.Fatalf("NewBufferString Cap mismatch: %d", b.Cap())
	}
	if val, err := b.ReadString('y'); err != nil || val != "xy" {
		t.Fatalf("ReadString from NewBufferString mismatch: %v %q", err, val)
	}
	b.Truncate(0)
	if b.Len() != 0 {
		t.Fatal("Truncate should clear buffer")
	}
}
