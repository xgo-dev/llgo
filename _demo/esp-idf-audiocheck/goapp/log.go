package main

import (
	"github.com/xgo-dev/llgo/pkg/esp/esperr"
	"github.com/xgo-dev/llgo/pkg/esp/opus"
)

// errText renders a demo failure the way a console log wants it: the
// family's "NAME (code)" plus, for the opus package's own vocabulary, the
// plain-language sentence (ErrText) behind the code - the same rendering
// the esp-idf-opus and esp-idf-voice-loop demos print.
func errText(err error) string {
	e, ok := err.(*esperr.Error)
	if !ok {
		return err.Error()
	}
	text := opus.ErrText(e.Code)
	if text != "" && text != opus.ErrName(e.Code) {
		return e.Error() + ": " + text
	}
	return e.Error()
}
