//go:build llgo && (amd64 || 386 || arm64 || arm || ppc64 || ppc64le || riscv64) && !baremetal && go1.27 && !go1.28

package aes_test

import (
	"crypto/aes"
	"crypto/cipher"
	"testing"
	_ "unsafe"
)

//go:linkname aesFeaturesEnabled crypto/internal/fips140/aes.llgoAESFeaturesEnabled
func aesFeaturesEnabled(bool, string) bool

func TestLLVMFeatureDispatch(t *testing.T) {
	for _, tc := range []struct {
		env  string
		want bool
	}{
		{"", true},
		{"cpu.aes=off", false},
		{"cpu.all=off", false},
		{"cpu.all=off,cpu.aes=on", true},
		{"cpu.aes=on,cpu.all=off", false},
		{"cpu.aes=off,cpu.all=on", true},
		{"cpu.aes=invalid,cpu.other=off", true},
		{",cpu.aes=off,", false},
	} {
		if got := aesFeaturesEnabled(true, tc.env); got != tc.want {
			t.Errorf("hardware enabled, GODEBUG=%q: got %v, want %v", tc.env, got, tc.want)
		}
		if aesFeaturesEnabled(false, tc.env) {
			t.Errorf("GODEBUG=%q enabled instructions missing on the CPU", tc.env)
		}
	}
}

func TestLLVMKernelDoesNotAllocate(t *testing.T) {
	block, err := aes.NewCipher(make([]byte, 32))
	if err != nil {
		t.Fatal(err)
	}
	buf := make([]byte, 16<<10)
	stream := cipher.NewCTR(block, make([]byte, 16))
	if got := testing.AllocsPerRun(100, func() {
		block.Encrypt(buf[:16], buf[:16])
		block.Decrypt(buf[:16], buf[:16])
		stream.XORKeyStream(buf, buf)
	}); got != 0 {
		t.Fatalf("AES kernels allocated %.0f objects per operation", got)
	}
}
