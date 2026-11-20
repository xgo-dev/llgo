//go:build llgo && baremetal

package bytealg

//llgo:skip Compare CompareString Equal Count CountString IndexByte IndexByteString

func comparePortable[T ~[]byte | ~string](a, b T) int {
	n := len(a)
	if len(b) < n {
		n = len(b)
	}
	for i := 0; i < n; i++ {
		if a[i] < b[i] {
			return -1
		}
		if a[i] > b[i] {
			return 1
		}
	}
	if len(a) < len(b) {
		return -1
	}
	if len(a) > len(b) {
		return 1
	}
	return 0
}

func Compare(a, b []byte) int       { return comparePortable(a, b) }
func CompareString(a, b string) int { return comparePortable(a, b) }

func Equal(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i, c := range a {
		if c != b[i] {
			return false
		}
	}
	return true
}

func countBytePortable[T ~[]byte | ~string](s T, c byte) int {
	n := 0
	for i := 0; i < len(s); i++ {
		if s[i] == c {
			n++
		}
	}
	return n
}

func Count(b []byte, c byte) int       { return countBytePortable(b, c) }
func CountString(s string, c byte) int { return countBytePortable(s, c) }

func indexBytePortable[T ~[]byte | ~string](s T, c byte) int {
	for i := 0; i < len(s); i++ {
		if s[i] == c {
			return i
		}
	}
	return -1
}

func IndexByte(b []byte, c byte) int       { return indexBytePortable(b, c) }
func IndexByteString(s string, c byte) int { return indexBytePortable(s, c) }
