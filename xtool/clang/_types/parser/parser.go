/*
 * Copyright (c) 2022 The XGo Authors (xgo.dev). All rights reserved.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package parser

import (
	"errors"
	"fmt"
	"go/token"
	"go/types"
	"io"
	"log"
	"strconv"

	"github.com/goplus/gogen"
	"github.com/xgo-dev/llgo/xtool/clang/types/scanner"

	ctypes "github.com/xgo-dev/llgo/xtool/clang/types"
)

const (
	emsgDefArrWithoutLen = "define array without length"
)

var (
	ErrInvalidType = errors.New("invalid type")
)

type TypeNotFound struct {
	Literal       string
	StructOrUnion bool
}

func (p *TypeNotFound) Error() string {
	return fmt.Sprintf("type %s not found", p.Literal)
}

type ParseTypeError struct {
	QualType string
	ErrMsg   string
}

func (p *ParseTypeError) Error() string {
	return p.ErrMsg // TODO
}

func IsArrayWithoutLen(err error) bool {
	if e, ok := err.(*ParseTypeError); ok {
		return e.ErrMsg == emsgDefArrWithoutLen
	}
	return false
}

// -----------------------------------------------------------------------------

const (
	FlagIsParam = 1 << iota
	FlagIsStructField
	FlagIsExtern
	FlagIsTypedef
	FlagGetRetType
)

func getRetType(flags int) bool {
	return (flags & FlagGetRetType) != 0
}

type ParseEnv interface {
	Pkg() *types.Package
	Int128() types.Type
	Uint128() types.Type
}

type Config struct {
	ParseEnv
	Scope  *types.Scope
	Anonym types.Type
	Flags  int
}

const (
	KindFConst = 1 << iota
	KindFVolatile
	KindFAnonymous
	KindFVariadic
)

// qualType can be:
//   - unsigned int
//   - struct ConstantString
//   - volatile uint32_t
//   - int (*)(void *, int, char **, char **)
//   - int (*)(const char *, ...)
//   - int (*)(void)
//   - void (*(int, void (*)(int)))(int)
//   - const char *restrict
//   - const char [7]
//   - char *
//   - void
//   - ...
func ParseType(qualType string, conf *Config) (t types.Type, kind int, err error) {
	p := newParser(qualType, conf)
	if t, kind, err = p.parse(conf.Flags); err != nil {
		return
	}
	if p.tok != token.EOF {
		err = p.newError("unexpect token " + p.tok.String())
	}
	return
}

// -----------------------------------------------------------------------------

type parser struct {
	s     scanner.Scanner
	scope *types.Scope
	conf  *Config

	tok token.Token
	lit string
	old struct {
		tok token.Token
		lit string
	}
}

const (
	invalidTok token.Token = -1
)

func newParser(qualType string, conf *Config) *parser {
	p := &parser{scope: conf.Scope, conf: conf}
	p.old.tok = invalidTok
	p.s.Init(qualType)
	return p
}

func (p *parser) peek() token.Token {
	if p.old.tok == invalidTok {
		p.old.tok, p.old.lit = p.s.Scan()
	}
	return p.old.tok
}

func (p *parser) next() {
	if p.old.tok != invalidTok { // support unget
		p.tok, p.lit = p.old.tok, p.old.lit
		p.old.tok = invalidTok
		return
	}
	p.tok, p.lit = p.s.Scan()
}

func (p *parser) unget(tok token.Token, lit string) {
	p.old.tok, p.old.lit = p.tok, p.lit
	p.tok, p.lit = tok, lit
}

func (p *parser) skipUntil(tok token.Token) bool {
	for {
		p.next()
		switch p.tok {
		case tok:
			return true
		case token.EOF:
			return false
		}
	}
}

func (p *parser) newErrorf(format string, args ...interface{}) *ParseTypeError {
	return p.newError(fmt.Sprintf(format, args...))
}

func (p *parser) newError(errMsg string) *ParseTypeError {
	return &ParseTypeError{QualType: p.s.Source(), ErrMsg: errMsg}
}

// TODO(xsw): check expect results
func (p *parser) expect(tokExp token.Token) error {
	p.next()
	if p.tok != tokExp {
		return p.newErrorf("expect %v, got %v", tokExp, p.tok)
	}
	return nil
}

const (
	flagShort = 1 << iota
	flagLong
	flagLongLong
	flagUnsigned
	flagSigned
	flagComplex
	flagStructOrUnion
)

func (p *parser) lookupType(tylit string, flags int) (t types.Type, err error) {
	structOrUnion := (flags & flagStructOrUnion) != 0
	_, o := gogen.LookupParent(p.scope, tylit, token.NoPos)
	if o == nil {
		return nil, &TypeNotFound{Literal: tylit, StructOrUnion: structOrUnion}
	}
	t = o.Type()
	if !structOrUnion && flags != 0 {
		tt, ok := t.(*types.Basic)
		if !ok {
			tyInt128 := p.conf.Int128()
			if t == tyInt128 {
				switch flags {
				case flagSigned:
					return tyInt128, nil
				case flagUnsigned:
					return p.conf.Uint128(), nil
				}
			}
		} else if (flags & flagComplex) != 0 {
			switch tt.Kind() {
			case types.Float32:
				return types.Typ[types.Complex64], nil
			case types.Float64:
				return types.Typ[types.Complex128], nil
			case types.Int:
				return types.Typ[types.Complex128], nil
			}
		} else {
			switch tt.Kind() {
			case types.Int:
				if t = intTypes[flags&^flagSigned]; t != nil {
					return
				}
			case types.Int8:
				switch flags {
				case flagUnsigned:
					return types.Typ[types.Uint8], nil
				case flagSigned:
					return types.Typ[types.Int8], nil
				}
			case types.Float64:
				switch flags {
				case flagLong:
					return ctypes.LongDouble, nil
				}
			}
		}
		log.Panicln("lookupType: TODO - invalid type")
		return nil, ErrInvalidType
	}
	if t == types.Typ[types.Int] {
		return ctypes.Int, nil
	}
	return
}

var intTypes = [...]types.Type{
	0:                                      ctypes.Int,
	flagShort:                              types.Typ[types.Int16],
	flagLong:                               ctypes.Long,
	flagLong | flagLongLong:                types.Typ[types.Int64],
	flagUnsigned:                           ctypes.Uint,
	flagShort | flagUnsigned:               types.Typ[types.Uint16],
	flagLong | flagUnsigned:                ctypes.Ulong,
	flagLong | flagLongLong | flagUnsigned: types.Typ[types.Uint64],
	flagShort | flagLong | flagLongLong | flagUnsigned: nil,
}

func (p *parser) parseArray(t types.Type, inFlags int) (types.Type, error) {
	var n int64
	var err error
	p.next()
	switch p.tok {
	case token.RBRACK: // ]
		if (inFlags & FlagIsStructField) != 0 {
			n = 0
		} else {
			n = -1
		}
	case token.INT:
		if n, err = strconv.ParseInt(p.lit, 10, 64); err != nil {
			return nil, p.newError(err.Error())
		}
		if err = p.expect(token.RBRACK); err != nil { // ]
			return nil, err
		}
	default:
		return nil, p.newError("array length not an integer")
	}
	if n >= 0 || (inFlags&(FlagIsExtern|FlagIsTypedef|FlagIsParam)) != 0 {
		t = types.NewArray(t, n)
	} else {
		return nil, p.newError(emsgDefArrWithoutLen)
	}
	return t, nil
}

func (p *parser) parseArrays(t types.Type, inFlags int) (ret types.Type, err error) {
	if t == nil {
		return nil, p.newError("array to nil")
	}
	var tyArr types.Type
	for {
		if tyArr, err = p.parseArray(tyArr, inFlags); err != nil {
			return
		}
		if p.peek() != token.LBRACK {
			return newArraysEx(t, tyArr, inFlags), nil
		}
		p.next()
	}
}

func (p *parser) parseFunc(pkg *types.Package, t types.Type, inFlags int) (ret types.Type, err error) {
	var results *types.Tuple
	if ctypes.NotVoid(t) {
		results = types.NewTuple(types.NewParam(token.NoPos, pkg, "", t))
	}
	args, variadic, err := p.parseArgs(pkg)
	if err != nil {
		return
	}
	_ = inFlags
	return ctypes.NewFunc(types.NewTuple(args...), results, variadic), nil
}

func (p *parser) parseArgs(pkg *types.Package) (args []*types.Var, variadic bool, err error) {
	for {
		arg, kind, e := p.parse(FlagIsParam)
		if e != nil {
			return nil, false, e
		}
		if ctypes.NotVoid(arg) {
			args = append(args, types.NewParam(token.NoPos, pkg, "", arg))
		}
		if p.tok != token.COMMA {
			variadic = (kind & KindFVariadic) != 0
			break
		}
	}
	if p.tok != token.RPAREN { // )
		return nil, false, p.newError("expect )")
	}
	return
}

func (p *parser) parseStars() (nstar int) {
	for isPtr(p.peek()) {
		p.next()
		nstar++
	}
	return
}

func (p *parser) parse(inFlags int) (t types.Type, kind int, err error) {
	flags := 0
	for {
		p.next()
	retry:
		switch p.tok {
		case token.IDENT:
		ident:
			switch lit := p.lit; lit {
			case "unsigned":
				flags |= flagUnsigned
			case "short":
				flags |= flagShort
			case "long":
				if (flags & flagLong) != 0 {
					flags |= flagLongLong
				} else {
					flags |= flagLong
				}
			case "signed":
				flags |= flagSigned
			case "const":
				kind |= KindFConst
			case "volatile":
				kind |= KindFVolatile
			case "_Complex":
				flags |= flagComplex
			case "restrict", "_Nullable", "_Nonnull":
			case "enum":
				if err = p.expect(token.IDENT); err != nil {
					return
				}
				if t != nil {
					return nil, 0, p.newError("illegal syntax: multiple types?")
				}
				t = ctypes.Int
				continue
			case "struct", "union":
				p.next()
				switch p.tok {
				case token.IDENT:
				case token.LPAREN:
					if t == nil && p.conf.Anonym != nil {
						p.skipUntil(token.RPAREN)
						t = p.conf.Anonym
						kind |= KindFAnonymous
						continue
					}
					fallthrough
				default:
					log.Panicln("c.types.ParseType: struct/union - TODO:", p.lit)
				}
				lit = ctypes.MangledName(lit, p.lit)
				flags |= flagStructOrUnion
				fallthrough
			default:
				if t != nil {
					return nil, 0, p.newError("illegal syntax: multiple types?")
				}
				if t, err = p.lookupType(lit, flags); err != nil {
					return
				}
				flags = 0
			}
			if flags != 0 {
				p.next()
				if p.tok == token.IDENT {
					goto ident
				}
				if t != nil {
					return nil, 0, p.newError("illegal syntax: multiple types?")
				}
				if t, err = p.lookupType("int", flags); err != nil {
					return
				}
				flags = 0
				goto retry
			}
		case token.MUL: // *
			if t == nil {
				return nil, 0, p.newError("pointer to nil")
			}
			t = ctypes.NewPointer(t)
		case token.LBRACK: // [
			if t, err = p.parseArrays(t, inFlags); err != nil {
				return
			}
		case token.LPAREN: // (
			if t == nil {
				log.Panicln("TODO")
				return nil, 0, p.newError("no function return type")
			}
			var nstar = p.parseStars()
			var nstarRet int
			var tyArr types.Type
			var pkg, isFn = p.conf.Pkg(), false
			var args []*types.Var
			var variadic bool
			if nstar == 0 {
				if getRetType(inFlags) {
					err = nil
					p.tok = token.EOF
					return
				}
				if args, variadic, err = p.parseArgs(pkg); err != nil {
					return
				}
				isFn = true
			} else {
			nextTok:
				p.next()
				switch p.tok {
				case token.RPAREN: // )
				case token.LPAREN: // (
					if !isFn {
						nstar, nstarRet = p.parseStars(), nstar
						if nstar != 0 {
							p.expect(token.RPAREN) // )
							p.expect(token.LPAREN) // (
						}
						if args, variadic, err = p.parseArgs(pkg); err != nil {
							return
						}
						isFn = true
						goto nextTok
					}
					return nil, 0, p.newError("expect )")
				case token.LBRACK:
					if tyArr, err = p.parseArrays(ctypes.Void, 0); err != nil {
						return
					}
					p.expect(token.RPAREN) // )
				case token.IDENT:
					switch p.lit {
					case "_Nullable", "_Nonnull", "const", "volatile":
						goto nextTok
					}
					fallthrough
				default:
					return nil, 0, p.newError("expect )")
				}
			}
			p.next()
			switch p.tok {
			case token.LPAREN: // (
				if t, err = p.parseFunc(pkg, t, inFlags); err != nil {
					return
				}
			case token.LBRACK: // [
				if t, err = p.parseArrays(t, 0); err != nil {
					return
				}
			case token.EOF:
			case token.IDENT:
				if p.lit == "__attribute__" {
					p.tok, p.lit = token.EOF, ""
					p.unget(token.EOF, "")
					break
				}
				fallthrough
			default:
				return nil, 0, p.newError("unexpected " + p.tok.String())
			}
			t = newPointers(t, nstarRet)
			if isFn {
				if getRetType(inFlags) {
					p.tok = token.EOF
					return
				}
				var results *types.Tuple
				if ctypes.NotVoid(t) {
					results = types.NewTuple(types.NewParam(token.NoPos, pkg, "", t))
				}
				t = ctypes.NewFunc(types.NewTuple(args...), results, variadic)
			}
			t = newPointers(t, nstar)
			t = newArrays(t, tyArr)
		case token.RPAREN:
			if t == nil {
				t = ctypes.Void
			}
			return
		case token.COMMA, token.EOF:
			if t == nil {
				err = io.ErrUnexpectedEOF
			}
			return
		case token.ELLIPSIS:
			if t != nil {
				return nil, 0, p.newError("illegal syntax: multiple types?")
			}
			t = ctypes.Valist
			kind |= KindFVariadic
		default:
			log.Panicln("c.types.ParseType: unknown -", p.tok, p.lit)
		}
	}
}

func newPointers(t types.Type, nstar int) types.Type {
	for nstar > 0 {
		t = ctypes.NewPointer(t)
		nstar--
	}
	return t
}

func isPtr(tok token.Token) bool {
	return tok == token.MUL || tok == token.XOR // * or ^
}

func newArrays(t types.Type, tyArr types.Type) types.Type {
retry:
	if arr, ok := tyArr.(*types.Array); ok {
		t = types.NewArray(t, arr.Len())
		tyArr = arr.Elem()
		goto retry
	}
	return t
}

func newArraysEx(t types.Type, tyArr types.Type, inFlags int) types.Type {
	t = newArrays(t, tyArr)
	if arr, ok := t.(*types.Array); ok {
		if (inFlags & FlagIsParam) != 0 {
			t = ctypes.NewPointer(arr.Elem())
		}
	}
	return t
}

// -----------------------------------------------------------------------------
