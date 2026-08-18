//go:build go1.25

package multipart_test

import (
	"mime"
	"mime/multipart"
	"testing"
)

func TestFileContentDisposition(t *testing.T) {
	value := multipart.FileContentDisposition("upload", `report "final".txt`)
	mediaType, params, err := mime.ParseMediaType(value)
	if err != nil {
		t.Fatal(err)
	}
	if mediaType != "form-data" || params["name"] != "upload" || params["filename"] != `report "final".txt` {
		t.Fatalf("FileContentDisposition = %q, parsed as %q, %#v", value, mediaType, params)
	}
}
