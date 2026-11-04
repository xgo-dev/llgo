package plan9asm

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"go/types"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/xgo-dev/llgo/internal/packages"
	extplan9asm "github.com/xgo-dev/plan9asm"
)

// Physical SP slots require an actual Go object owner. Logical negative AUTO
// slots retain their existing declaration-only path. Native components never
// enter this typed translation path.
func bindAMD64Frame(pkg *packages.Package, source []byte, filename, goos, goarch string,
	opt TranslateOptions, goPkg extplan9asm.GoPackage, target string, resolve func(string) string,
) (map[string]extplan9asm.FuncSig, func() error, error) {
	if goarch != "amd64" {
		return nil, nil, nil
	}
	file, err := extplan9asm.Parse(extplan9asm.ArchAMD64, string(source))
	if err != nil {
		return nil, nil, err
	}
	physical := false
	for _, function := range file.Funcs {
		for _, instruction := range function.Instrs {
			for _, operand := range instruction.Args {
				physical = physical || function.FrameSize > 0 && operand.Kind == extplan9asm.OpMem &&
					operand.Mem.Base == extplan9asm.SP && operand.Mem.Off >= 0
			}
		}
	}
	if !physical {
		return nil, nil, nil
	}
	assembly, directory, err := assemblyPackageFile(pkg, filename)
	if err != nil {
		return nil, nil, err
	}
	for path := range opt.Overlay {
		absolute, err := filepath.Abs(path)
		if err != nil {
			return nil, nil, err
		}
		if relative, err := filepath.Rel(directory, absolute); err == nil && filepath.IsLocal(relative) {
			return nil, nil, fmt.Errorf("%w: physical AMD64 frame needs an original source mapping for package overlays", extplan9asm.ErrProbeNeedsContext)
		}
	}
	root := opt.GOROOT
	if root == "" {
		root = runtime.GOROOT()
	}
	root, err = filepath.EvalSymlinks(root)
	if err != nil {
		return nil, nil, err
	}
	goBinary := filepath.Join(root, "bin", "go")
	if runtime.GOOS == "windows" {
		goBinary += ".exe"
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	env := opt.Environment
	if env == nil {
		env = os.Environ()
	}
	settings := map[string]string{
		"GOROOT": root, "GOOS": goos, "GOARCH": goarch,
		"GOTOOLCHAIN": "local", "GOENV": "off", "GOWORK": "off", "GOFLAGS": "",
	}
	if opt.GOAMD64 != "" {
		settings["GOAMD64"] = opt.GOAMD64
	}
	env = frameEnvironment(env, settings)
	paths := make(map[string]string)
	for _, value := range env {
		key, path, _ := strings.Cut(value, "=")
		if key == "GOPATH" || key == "GOMODCACHE" || key == "GOCACHE" {
			paths[key] = path
		}
	}
	if !filepath.IsAbs(paths["GOPATH"]) || !filepath.IsAbs(paths["GOMODCACHE"]) || !filepath.IsAbs(paths["GOCACHE"]) {
		// Standalone adapter callers have no invocation-level SDK snapshot.
		// The build driver supplies these from its existing single env query.
		command := exec.CommandContext(ctx, goBinary, "env", "-json", "GOPATH", "GOMODCACHE", "GOCACHE")
		command.Dir, command.Env = directory, env
		data, err := command.Output()
		if err != nil {
			cancel()
			return nil, nil, fmt.Errorf("physical AMD64 frame source environment: %w", err)
		}
		if err := json.Unmarshal(data, &paths); err != nil {
			cancel()
			return nil, nil, err
		}
	}
	temporary, err := os.MkdirTemp("", "llgo-amd64-frame-")
	if err != nil {
		cancel()
		return nil, nil, err
	}
	paths["TMPDIR"], paths["GOTMPDIR"] = temporary, temporary
	env = frameEnvironment(env, paths)
	request := extplan9asm.GoAMD64FrameCaptureRequest{
		GoBinary: goBinary, Directory: directory, AssemblyFile: filepath.Base(assembly),
		Target: goos + "/" + goarch, Environment: env,
	}
	var owner *extplan9asm.GoAMD64FrameSource
	relative, relativeErr := filepath.Rel(filepath.Join(root, "src"), directory)
	if relativeErr == nil && filepath.IsLocal(relative) {
		owner, err = extplan9asm.CaptureGoAMD64SDKFrameSource(ctx, goPkg, request)
	} else if pkg.Module != nil && pkg.Module.Version != "" && pkg.Module.Replace == nil {
		owner, err = extplan9asm.CaptureGoAMD64DownloadedFrameSource(ctx, goPkg, extplan9asm.GoAMD64DownloadedFrameCaptureRequest{
			Original: request, SourceModule: pkg.Module.Path, SourceVersion: pkg.Module.Version,
		})
	} else {
		owner, err = extplan9asm.CaptureGoAMD64FrameSource(ctx, goPkg, request)
	}
	finish := func() error {
		var result error
		if owner != nil {
			result = errors.Join(owner.Verify(ctx), owner.Close())
		}
		cancel()
		return errors.Join(result, os.RemoveAll(temporary))
	}
	if err != nil {
		return nil, nil, errors.Join(err, finish())
	}
	sigs := make(map[string]extplan9asm.FuncSig)
	for _, function := range file.Funcs {
		name := resolve(function.Sym)
		local := strings.TrimPrefix(name, goPkg.Path+".")
		if declaration, ok := pkg.Types.Scope().Lookup(local).(*types.Func); ok {
			sigs[name], err = extplan9asm.DeriveGoFuncABI0Signature(name, declaration, goarch, types.SizesFor("gc", goarch))
			if err != nil {
				return nil, nil, errors.Join(err, finish())
			}
		}
	}
	bound, err := owner.BindFixedLocal(ctx, goPkg, file, extplan9asm.Options{
		Goarch: goarch, TargetTriple: target, ResolveSym: resolve, Sigs: sigs,
	})
	if err != nil {
		return nil, nil, errors.Join(err, finish())
	}
	return bound.Sigs, finish, nil
}

func frameEnvironment(original []string, settings map[string]string) []string {
	result := make([]string, 0, len(original)+len(settings))
	for _, value := range original {
		key, _, _ := strings.Cut(value, "=")
		if _, replaced := settings[key]; !replaced {
			result = append(result, value)
		}
	}
	for key, value := range settings {
		result = append(result, key+"="+value)
	}
	return result
}
