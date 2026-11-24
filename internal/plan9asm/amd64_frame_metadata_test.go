package plan9asm

import (
	"encoding/json"
	"go/types"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	extplan9asm "github.com/xgo-dev/plan9asm"
	"golang.org/x/tools/go/packages"
)

// The existing test executable is a portable fake SDK tool. Only the child
// receives this variable; no process-wide command hook or mutable client exists.
func init() {
	mode := os.Getenv("LLGO_FRAME_METADATA_TEST_CHILD")
	if mode == "" {
		return
	}
	directory, _ := os.Getwd()
	metadata, _ := os.ReadFile(filepath.Join(directory, "go.mod"))
	record := struct {
		Arguments   []string
		Directory   string
		GoMod       string
		Environment []string
	}{os.Args[1:], directory, string(metadata), os.Environ()}
	body, _ := json.Marshal(record)
	stream, _ := os.OpenFile(os.Getenv("LLGO_FRAME_METADATA_TEST_TRACE"), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if stream != nil {
		_, _ = stream.Write(append(body, '\n'))
		_ = stream.Close()
	}
	if len(os.Args) != 5 || strings.Join(os.Args[1:4], " ") != "mod download -json" {
		os.Exit(71) // The separate real-source/SDK owner is outside this fake.
	}
	switch mode {
	case "deadline":
		time.Sleep(10 * time.Second)
	case "command-error":
		_, _ = os.Stdout.WriteString(`{"Path":"example.com/sdkframe","Version":"v1.0.0","Error":"checksum verifier metadata missing"}`)
		_, _ = os.Stderr.WriteString("normal Go diagnostic retained")
		os.Exit(23)
	case "json-error":
		_, _ = os.Stdout.WriteString(`{"Path":"example.com/sdkframe","Version":"v1.0.0","Error":"normal download JSON failure"}`)
	case "invalid-json":
		_, _ = os.Stdout.WriteString(`{`)
	case "bad-checksum":
		_, _ = os.Stdout.WriteString(`{"Path":"example.com/sdkframe","Version":"v1.0.0","Sum":"h1:invalid","GoModSum":"h1:AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA="}`)
	case "wrong-version":
		_, _ = os.Stdout.WriteString(`{"Path":"example.com/sdkframe","Version":"v2.0.0","Sum":"h1:AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=","GoModSum":"h1:AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA="}`)
	case "wrong-identity":
		_, _ = os.Stdout.WriteString(`{"Path":"example.com/other","Version":"v1.0.0","Sum":"h1:AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=","GoModSum":"h1:AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA="}`)
	case "trailing-json":
		_, _ = os.Stdout.WriteString(`{"Path":"example.com/sdkframe","Version":"v1.0.0","Sum":"h1:AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=","GoModSum":"h1:AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA="}{}`)
	default:
		_, _ = os.Stdout.WriteString(`{"Path":"example.com/sdkframe","Version":"v1.0.0","Sum":"h1:AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=","GoModSum":"h1:AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA="}`)
	}
	os.Exit(0)
}

func TestAMD64DownloadedFrameMetadataPreparation(t *testing.T) {
	for _, mode := range []string{"success", "command-error", "json-error", "invalid-json", "bad-checksum", "wrong-version", "wrong-identity", "trailing-json"} {
		t.Run(mode, func(t *testing.T) {
			pkg, directory := amd64SDKFrameFixture(t)
			pkg.Module = &packages.Module{Path: "example.com/sdkframe", Version: "v1.0.0", Dir: directory}
			root := t.TempDir()
			if err := os.MkdirAll(filepath.Join(root, "bin"), 0o700); err != nil {
				t.Fatal(err)
			}
			executable, err := os.Executable()
			if err != nil {
				t.Fatal(err)
			}
			name := "go"
			if strings.HasSuffix(executable, ".exe") {
				name += ".exe"
			}
			tool := filepath.Join(root, "bin", name)
			body, err := os.ReadFile(executable)
			if err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(tool, body, 0o700); err != nil {
				t.Fatal(err)
			}
			trace := filepath.Join(t.TempDir(), "commands.jsonl")
			environment := frameEnvironment(os.Environ(), map[string]string{
				"LLGO_FRAME_METADATA_TEST_CHILD": mode,
				"LLGO_FRAME_METADATA_TEST_TRACE": trace,
				"GOPATH":                         t.TempDir(), "GOMODCACHE": t.TempDir(), "GOCACHE": t.TempDir(),
				"GOPROXY": "https://normal.example.invalid", "GOSUMDB": "sum.golang.org",
			})
			_, _, callErr := bindAMD64Frame(pkg, []byte(amd64SDKFrameAssembly), "copy_amd64.s", "linux", "amd64",
				TranslateOptions{GOROOT: root, GOAMD64: "v2", Environment: environment}, extplan9asm.GoPackage{Path: pkg.PkgPath, Types: pkg.Types, Syntax: pkg.Syntax, Imports: map[string]*types.Package{"runtime": pkg.Imports["runtime"].Types}}, "x86_64-unknown-linux-gnu", ResolveSymFunc(pkg.PkgPath))
			data, err := os.ReadFile(trace)
			if err != nil {
				t.Fatal("downloaded physical frame never invoked its normal metadata tool:", err, callErr)
			}
			var commands []struct {
				Arguments   []string
				Directory   string
				GoMod       string
				Environment []string
			}
			for _, line := range strings.Split(strings.TrimSpace(string(data)), "\n") {
				var command struct {
					Arguments   []string
					Directory   string
					GoMod       string
					Environment []string
				}
				if err := json.Unmarshal([]byte(line), &command); err != nil {
					t.Fatal(err)
				}
				commands = append(commands, command)
			}
			first := commands[0]
			metadataCommands := 0
			for _, command := range commands {
				if strings.Join(command.Arguments, " ") == "mod download -json example.com/sdkframe@v1.0.0" {
					metadataCommands++
				}
			}
			if metadataCommands != 1 {
				t.Fatalf("expected one normal command for the selected source file, got %d (%v)", metadataCommands, callErr)
			}
			if mode == "success" && (callErr == nil || strings.Contains(callErr.Error(), "downloaded AMD64 frame metadata")) {
				t.Fatal("normal metadata preparation must reach the separately rejecting fake SDK owner", callErr)
			}
			if strings.Join(first.Arguments, " ") != "mod download -json example.com/sdkframe@v1.0.0" {
				t.Fatalf("first needed-file command must prepare normal exact-module metadata, got %q (%v)", first.Arguments, callErr)
			}
			if first.Directory == directory || !strings.Contains(first.GoMod, "module example.invalid/") {
				t.Fatal("metadata must use its owned empty module, never mutate the consumer", first)
			}
			for _, expected := range []string{"GOPROXY=https://normal.example.invalid", "GOSUMDB=sum.golang.org", "GOAMD64=v2", "GOTOOLCHAIN=local", "GOWORK=off", "GOFLAGS="} {
				if !strings.Contains("\n"+strings.Join(first.Environment, "\n")+"\n", "\n"+expected+"\n") {
					t.Errorf("resolved source SDK/environment missing %q", expected)
				}
			}
			if _, err := os.Stat(first.Directory); !os.IsNotExist(err) {
				t.Fatal("owned metadata boundary was not removed", first.Directory, err)
			}
			if mode != "success" && (callErr == nil || !strings.Contains(callErr.Error(), "metadata")) {
				t.Fatal("normal metadata failure was not identified", callErr)
			}
			if mode == "command-error" && (!strings.Contains(callErr.Error(), "checksum verifier metadata missing") || !strings.Contains(callErr.Error(), "normal Go diagnostic retained")) {
				t.Fatal("actual JSON stdout and stderr were discarded", callErr)
			}
		})
	}
}
