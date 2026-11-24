// Copyright 2014 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license.
// See LICENSES/Go-BSD-3-Clause.txt at this module root for license terms.

package runtime

var divideError error
var overflowError error

var shiftError = error(errorString("negative shift amount"))
