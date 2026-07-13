//go:build baremetal

/*
 * Copyright (c) 2026 The XGo Authors (xgo.dev). All rights reserved.
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

import "unsafe"

var baremetalG g

func getg() *g {
	return &baremetalG
}

// Bare-metal panic recovery is not implemented yet. Keep the existing
// pthread-key behavior, where storing and loading the panic value are no-ops.
func getPanic(*g) unsafe.Pointer {
	return nil
}

func setPanic(*g, unsafe.Pointer) {
}
