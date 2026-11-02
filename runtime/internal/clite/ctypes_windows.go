//go:build windows
// +build windows

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

package c

// For Windows (LLP64 model), Long is 32-bit, regardless of architecture
type (
	Long  = int32
	Ulong = uint32
	// Windows uses LLP64: ssize_t/SSIZE_T follows the pointer, not C long.
	SsizeT = int
)
