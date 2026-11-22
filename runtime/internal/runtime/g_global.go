//go:build !llgo || baremetal

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

// Host builds and single-context targets share one process-wide current G.
var currentG *g

func getg() *g {
	if EmbeddedStartupProof {
		RequireEmbeddedStartup()
	}
	if currentG == nil {
		currentG = initRuntimeContext(new(runtimeContext), nil, _Grunning)
	}
	return currentG
}

func setg(gp *g) {
	currentG = gp
}

func currentGUsesLifecycle() bool {
	return false
}
