//go:build llgo && llgo.size.json && goexperiment.jsonv2 && go1.26

package json

import (
	"encoding/json/jsontext"
	jsonv2 "encoding/json/v2"
	"io"
	"reflect"
	"strconv"
	"strings"
)

// LlgoDecodeTyped uses the buffered tokenizer and returns the public legacy
// errors without using the generic codec's callbacks or error registry.
func LlgoDecodeTyped[T any](data []byte, dst *T, decode func(*jsontext.Decoder, *T) error) error {
	if dst == nil {
		return &InvalidUnmarshalError{reflect.TypeOf(dst)}
	}
	err := jsonv2.LlgoReadTyped(data, dst, decode)
	if semantic, ok := err.(*jsonv2.SemanticError); ok {
		value := semantic.JSONKind.String()
		switch semantic.JSONKind {
		case 'f', 't':
			value = "bool"
		case '[', ']':
			value = "array"
		case '{', '}':
			value = "object"
		}
		cause := semantic.Err
		if len(semantic.JSONValue) != 0 {
			// The typed decoders include a value only for numeric conversion
			// errors. The legacy API reports the value and drops the cause.
			value += " " + string(semantic.JSONValue)
			if cause == strconv.ErrRange || cause == strconv.ErrSyntax {
				cause = nil
			}
		}
		var root string
		if semantic.JSONPointer != "" {
			root = reflect.TypeFor[T]().Name()
		}
		field := strings.ReplaceAll(strings.TrimPrefix(string(semantic.JSONPointer), "/"), "/", ".")
		return &UnmarshalTypeError{
			Value: value, Type: semantic.GoType, Offset: semantic.ByteOffset,
			Struct: root, Field: field, Err: llgoTypedSyntaxError(cause),
		}
	}
	return llgoTypedSyntaxError(err)
}

func llgoTypedSyntaxError(err error) error {
	syntax, ok := err.(*jsontext.SyntacticError)
	if !ok {
		return err
	}
	message := "unexpected end of JSON input"
	if syntax.Err != io.ErrUnexpectedEOF {
		message = syntax.Err.Error()
		if i := strings.Index(message, " (expecting"); i >= 0 && !strings.Contains(message, " in literal") {
			message = message[:i]
		}
		// Keep the legacy spellings and replacement order. Construct the
		// replacer on the error path so the parser needs no v1 package init.
		message = strings.NewReplacer(
			"object name", "object key",
			"at start of value", "looking for beginning of value",
			"at start of string", "looking for beginning of object key string",
			"after object value", "after object key:value pair",
			"in number", "in numeric literal",
		).Replace(message)
	}
	return &SyntaxError{Offset: syntax.ByteOffset, msg: message}
}

//llgointernal:template:bind LlgoDecodeTyped json op=fallback
