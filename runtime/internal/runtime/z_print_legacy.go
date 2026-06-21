//go:build !go1.26

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

import c "github.com/goplus/llgo/runtime/internal/clite"

func formatFloat(v float64) string {
	if s, ok := formatSpecialFloat(v); ok {
		return s
	}
	return padLegacyPrintExponent(formatFloatWithC(v, c.Str("%+.6e")))
}

func formatComplex(v complex128) string {
	return "(" + formatFloat(real(v)) + formatFloat(imag(v)) + "i)"
}

// Go's legacy printfloat path pads exponents to three digits; C's %e commonly
// pads to two, so normalize after using C for the mantissa formatting.
func padLegacyPrintExponent(s string) string {
	for i := len(s) - 1; i >= 0; i-- {
		if s[i] != 'e' && s[i] != 'E' {
			continue
		}
		exp := i + 2
		if i+1 >= len(s) || exp >= len(s) || (s[i+1] != '+' && s[i+1] != '-') {
			return s
		}
		switch digits := len(s) - exp; digits {
		case 1:
			return s[:exp] + "00" + s[exp:]
		case 2:
			return s[:exp] + "0" + s[exp:]
		default:
			return s
		}
	}
	return s
}
