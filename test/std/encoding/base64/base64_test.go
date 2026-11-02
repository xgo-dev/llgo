package base64_test

import (
	"bytes"
	"encoding/base32"
	"encoding/base64"
	"errors"
	"io"
	"strings"
	"testing"
)

func TestStdEncodingRoundTrip(t *testing.T) {
	src := []byte("goplus-base64-test")

	encLen := base64.StdEncoding.EncodedLen(len(src))
	if encLen%4 != 0 {
		t.Fatalf("EncodedLen should be multiple of 4, got %d", encLen)
	}

	encoded := base64.StdEncoding.EncodeToString(src)
	if encoded != base64.StdEncoding.EncodeToString(src) {
		t.Fatalf("EncodeToString not deterministic: %q", encoded)
	}

	if base64.StdEncoding.DecodedLen(len(encoded)) < len(src) {
		t.Fatalf("DecodedLen too small: %d", base64.StdEncoding.DecodedLen(len(encoded)))
	}

	decoded, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		t.Fatalf("DecodeString error: %v", err)
	}
	if !bytes.Equal(decoded, src) {
		t.Fatalf("DecodeString mismatch: got %q want %q", decoded, src)
	}

	buf := make([]byte, base64.StdEncoding.EncodedLen(len(src)))
	base64.StdEncoding.Encode(buf, src)
	if string(buf) != encoded {
		t.Fatalf("Encode mismatch: %q", buf)
	}

	decodeBuf := make([]byte, base64.StdEncoding.DecodedLen(len(buf)))
	n, err := base64.StdEncoding.Decode(decodeBuf, buf)
	if err != nil {
		t.Fatalf("Decode error: %v", err)
	}
	if !bytes.Equal(decodeBuf[:n], src) {
		t.Fatalf("Decode result mismatch: %q", decodeBuf[:n])
	}
}

func TestStreamingEncoderDecoder(t *testing.T) {
	data := []byte("streaming-base64")
	var encoded bytes.Buffer

	encWriter := base64.NewEncoder(base64.StdEncoding, &encoded)
	if _, err := encWriter.Write(data[:5]); err != nil {
		t.Fatalf("Write error: %v", err)
	}
	if _, err := encWriter.Write(data[5:]); err != nil {
		t.Fatalf("Write error: %v", err)
	}
	if err := encWriter.Close(); err != nil {
		t.Fatalf("Close error: %v", err)
	}

	decReader := base64.NewDecoder(base64.StdEncoding, &encoded)
	out, err := io.ReadAll(decReader)
	if err != nil {
		t.Fatalf("ReadAll error: %v", err)
	}
	if !bytes.Equal(out, data) {
		t.Fatalf("decoded data mismatch: %q", out)
	}
}

func TestPredefinedEncodings(t *testing.T) {
	sample := []byte{0xff, 0x00, 0x01}

	std := base64.StdEncoding.EncodeToString(sample)
	if len(std)%4 != 0 {
		t.Fatalf("StdEncoding output length not multiple of 4: %d", len(std))
	}
	if decoded, err := base64.StdEncoding.DecodeString(std); err != nil || !bytes.Equal(decoded, sample) {
		t.Fatalf("StdEncoding round-trip failed: data=%q err=%v", decoded, err)
	}

	rawStd := base64.RawStdEncoding.EncodeToString(sample)
	if strings.ContainsRune(rawStd, base64.StdPadding) {
		t.Fatalf("RawStdEncoding should not contain padding: %s", rawStd)
	}
	if decoded, err := base64.RawStdEncoding.DecodeString(rawStd); err != nil || !bytes.Equal(decoded, sample) {
		t.Fatalf("RawStdEncoding round-trip failed: data=%q err=%v", decoded, err)
	}

	urlEnc := base64.URLEncoding.EncodeToString(sample)
	if strings.Contains(urlEnc, "+") || strings.Contains(urlEnc, "/") {
		t.Fatalf("URLEncoding should not use +/ characters: %s", urlEnc)
	}
	if decoded, err := base64.URLEncoding.DecodeString(urlEnc); err != nil || !bytes.Equal(decoded, sample) {
		t.Fatalf("URLEncoding round-trip failed: data=%q err=%v", decoded, err)
	}

	rawURL := base64.RawURLEncoding.EncodeToString(sample)
	if strings.ContainsRune(rawURL, base64.StdPadding) {
		t.Fatalf("RawURLEncoding should not include padding: %s", rawURL)
	}
	if decoded, err := base64.RawURLEncoding.DecodeString(rawURL); err != nil || !bytes.Equal(decoded, sample) {
		t.Fatalf("RawURLEncoding round-trip failed: data=%q err=%v", decoded, err)
	}
}

func TestAppendEncodeDecode(t *testing.T) {
	src := []byte("append encoding")
	dst := base64.StdEncoding.AppendEncode(nil, src)
	if len(dst) != base64.StdEncoding.EncodedLen(len(src)) {
		t.Fatalf("AppendEncode length mismatch: %d", len(dst))
	}

	decoded, err := base64.StdEncoding.AppendDecode(nil, dst)
	if err != nil {
		t.Fatalf("AppendDecode error: %v", err)
	}
	if string(decoded) != string(src) {
		t.Fatalf("AppendDecode mismatch: %q", decoded)
	}
}

func TestStrictDecoding(t *testing.T) {
	strict := base64.StdEncoding.Strict()
	good := base64.StdEncoding.EncodeToString([]byte{0xff})
	if _, err := strict.DecodeString(good); err != nil {
		t.Fatalf("Strict DecodeString failed: %v", err)
	}

	bad := good[:len(good)-1] + "A"
	_, err := strict.DecodeString(bad)
	if err == nil {
		t.Fatal("Strict decoding should fail on invalid padding bits")
	}
	var corrupt base64.CorruptInputError
	if !errors.As(err, &corrupt) {
		t.Fatalf("expected CorruptInputError, got %T", err)
	}
}

func TestWithPadding(t *testing.T) {
	custom := base64.StdEncoding.WithPadding('*')
	padded := custom.EncodeToString([]byte("go"))
	if !strings.ContainsRune(padded, '*') {
		t.Fatalf("custom padding not used: %s", padded)
	}

	noPad := base64.StdEncoding.WithPadding(base64.NoPadding)
	out := noPad.EncodeToString([]byte("go"))
	if strings.ContainsRune(out, base64.StdPadding) {
		t.Fatalf("no padding encoding should omit '=': %s", out)
	}
}

func TestNewEncoding(t *testing.T) {
	alphabet := "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz+/"
	enc := base64.NewEncoding(alphabet)
	data := []byte("custom")
	encoded := enc.EncodeToString(data)
	decoded, err := enc.DecodeString(encoded)
	if err != nil {
		t.Fatalf("DecodeString error: %v", err)
	}
	if !bytes.Equal(decoded, data) {
		t.Fatalf("round-trip mismatch: %q", decoded)
	}
}

func TestCorruptInputError(t *testing.T) {
	_, err := base64.StdEncoding.DecodeString("Zm9v?")
	if err == nil {
		t.Fatal("expected corrupt input error")
	}
	var corrupt base64.CorruptInputError
	if !errors.As(err, &corrupt) {
		t.Fatalf("expected CorruptInputError, got %T", err)
	}
	if corrupt.Error() == "" {
		t.Fatal("CorruptInputError.Error should return message")
	}
}

var boundAppendBase32 = base32.StdEncoding.AppendEncode
var boundAppendBase64 = base64.StdEncoding.AppendEncode
var boundLenBase32 = base32.StdEncoding.EncodedLen
var boundLenBase64 = base64.StdEncoding.EncodedLen

func TestBoundMethodsAcrossEncodingPackages(t *testing.T) {
	src := []byte{13}
	for _, test := range []struct {
		name   string
		encode func([]byte, []byte) []byte
		length func(int) int
		want   string
	}{
		{"base32", boundAppendBase32, boundLenBase32, "BU======"},
		{"base64", boundAppendBase64, boundLenBase64, "DQ=="},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := test.encode(nil, src)
			if string(got) != test.want || test.length(len(src)) != len(test.want) {
				t.Fatalf("got %q, length %d; want %q", got, test.length(len(src)), test.want)
			}
		})
	}
	local32, local64 := base32.StdEncoding.AppendEncode, base64.StdEncoding.AppendEncode
	if string(local32(nil, src)) != "BU======" || string(local64(nil, src)) != "DQ==" {
		t.Fatal("local method values lost receiver package")
	}
}
