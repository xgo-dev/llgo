/*
 * Copyright (c) 2024 The XGo Authors (xgo.dev). All rights reserved.
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

package math

import (
	_ "unsafe"

	c "github.com/xgo-dev/llgo/runtime/internal/clite"
)

const (
	LLGoPackage = "decl"
)

// -----------------------------------------------------------------------------

//go:linkname Acos C.acos
func Acos(x float64) float64

//go:linkname Acosh C.acosh
func Acosh(x float64) float64

//go:linkname Asin C.asin
func Asin(x float64) float64

//go:linkname Asinh C.asinh
func Asinh(x float64) float64

//go:linkname Atan C.atan
func Atan(x float64) float64

//go:linkname Atan2 C.atan2
func Atan2(y, x float64) float64

//go:linkname Atanh C.atanh
func Atanh(x float64) float64

//go:linkname Cbrt C.cbrt
func Cbrt(x float64) float64

//go:linkname Ceil C.ceil
func Ceil(x float64) float64

//go:linkname Cos C.cos
func Cos(x float64) float64

//go:linkname Cosh C.cosh
func Cosh(x float64) float64

//go:linkname Copysign C.copysign
func Copysign(x, y float64) float64

//go:linkname Erf C.erf
func Erf(x float64) float64

//go:linkname Erfc C.erfc
func Erfc(x float64) float64

//go:linkname Exp C.exp
func Exp(x float64) float64

//go:linkname Exp2 C.exp2
func Exp2(x float64) float64

//go:linkname Expm1 C.expm1
func Expm1(x float64) float64

//go:linkname Fdim C.fdim
func Fdim(x, y float64) float64

//go:linkname Floor C.floor
func Floor(x float64) float64

//go:linkname Fma C.fma
func Fma(x, y, z float64) float64

//go:linkname Fmax C.fmax
func Fmax(x, y float64) float64

//go:linkname Fmin C.fmin
func Fmin(x, y float64) float64

//go:linkname Fmod C.fmod
func Fmod(x, y float64) float64

//go:linkname Frexp C.frexp
func Frexp(x float64, exp *c.Int) float64

//go:linkname Gamma C.gamma
func Gamma(x float64) float64

//go:linkname Hypot C.hypot
func Hypot(x, y float64) float64

//go:linkname Ilogb C.ilogb
func Ilogb(x float64) c.Int

//go:linkname J0 C.j0
func J0(x float64) float64

//go:linkname J1 C.j1
func J1(x float64) float64

//go:linkname Jn C.jn
func Jn(n c.Int, x float64) float64

//go:linkname Ldexp C.ldexp
func Ldexp(x float64, exp c.Int) float64

//go:linkname Lgamma C.lgamma
func Lgamma(x float64) float64

//go:linkname Log C.log
func Log(x float64) float64

//go:linkname Log10 C.log10
func Log10(x float64) float64

//go:linkname Log1p C.log1p
func Log1p(x float64) float64

//go:linkname Log2 C.log2
func Log2(x float64) float64

//go:linkname Logb C.logb
func Logb(x float64) float64

//go:linkname Modf C.modf
func Modf(x float64, ipart *float64) float64

//go:linkname Nan C.nan
func Nan(tag *c.Char) float64

//go:linkname Nextafter C.nextafter
func Nextafter(x, y float64) float64

//go:linkname Pow C.pow
func Pow(x, y float64) float64

//go:linkname Remainder C.remainder
func Remainder(x, y float64) float64

//go:linkname Round C.round
func Round(x float64) float64

//go:linkname Sin C.sin
func Sin(x float64) float64

//go:linkname Sinh C.sinh
func Sinh(x float64) float64

//go:linkname Sqrt C.sqrt
func Sqrt(x float64) float64

//go:linkname Tan C.tan
func Tan(x float64) float64

//go:linkname Tanh C.tanh
func Tanh(x float64) float64

//go:linkname Tgamma C.tgamma
func Tgamma(x float64) float64

//go:linkname Trunc C.trunc
func Trunc(x float64) float64

// -----------------------------------------------------------------------------

//go:linkname Acosf C.acosf
func Acosf(x float32) float32

//go:linkname Acoshf C.acoshf
func Acoshf(x float32) float32

//go:linkname Asinf C.asinf
func Asinf(x float32) float32

//go:linkname Asinhf C.asinhf
func Asinhf(x float32) float32

//go:linkname Atanf C.atanf
func Atanf(x float32) float32

//go:linkname Atan2f C.atan2f
func Atan2f(y, x float32) float32

//go:linkname Atanhf C.atanhf
func Atanhf(x float32) float32

//go:linkname Cbrtf C.cbrtf
func Cbrtf(x float32) float32

//go:linkname Ceilf C.ceilf
func Ceilf(x float32) float32

//go:linkname Cosf C.cosf
func Cosf(x float32) float32

//go:linkname Coshf C.coshf
func Coshf(x float32) float32

//go:linkname Copysignf C.copysignf
func Copysignf(x, y float32) float32

//go:linkname Erff C.erff
func Erff(x float32) float32

//go:linkname Erfcf C.erfcf
func Erfcf(x float32) float32

//go:linkname Expf C.expf
func Expf(x float32) float32

//go:linkname Exp2f C.exp2f
func Exp2f(x float32) float32

//go:linkname Expm1f C.expm1f
func Expm1f(x float32) float32

//go:linkname Fdimf C.fdimf
func Fdimf(x, y float32) float32

//go:linkname Floorf C.floorf
func Floorf(x float32) float32

//go:linkname Fmaf C.fmaf
func Fmaf(x, y, z float32) float32

//go:linkname Fmaxf C.fmaxf
func Fmaxf(x, y float32) float32

//go:linkname Fminf C.fminf
func Fminf(x, y float32) float32

//go:linkname Fmodf C.fmodf
func Fmodf(x, y float32) float32

//go:linkname Frexpf C.frexpf
func Frexpf(x float32, exp *c.Int) float32

//go:linkname Gammaf C.gammaf
func Gammaf(x float32) float32

//go:linkname Hypotf C.hypotf
func Hypotf(x, y float32) float32

//go:linkname Ilogbf C.ilogbf
func Ilogbf(x float32) c.Int

//go:linkname J0f C.j0f
func J0f(x float32) float32

//go:linkname J1f C.j1f
func J1f(x float32) float32

//go:linkname Jnf C.jnf
func Jnf(n c.Int, x float32) float32

//go:linkname Ldexpf C.ldexpf
func Ldexpf(x float32, exp c.Int) float32

//go:linkname Lgammaf C.lgammaf
func Lgammaf(x float32) float32

//go:linkname Logf C.logf
func Logf(x float32) float32

//go:linkname Log10f C.log10f
func Log10f(x float32) float32

//go:linkname Log1pf C.log1pf
func Log1pf(x float32) float32

//go:linkname Log2f C.log2f
func Log2f(x float32) float32

//go:linkname Logbf C.logbf
func Logbf(x float32) float32

//go:linkname Modff C.modff
func Modff(x float32, ipart *float32) float32

//go:linkname Nanf C.nanf
func Nanf(tag *c.Char) float32

//go:linkname Nextafterf C.nextafterf
func Nextafterf(x, y float32) float32

//go:linkname Powf C.powf
func Powf(x, y float32) float32

//go:linkname Remainderf C.remainderf
func Remainderf(x, y float32) float32

//go:linkname Roundf C.roundf
func Roundf(x float32) float32

//go:linkname Sinf C.sinf
func Sinf(x float32) float32

//go:linkname Sinhf C.sinhf
func Sinhf(x float32) float32

//go:linkname Sqrtf C.sqrtf
func Sqrtf(x float32) float32

//go:linkname Tanf C.tanf
func Tanf(x float32) float32

//go:linkname Tanhf C.tanhf
func Tanhf(x float32) float32

//go:linkname Tgammaf C.tgammaf
func Tgammaf(x float32) float32

//go:linkname Truncf C.truncf
func Truncf(x float32) float32

// -----------------------------------------------------------------------------
