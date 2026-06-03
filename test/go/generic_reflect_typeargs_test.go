package gotest

import (
	"reflect"
	"testing"

	"github.com/goplus/llgo/test/go/typeargpkg"
)

type genericReflectTuple[T any] struct {
	v T
}

type genericReflectFuture[T any] struct {
	v T
}

func TestGenericReflectTypeArgString(t *testing.T) {
	cases := []struct {
		name string
		got  string
		want string
	}{
		{
			name: "nested same-package type arg",
			got:  reflect.TypeOf(genericReflectFuture[genericReflectTuple[error]]{}).String(),
			want: "gotest.genericReflectFuture[github.com/goplus/llgo/test/go.genericReflectTuple[error]]",
		},
		{
			name: "cross-package type arg",
			got:  reflect.TypeOf(genericReflectFuture[typeargpkg.Item]{}).String(),
			want: "gotest.genericReflectFuture[github.com/goplus/llgo/test/go/typeargpkg.Item]",
		},
		{
			name: "composite type arg",
			got:  reflect.TypeOf(genericReflectFuture[*genericReflectTuple[error]]{}).String(),
			want: "gotest.genericReflectFuture[*github.com/goplus/llgo/test/go.genericReflectTuple[error]]",
		},
	}

	for _, tc := range cases {
		if tc.got != tc.want {
			t.Fatalf("%s: Type.String() = %q, want %q", tc.name, tc.got, tc.want)
		}
	}
}
