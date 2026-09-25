# Function attributes

LLGo accepts `cold` and `noreturn` on named function and method declarations,
including external declarations. Package `init` functions and blank-named
functions are not supported. Write each attribute on its own comment line:

```go
//llgo:cold
//llgo:noreturn
func Fatal(message string) {
    panic(message)
}
```

The spaced spelling `// llgo:cold` is also accepted. These attributes take no
arguments. Identical repetitions are harmless, and their order does not matter.

- `cold` is an optimization hint that calls are uncommon.
- `noreturn` promises that the function never returns normally. It may panic or
  run indefinitely. Panic unwinding and deferred functions retain their normal
  behavior. A function that recovers its own panic and returns does not satisfy
  this promise.

Attributes are programmer promises, not runtime checks. An incorrect `noreturn`
annotation can cause incorrect optimization. The compiler checks annotation
placement and syntax; it does not prove the behavior of arbitrary function bodies.

Imported declarations and generic instances retain the attributes of their source
declarations. Package patches select the replacement declaration's annotations.
An omitted annotation supplies no guarantee. Source declarations remain separate
even when `go:linkname` makes them refer to one backend symbol; each promise must
be valid for the implementation it names. An indirect call with an unknown target
does not acquire guarantees merely from its Go function type.

## Package exports

Validated attributes are stored in the existing package cache manifest:

```yaml
exports:
  version: 1
  functions:
    - name: Fatal
      cold: true
      noreturn: true
```

Names identify source declarations within the package (`F`, `T.Method`, or
`(*T).Method`), not linker symbols. Records are sorted by name. A supported
version with an empty function list means the package has no exported attributes.
Missing or incompatible export records invalidate the cache entry.

The build driver restores records on cache hits and collects them from effective
source, including patches, on cache misses. It associates these records with Go
package identities before backend workers start. Main packages, declaration-only
packages, and builds without caching use the same records in memory. Cache
manifests are published after the archive is ready. Source changes, overlays,
patches, and the export format version participate in build fingerprints;
dependencies retain their build fingerprint even when they have a module version.

For embedders using `cl.Options`, call `CollectPackageExports` for each package
and bind the result to its Go package identity with `PackageExports.Set`. Supply
the shared index through `Options.Exports` before concurrent lowering starts.
One-shot compilation without an index prepares the current package automatically.
Imported declarations use validated records rather than rereading source comments.
