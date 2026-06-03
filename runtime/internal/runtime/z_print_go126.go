//go:build go1.26

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
	return formatFloatWithC(v, c.Str("%g"))
}

func formatComplex(v complex128) string {
	re := formatFloat(real(v))
	im := formatFloat(imag(v))
	if im[0] != '-' && im[0] != '+' {
		im = "+" + im
	}
	return "(" + re + im + "i)"
}
