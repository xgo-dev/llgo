// Package funcattrs lowers prepared directive contracts into LLVM.
package funcattrs

import "github.com/xgo-dev/llgo/internal/directive"

type Attribute = directive.Attribute
type Target = directive.Target
type RangeBounds = directive.RangeBounds

const (
	Parameter = directive.Parameter
	Result    = directive.Result
	Receiver  = directive.Receiver
)

var (
	Validate      = directive.Validate
	ResolveTarget = directive.ResolveTarget
	IntegerRange  = directive.IntegerRange
)
