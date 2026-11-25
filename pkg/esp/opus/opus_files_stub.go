//go:build !(llgo && !llgo.idf)

// Host builds and ESP-IDF builds compile no vendored C: on a host there is
// no codec at all (the stubs answer), and on an ESP-IDF link the codec comes
// from the project's own component registration with the project-side glue
// (glue/opus_glue.c), exactly as before this package learned the baremetal
// form.
package opus

const csrcFiles = ""
