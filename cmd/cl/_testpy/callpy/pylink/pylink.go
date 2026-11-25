package pylink

// This tiny package supplies only the Python embed link and module-symbol
// loader required by the local fixtures.
const (
	LLGoFiles   = "_pylib/module.c"
	LLGoPackage = "link: $LLGO_LIB_PYTHON; $(pkg-config --libs python3-embed)"
)
