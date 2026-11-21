// Behavior regression fixture: runs on hosted and on the RV32 baremetal
// scheduler; the golden output must stay identical on both.

package main

import "bytes"

func main() {
	b := []byte("hello world")
	println("index", bytes.Index(b, []byte("world")), bytes.Index(b, []byte("xyz")))
	println("contains", bytes.Contains(b, []byte("lo w")))
	println("edge", bytes.HasPrefix(b, []byte("hello")), bytes.HasSuffix(b, []byte("world")))
	println("join", string(bytes.Join([][]byte{[]byte("a"), []byte("b")}, []byte("-"))))
	println("upper", string(bytes.ToUpper([]byte("abc"))))
	println("repeat", string(bytes.Repeat([]byte("ab"), 3)))
	println("trim", string(bytes.TrimSpace([]byte("  x "))))
	fs := bytes.Fields([]byte("p  q r"))
	println("fields", len(fs), string(fs[1]))
	var buf bytes.Buffer
	buf.WriteString("one,")
	buf.Write([]byte("two"))
	println("buffer", buf.Len(), buf.String())
	r := bytes.NewReader([]byte("abcdef"))
	p := make([]byte, 3)
	n, err := r.Read(p)
	println("read", n, err == nil, string(p))
	n2, err2 := r.Read(p)
	println("read2", n2, err2 == nil, string(p[:n2]))
}
