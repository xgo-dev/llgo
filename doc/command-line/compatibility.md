# Go command-line compatibility

LLGo compiles Go source through its own frontend, LLVM backend and runtime.
Accepting a flag in the Go package loader does not implement that flag in LLGo.
The reviewed public command and flag inventory is pinned to `.go-version` in
[`cmd/internal/compat/go1.27.json`](../../cmd/internal/compat/go1.27.json).
[`manifest.go`](../../cmd/internal/compat/manifest.go) records each command and
flag's status, owner and limits, separately from LLGo extensions. The inventory
includes `tool compile` and module/workspace subcommands. CI compares it with
public help from the pinned Go executable. Compiler usage is read directly:
its `-h` means halt-on-error, so it must not terminate artifact/profile probes.
After a toolchain
upgrade, run `python3 dev/check_cli_surface.py --update` and review the resulting
surface and contract changes together.

`fmt`, `doc`, `mod` and `work` delegate to Go unchanged. `fix`, `generate`, `get`,
`list` and `vet` delegate with LLGo source tags and target selection. Their
compiler controls describe the delegated Go operation, not an LLGo executable.
`env` combines Go environment queries with explicit LLGo queries. Go telemetry
is intentionally omitted. Native build, run, test, install, cleanup, version and
tool workflows implement a documented subset; unsupported controls fail.
Command status identifies the implemented LLGo workflow. Individual flag
records define supported effects and limits; native status does not imply
support for every Go flag or gc artifact format.

Native package commands accept a leading `-C dir` or `-C=dir` after the command
name. It applies before relative source, output, module and overlay paths.
`GOFLAGS`, including persisted `go env -w` defaults, are parsed before command
flags, so explicit options take precedence. Defaults belonging only to other
commands are skipped, as in Go. `-C` is not permitted in `GOFLAGS`.
Unimplemented instrumentation cannot bypass validation through environment
settings.

`-mod` and `-modfile` select the source module graph. `-modcacherw` leaves newly
downloaded module directories writable. `-overlay` accepts Go's `Replace` JSON
object for Go source replacement, addition and deletion. Relative logical and
backing paths are resolved after `-C`; runtime loads and the LLGo parser share
that source view. Non-Go replacements (including embedded data and C headers)
are rejected until all native consumers support them. Runtime file operations
see the ordinary filesystem, as with Go's build overlay.
Empty replacement maps are valid. Logical paths that resolve to the same
absolute path are rejected, as in Go.

The portable `-gcflags` subset is `-lang`, `-N`, `-l` (0/1 or boolean), and global
`-d=converthash` settings. Unpatterned and `all=` lists configure the global LLGo
frontend; gc diagnostics, selective hash patterns and other package patterns
are rejected. The `-ldflags` subset is `-s`, `-w`, `-extld` and `-extldflags` with
unpatterned or `all=` lists. These configure LLGo's native linker. Other linker
options are rejected rather than discarded.

`build`, `run` and `install -v` print successfully compiled package paths.
Use LLGo's separate `-compiler-verbose`/`-cv` extension for compiler and device
diagnostics. Versioned `install package@version` operands are explicitly
rejected until isolated module selection is implemented. Ordinary installs
use the invocation's module graph and configured GOBIN.

Compiler options accept Go's single- and double-dash spellings and
space-separated or `=value` arguments where applicable. For example,
`-gcflags='-lang go1.21'` and `-gcflags=-lang=go1.21` both select Go 1.21 language
semantics, including through quoted `GOFLAGS` values. As in `cmd/compile`, each
bare `-N` or `-l` (or `=true`) increments its count, `=false` resets the count,
and an integer replaces it. Only the literal `true` and `false` boolean values
are accepted. The effective `-l` count must remain 0 or 1; gc inliner diagnostic
levels are unsupported.

Repeated `-gcflags` lists use Go's last-matching-list rule within these supported
scopes. For example, command-line `-gcflags=-N` replaces `GOFLAGS=-gcflags=-m`;
the superseded compiler option is not validated. A later `all=` list replaces
both scopes, while an unpatterned list does not replace an earlier `all=` list
for dependencies. Any unsupported compiler options that remain applicable are
rejected. Malformed argument lists and unsupported package patterns are always
rejected, including in superseded lists. For `-ldflags`, malformed syntax and
invalid values of recognized options are still rejected in superseded lists.

`-race`, `-msan`, `-asan`, `-pgo`, `-linkshared`, compiler selection and gc object
archive controls have no implemented LLGo backend contract. `-n`, `-work`,
`-trimpath`, `-buildvcs` and build JSON are also rejected pending native artifact
support. `-buildmode` supports `exe`, `c-archive` and `c-shared`; gc shared-package,
plugin, PIE and other modes are rejected. `-buildenv` has been removed because it
had no consumer. `clean` exposes
only its implemented package tags, verbosity and directory selection; cleanup
planning and cache controls are pending. Size reports belong to `build`, and
LTO pass plugins belong to build-producing commands rather than `monitor`.

`run` and `cmptest` select consecutive `.go` file operands before program
arguments. The Go reference used by `cmptest` receives exactly those files.
`_test.go` files and versioned `package@version` run operands are rejected.
`run -exec` and `test -exec` accept a quoted execution wrapper. The wrapper
receives its own arguments, the linked artifact and unchanged program/test
arguments, without shell expansion. It overrides the default native or target
runner. Test wrappers execute in each package's source directory.

`test` accepts public test-binary flags with or without a `test.` prefix before
`-args`, including after package operands. Both spellings share the same value;
later settings win. Driver-only flags such as `-json`, `-cover` and `-exec` do
not have `test.` aliases. Arguments after `-args` are passed to the test binary.
Execution trace and block/mutex profile controls are rejected, including their
reserved `-test.*` spellings after `-args`, because LLGo does not produce those
runtime samples. `-cpu` is also rejected until native `GOMAXPROCS` enforces the
requested execution limit. Merely repeating a test with CPU suffixes does not
provide that behavior. Fuzz engine and worker/cache controls are rejected
until instrumentation, minimization, crash and cache behavior are qualified.
Fuzz seed cases still run as ordinary tests without `-fuzz`.

Profiling is qualified for native Linux, macOS and Windows runtimes without
named targets or freestanding tags. CPU sampling requires amd64/arm64 on
Linux/macOS or 386/amd64/arm64 on Windows. Unsupported targets fail before
package loading instead of emitting placeholder profiles. Profile flags
require one package, retain its executable in the invocation directory (or
the explicit `-o` path), and resolve output directories from the invocation
directory.

Allocation profiles contain cumulative size-class counts, rather than gc's
sampled allocation stacks or precise live-heap accounting. LLGo counts every
allocation and defaults `runtime.MemProfileRate` to 1 so pprof does not rescale
these counts as probabilistic samples. `-memprofilerate=1` is supported; other
rates fail explicitly. CPU samples come from LLGo's native sampler. Executable
acceptance tests validate both file formats with Go's pprof reader.

`-artifacts` retains `testing.ArtifactDir` output. Its default output directory
is the invocation directory; relative `-outputdir` paths are resolved there
before changing into package directories. Test execution requires
`-buildmode=exe`; `c-archive` and `c-shared` test artifacts require `-c` and
remain LLGo C-ABI artifacts.

`tool compile` parses source and generates LLGo IR in memory. It rejects gc
object output, import configurations, search paths and package import identity
controls. `version` prints LLGo identity when no file operand, `-m` or `-json`
is present, including with `-v` alone. File inspection, `-m` and `-json` delegate
to Go, preserving diagnostics and exit status. This reader requires Go build
information; it cannot inspect LLGo artifacts that lack that metadata.
`bug` opens an LLGo issue draft containing
only LLGo/LLVM/Go build versions and the host platform; `bug -n` prints the
payload. Paths, arbitrary environment settings and dependency lists are not
collected.

The pinned public surface has a final contract when each command has a native,
delegated, omitted or rejected decision and each accepted flag has a tested
consumer. Explicit rejection is the contract for unsupported features, whose
later implementation requires a manifest update and new acceptance tests.
The contract is connected to its verification as follows:

| Contract | Evidence |
| --- | --- |
| Pinned command/flag inventory, including compiler flags | `dev/check_cli_surface.py`, `TestManifestNamesOwnersAndRegistration`, `TestCompilerManifestAndRegistration` |
| Early rejection, including aliases, GOFLAGS and reserved `-args` controls | `TestNativeUnsupportedControlsFailBeforeLoading`, `TestNativeUnqualifiedTestControlsFailEarly`, compiler rejection tests |
| Source/module/overlay selection and frontend/linker subsets | Native CLI source-selection tests, `internal/goflags` validation tests and build integration tests |
| Run/install consumers and package verbosity | `TestNativeCompilerLanguageFlagSpellings`, `TestNativeDirectoryAndOverlaySourceSelection`, `TestNativeRunAndInstallVerbosity` |
| Compile-only C ABI test libraries | `TestNativeTestLibraryArtifactsRequireCompileOnly` |
| Multi-file run and exact-file cmptest reference | `TestRunMultipleFilesPreservesProgramArgs`, `TestCmpTestUsesSelectedFiles` |
| Wrapper argv, working directory, exit, timeout, coverage and JSON | `TestExecOverridesTargetRunner`, `TestExecExitAndTimeout`, wrapped-run/artifact tests and coverage/JSON runner tests |
| Selection, repetition, parallelism, shuffle, listing, benchmarks, fail-fast, full paths and seeds | `TestNativeTestSelectionAndBenchmarkConsumers` |
| Profile lifecycle, retained binaries, valid pprof and multi-package rejection | `TestNativeProfilesRetainBinaryAndUseInvocationDirectory`, target qualification and native CPU/memory runtime tests |
| Artifact lifecycle and test-binary aliases | `TestArtifactsUseInvocationOutputDirectory`, alias precedence and bidirectional registration tests |
| Delegated stdout, stderr, help/error diagnostics and status | `TestDelegatedCommandStreamsAndExitStatus`, source-aware delegation and file-inspection parity tests |

LLGo-specific optimization, LTO, target, runner, firmware, debugger, size and
runtime-stack flags remain separate from the Go inventory. Their owners are
listed in `compat.Extensions`; they do not imply gc compiler, scheduler,
artifact, telemetry or profiling compatibility.
