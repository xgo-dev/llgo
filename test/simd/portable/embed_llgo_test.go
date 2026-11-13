//go:build goexperiment.simd && llgo && (amd64 || arm64 || wasm)

package portable

import (
	_ "embed"
	"simd"
	"testing"
)

type embeddedText string
type embeddedBytes []byte

func (embeddedText) Width() int  { var v simd.Uint32s; return v.Len() }
func (embeddedBytes) Width() int { var v simd.Uint32s; return v.Len() }

//go:embed testdata/embedded.txt
var midwayEmbeddedText embeddedText

var (
	//go:embed testdata/embedded.txt
	midwayEmbeddedBytes embeddedBytes
)

var embeddedAtInit = string(midwayEmbeddedText) + string(midwayEmbeddedBytes)

func TestPortableEmbeddedGlobals(t *testing.T) {
	const want = "embedded data"
	if string(midwayEmbeddedText) != want || string(midwayEmbeddedBytes) != want || embeddedAtInit != want+want {
		t.Fatalf("embedded globals: text=%q bytes=%q init=%q", midwayEmbeddedText, midwayEmbeddedBytes, embeddedAtInit)
	}
	if midwayEmbeddedText.Width() != simd.VectorBitSize()/32 || midwayEmbeddedBytes.Width() != simd.VectorBitSize()/32 {
		t.Fatal("unexpected SIMD width")
	}
}
