package build

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"go/types"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/xgo-dev/llgo/cl"
	"github.com/xgo-dev/llgo/internal/goescape"
	"golang.org/x/tools/go/ssa"
)

const llgoEscapeCache = "LLGO_ESCAPE_CACHE"

func (ctx *context) prepareEscapeAnalysis(event func(int, int)) *goescape.Result {
	var cache *goescape.Cache
	var files *escapeCacheFiles
	hits, misses := 0, 0
	// Validation can cost more than solving small functions. Keep persistence
	// opt-in until representative builds establish a worthwhile saving.
	if cacheEnabled() && isEnvOn(llgoEscapeCache, false) {
		// Share the archive cache's target, compiler, source toolchain and
		// feature identities. The analyzer additionally hashes exact SSA,
		// named type bodies, replacement effects and backend layout policies.
		manifest := newManifestBuilder()
		ctx.collectEnvInputs(manifest)
		ctx.collectCommonInputs(manifest)
		namespace := fmt.Sprintf("%s|roots=%v|word=%d|layout=%s", manifest.Fingerprint(), ctx.prog.GCRootsEnabled(), ctx.prog.GoWordSize(), ctx.prog.DataLayout())
		dir := filepath.Join(ctx.ensureCacheManager().root, "escape")
		files = &escapeCacheFiles{dir: dir, shards: make(map[string]*escapeCacheShard)}
		cache = &goescape.Cache{
			Namespace: namespace,
			BodyKey:   ctx.escapeBodyKeys(),
			Load:      files.load,
			Store:     files.store,
			Event: func(hit bool) {
				if hit {
					hits++
				} else {
					misses++
				}
			},
		}
	}
	var imports map[*ssa.Function]goescape.Summary
	var keys map[*ssa.Function]string
	var roots []*ssa.Function
	if ctx.frontend != nil {
		imports, keys, roots = ctx.frontend.imports, ctx.frontend.importKeys, ctx.frontend.methods
	}
	result := cl.PrepareEscapeAnalysisImported(ctx.prog, ctx.progSSA, ctx.patches, cache, imports, keys, roots)
	if files != nil {
		files.flush()
	}
	if event != nil {
		event(hits, misses)
	}
	if cache != nil && ctx.buildConf.Verbose {
		fmt.Fprintf(os.Stderr, "ESCAPE CACHE hits=%d misses=%d\n", hits, misses)
	}
	return result
}

// Reuse the archive input fingerprints, including overlays, alternate runtime
// sources and transforms. They are memoized for the later archive lookup, so
// this moves existing work rather than digesting files a second time. Bodies
// without an unambiguous package identity retain the full SSA fingerprint.
func (ctx *context) escapeBodyKeys() func(*ssa.Function) string {
	keys := make(map[*types.Package]string)
	add := func(pkg *types.Package, key string) {
		if previous, found := keys[pkg]; found && previous != key {
			key = "" // Conflicting package identities require exact SSA.
		}
		keys[pkg] = key
	}
	ids := make([]string, 0, len(ctx.pkgByID))
	for id := range ctx.pkgByID {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		pkg := ctx.pkgByID[id]
		if ctx.collectFingerprint(pkg) != nil || ctx.packageCacheDisabled(id) {
			continue
		}
		add(pkg.Types, pkg.Fingerprint)
		if pkg.AltPkg != nil {
			add(pkg.AltPkg.Types, pkg.Fingerprint)
		}
	}
	return func(fn *ssa.Function) string {
		// Instantiations can fold constants using type arguments defined by
		// callers outside the generic package's dependency graph.
		if fn.Origin() != nil {
			return ""
		}
		for fn != nil {
			if fn.Pkg != nil {
				return keys[fn.Pkg.Pkg]
			}
			fn = fn.Parent()
		}
		return ""
	}
}

type escapeCacheShard struct {
	values map[string]json.RawMessage
	used   map[string]bool
	dirty  bool
}

// Batch by package: one filesystem read/write per used package, rather than
// thousands of tiny per-function files. Atomic replacement can lose a racing
// writer's cache entries, which only causes later misses, never partial facts.
type escapeCacheFiles struct {
	dir    string
	shards map[string]*escapeCacheShard
}

func (files *escapeCacheFiles) shard(key string) (*escapeCacheShard, string, error) {
	bucket, item, ok := strings.Cut(key, ".")
	if !ok || len(bucket) != 64 || len(item) != 64 {
		return nil, "", fmt.Errorf("invalid escape cache key")
	}
	if _, err := hex.DecodeString(bucket + item); err != nil {
		return nil, "", err
	}
	shard := files.shards[bucket]
	if shard == nil {
		shard = &escapeCacheShard{values: make(map[string]json.RawMessage), used: make(map[string]bool)}
		path := filepath.Join(files.dir, bucket+".json")
		if info, err := os.Stat(path); err == nil && info.Size() <= goescape.MaxCacheBytes {
			if data, err := os.ReadFile(path); err == nil {
				if json.Unmarshal(data, &shard.values) != nil || shard.values == nil {
					shard.values = make(map[string]json.RawMessage)
				}
			}
		}
		files.shards[bucket] = shard
	}
	shard.used[item] = true
	return shard, item, nil
}

func (files *escapeCacheFiles) load(key string) ([]byte, error) {
	shard, item, err := files.shard(key)
	if err != nil {
		return nil, err
	}
	if data, ok := shard.values[item]; ok {
		return data, nil
	}
	return nil, os.ErrNotExist
}

func (files *escapeCacheFiles) store(key string, data []byte) error {
	shard, item, err := files.shard(key)
	if err != nil {
		return err
	}
	shard.values[item] = append(json.RawMessage(nil), data...)
	shard.dirty = true
	return nil
}

func (files *escapeCacheFiles) flush() {
	for bucket, shard := range files.shards {
		if !shard.dirty {
			continue
		}
		data, err := json.Marshal(shard.values)
		if err != nil {
			continue
		}
		if len(data) > goescape.MaxCacheBytes {
			for key := range shard.values {
				if !shard.used[key] {
					delete(shard.values, key)
				}
			}
			data, err = json.Marshal(shard.values)
			if err != nil {
				continue
			}
		}
		if len(data) > goescape.MaxCacheBytes {
			// Even the entries used by this build exceed the shard limit.
			// Leave the previous file intact; new facts will miss next time.
			continue
		}
		writeManifest(filepath.Join(files.dir, bucket+".json"), string(data))
	}
}
