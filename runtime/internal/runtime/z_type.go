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

import (
	"github.com/xgo-dev/llgo/runtime/abi"
)

type Kind = abi.Kind
type Type = abi.Type

// -----------------------------------------------------------------------------

func isDirectIface(t *_type) bool {
	return t.Kind_&abi.KindDirectIface != 0
}

// runtime type list initialized by compiler
var typelist []*abi.Type

// -----------------------------------------------------------------------------
