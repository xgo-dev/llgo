package abi_test

import (
	"fmt"
	"testing"

	"go/types"
)

const (
	MAXKEYSIZE  = 128
	MAXELEMSIZE = 128
	BUCKETSIZE  = 8
)

type mockSizes struct {
	alignof func(types.Type) int64
	sizeof  func(types.Type) int64
}

func (m mockSizes) Alignof(t types.Type) int64 { return m.alignof(t) }
func (m mockSizes) Sizeof(t types.Type) int64  { return m.sizeof(t) }
func (m mockSizes) Offsetsof(f []*types.Var) []int64 {
	// Mock implementation for testing alignment checks
	return []int64{0, 8, 16} // Typical offsets for bucket fields
}

// isPointer reports whether t is a pointer type.
func isPointer(t types.Type) (ok bool) {
	_, ok = t.Underlying().(*types.Pointer)
	return
}

// MapTypeFlags computes the flags for the given map type.
func MapTypeFlags(t *types.Map, sizes types.Sizes) (flags int) {
	if sizes.Sizeof(t.Key()) > MAXKEYSIZE {
		flags |= 1 // indirect key
	}
	if sizes.Sizeof(t.Elem()) > MAXELEMSIZE {
		flags |= 2 // indirect value
	}
	return
}

// MapBucketType returns the bucket type for a map.
func MapBucketType(t *types.Map, sizes types.Sizes) *types.Struct {
	keytype := t.Key()
	elemtype := t.Elem()
	bucket := types.NewStruct([]*types.Var{}, nil)

	if !types.Comparable(keytype) {
		log.Fatalf("unsupported map key type for %v", t)
	}
	if BUCKETSIZE < 8 {
		log.Fatalf("bucket size %d too small for proper alignment %d", BUCKETSIZE, 8)
	}
	if uint8(sizes.Alignof(keytype)) > BUCKETSIZE {
		log.Fatalf("key align too big for %v", t)
	}
	if uint8(sizes.Alignof(elemtype)) > BUCKETSIZE {
		log.Fatalf("elem align %d too big for %v, BUCKETSIZE=%d", sizes.Alignof(elemtype), t, BUCKETSIZE)
	}
	if sizes.Alignof(keytype) > MAXKEYSIZE {
		log.Fatalf("key align too big for %v", t)
	}
	if sizes.Alignof(elemtype) > MAXELEMSIZE {
		log.Fatalf("elem align too big for %v", t)
	}
	if sizes.Alignof(keytype) > MAXKEYSIZE && !isPointer(keytype) {
		log.Fatalf("key indirect incorrect for %v", t)
	}
	if sizes.Alignof(elemtype) > MAXELEMSIZE && !isPointer(elemtype) {
		log.Fatalf("elem indirect incorrect for %v", t)
	}
	if sizes.Sizeof(keytype)%sizes.Alignof(keytype) != 0 {
		log.Fatalf("key size not a multiple of key align for %v", t)
	}
	if sizes.Sizeof(elemtype)%sizes.Alignof(elemtype) != 0 {
		log.Fatalf("elem size not a multiple of elem align for %v", t)
	}
	if uint8(sizes.Alignof(bucket))%uint8(sizes.Alignof(keytype)) != 0 {
		log.Fatalf("bucket align not multiple of key align %v", t)
	}
	if uint8(sizes.Alignof(bucket))%uint8(sizes.Alignof(elemtype)) != 0 {
		log.Fatalf("bucket align not multiple of elem align %v", t)
	}
	offs := sizes.Offsetsof(nil)
	if len(offs) >= 3 && offs[1]%sizes.Alignof(keytype) != 0 {
		log.Fatalf("bad alignment of keys in bmap for %v", t)
	}
	if len(offs) >= 3 && offs[2]%sizes.Alignof(elemtype) != 0 {
		log.Fatalf("bad alignment of elems in bmap for %v", t)
	}

	return bucket
}

// logger interface for testing
type logger interface {
	Fatalf(format string, args ...interface{})
}

var log logger = &defaultLogger{}

type defaultLogger struct{}

func (l *defaultLogger) Fatalf(format string, args ...interface{}) {
	panic(fmt.Sprintf(format, args...))
}

type testLogger struct {
	t *testing.T
}

func (l *testLogger) Fatalf(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	panic(msg)
}

func TestIsPointer(t *testing.T) {
	tests := []struct {
		name string
		typ  types.Type
		want bool
	}{
		{
			name: "pointer type",
			typ:  types.NewPointer(types.Typ[types.Int]),
			want: true,
		},
		{
			name: "non-pointer type",
			typ:  types.Typ[types.Int],
			want: false,
		},
		{
			name: "pointer to struct",
			typ:  types.NewPointer(types.NewStruct([]*types.Var{}, nil)),
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isPointer(tt.typ); got != tt.want {
				t.Errorf("isPointer() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMapTypeFlags(t *testing.T) {
	tests := []struct {
		name     string
		key      types.Type
		elem     types.Type
		sizeFn   func(types.Type) int64
		alignFn  func(types.Type) int64
		wantFlag int
	}{
		{
			name: "small key and elem",
			key:  types.Typ[types.Int],
			elem: types.Typ[types.Int],
			sizeFn: func(t types.Type) int64 {
				return 8 // 64-bit integers
			},
			alignFn: func(t types.Type) int64 {
				return 8
			},
			wantFlag: 0,
		},
		{
			name: "large key",
			key:  types.NewArray(types.Typ[types.Int64], 20), // 160 bytes
			elem: types.Typ[types.Int],
			sizeFn: func(t types.Type) int64 {
				if _, ok := t.(*types.Array); ok {
					return 160
				}
				return 8
			},
			alignFn: func(t types.Type) int64 {
				return 8
			},
			wantFlag: 1, // indirect key
		},
		{
			name: "large elem",
			key:  types.Typ[types.Int],
			elem: types.NewArray(types.Typ[types.Int64], 20), // 160 bytes
			sizeFn: func(t types.Type) int64 {
				if _, ok := t.(*types.Array); ok {
					return 160
				}
				return 8
			},
			alignFn: func(t types.Type) int64 {
				return 8
			},
			wantFlag: 2, // indirect elem
		},
		{
			name: "large key and elem",
			key:  types.NewArray(types.Typ[types.Int64], 20), // 160 bytes
			elem: types.NewArray(types.Typ[types.Int64], 20), // 160 bytes
			sizeFn: func(t types.Type) int64 {
				if _, ok := t.(*types.Array); ok {
					return 160
				}
				return 8
			},
			alignFn: func(t types.Type) int64 {
				return 8
			},
			wantFlag: 3, // both indirect
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sizes := &mockSizes{
				sizeof:  tt.sizeFn,
				alignof: tt.alignFn,
			}
			mapType := types.NewMap(tt.key, tt.elem)
			if got := MapTypeFlags(mapType, sizes); got != tt.wantFlag {
				t.Errorf("MapTypeFlags() = %v, want %v", got, tt.wantFlag)
			}
		})
	}
}

func TestMapTypeChecks(t *testing.T) {
	origLog := log
	defer func() { log = origLog }()

	tests := []struct {
		name    string
		setup   func() (*types.Map, types.Sizes)
		wantErr bool
	}{
		{
			name: "non-comparable key type",
			setup: func() (*types.Map, types.Sizes) {
				key := types.NewSlice(types.Typ[types.Int])
				elem := types.Typ[types.Int]
				return types.NewMap(key, elem), &mockSizes{
					sizeof:  func(t types.Type) int64 { return 8 },
					alignof: func(t types.Type) int64 { return 8 },
				}
			},
			wantErr: true,
		},
		{
			name: "key align too big",
			setup: func() (*types.Map, types.Sizes) {
				return types.NewMap(types.Typ[types.Int], types.Typ[types.Int]), &mockSizes{
					sizeof: func(t types.Type) int64 { return 8 },
					alignof: func(t types.Type) int64 {
						if _, ok := t.(*types.Basic); ok && t.String() == "int" {
							return BUCKETSIZE + 1
						}
						return 8
					},
				}
			},
			wantErr: true,
		},
		{
			name: "key size not multiple of align",
			setup: func() (*types.Map, types.Sizes) {
				return types.NewMap(types.Typ[types.Int], types.Typ[types.Int]), &mockSizes{
					sizeof: func(t types.Type) int64 {
						if _, ok := t.(*types.Basic); ok && t.String() == "int" {
							return 7
						}
						return 8
					},
					alignof: func(t types.Type) int64 {
						if _, ok := t.(*types.Basic); ok && t.String() == "int" {
							return 4
						}
						return 8
					},
				}
			},
			wantErr: true,
		},
		{
			name: "elem align too big",
			setup: func() (*types.Map, types.Sizes) {
				return types.NewMap(types.Typ[types.Int], types.Typ[types.Int]), &mockSizes{
					sizeof: func(t types.Type) int64 { return 8 },
					alignof: func(t types.Type) int64 {
						if _, ok := t.(*types.Basic); ok && t.String() == "int" {
							return 4
						}
						return BUCKETSIZE + 1
					},
				}
			},
			wantErr: true,
		},
		{
			name: "valid map type",
			setup: func() (*types.Map, types.Sizes) {
				return types.NewMap(types.Typ[types.Int], types.Typ[types.Int]), &mockSizes{
					sizeof:  func(t types.Type) int64 { return 8 },
					alignof: func(t types.Type) int64 { return 8 },
				}
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			log = &testLogger{t: t}
			defer func() {
				if r := recover(); r != nil {
					if !tt.wantErr {
						t.Errorf("MapBucketType() panicked unexpectedly: %v", r)
					}
				} else if tt.wantErr {
					t.Error("expected panic but got none")
				}
			}()

			mapType, sizes := tt.setup()
			MapBucketType(mapType, sizes)
		})
	}
}
