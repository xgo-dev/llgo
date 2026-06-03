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

package runtime

var complexDivZero float64

func complexDivAbs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}

func complexDivIsInf(x float64) bool {
	return x != 0 && x+x == x
}

func complexDivIsNaN(x float64) bool {
	return x != x
}

func complexDivIsFinite(x float64) bool {
	return !complexDivIsNaN(x) && !complexDivIsInf(x)
}

func complexDivInf() float64 {
	return 1 / complexDivZero
}

func complexDivCopysign(x, y float64) float64 {
	if y < 0 || y == 0 && 1/y < 0 {
		return -x
	}
	return x
}

func complexDivInf2one(x float64) float64 {
	r := 0.0
	if complexDivIsInf(x) {
		r = 1.0
	}
	return complexDivCopysign(r, x)
}

func Complex128Div(n, m complex128) complex128 {
	var e, f float64

	if complexDivAbs(real(m)) >= complexDivAbs(imag(m)) {
		ratio := imag(m) / real(m)
		denom := real(m) + ratio*imag(m)
		e = (real(n) + imag(n)*ratio) / denom
		f = (imag(n) - real(n)*ratio) / denom
	} else {
		ratio := real(m) / imag(m)
		denom := imag(m) + ratio*real(m)
		e = (real(n)*ratio + imag(n)) / denom
		f = (imag(n)*ratio - real(n)) / denom
	}

	if complexDivIsNaN(e) && complexDivIsNaN(f) {
		a, b := real(n), imag(n)
		c, d := real(m), imag(m)
		inf := complexDivInf()
		switch {
		case m == 0 && (!complexDivIsNaN(a) || !complexDivIsNaN(b)):
			e = complexDivCopysign(inf, c) * a
			f = complexDivCopysign(inf, c) * b
		case (complexDivIsInf(a) || complexDivIsInf(b)) && complexDivIsFinite(c) && complexDivIsFinite(d):
			a = complexDivInf2one(a)
			b = complexDivInf2one(b)
			e = inf * (a*c + b*d)
			f = inf * (b*c - a*d)
		case (complexDivIsInf(c) || complexDivIsInf(d)) && complexDivIsFinite(a) && complexDivIsFinite(b):
			c = complexDivInf2one(c)
			d = complexDivInf2one(d)
			e = 0 * (a*c + b*d)
			f = 0 * (b*c - a*d)
		}
	}

	return complex(e, f)
}
