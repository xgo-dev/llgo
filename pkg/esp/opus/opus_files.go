package opus

// LLGoFiles feeds the baremetal build's vendored Opus sources to the llgo
// compiler; the value lives in the build-tagged pair (opus_files_csrc.go /
// opus_files_stub.go) so only the no-IDF llgo build carries them.
const LLGoFiles = csrcFiles
