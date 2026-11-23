//go:build llgo && llgo.size.binary && go1.23

package binary

//llgointernal:template:bind Encode binary op=encode buffer=buf order=order value=data
//llgointernal:template:bind Decode binary op=decode buffer=buf order=order value=data
//llgointernal:template:bind Append binary op=fallback

// LLGoBinaryShortBuffer keeps the error object used by Encode and Decode.
func LLGoBinaryShortBuffer() error { return errBufferTooSmall }
