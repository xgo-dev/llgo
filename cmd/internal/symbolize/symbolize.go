package symbolize

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"regexp"
	"strconv"

	"github.com/xgo-dev/llgo/cmd/compiler/pclnmap"
	"github.com/xgo-dev/llgo/cmd/compiler/wasmdebug"
)

// Run resolves raw logical frames using a module and its adjacent PCLN map.
func Run(modulePath string, trace io.Reader, out io.Writer) error {
	module, err := os.ReadFile(modulePath)
	if err != nil {
		return err
	}
	raw, err := os.ReadFile(modulePath + ".pclntab")
	if err != nil {
		return err
	}
	locations, err := pclnmap.ParseWasmLocations(raw)
	if err != nil {
		return err
	}
	if err := wasmdebug.MatchPCLNIdentity(module, locations.Identity()); err != nil {
		return err
	}
	return resolveTrace(trace, out, locations)
}

var rawFrame = regexp.MustCompile(`\?\?\? \[pc=[^\]]+\] \[llgo-location=([0-9]+)\]\(\.\.\.\)$`)

var rawPosition = regexp.MustCompile(`\t\?\?\?:[0-9]+$`)

func resolveTrace(trace io.Reader, out io.Writer, locations pclnmap.WasmLocations) error {
	scan := bufio.NewScanner(trace)
	scan.Buffer(make([]byte, 4096), 4<<20)
	replacePosition := false
	for scan.Scan() {
		line := scan.Text()
		skip := replacePosition && rawPosition.MatchString(line)
		replacePosition = false
		if skip {
			continue
		}
		if match := rawFrame.FindStringSubmatchIndex(line); match != nil {
			id, err := strconv.ParseUint(line[match[2]:match[3]], 10, 64)
			if err == nil {
				if loc, ok := locations.Resolve(id); ok {
					prefix := line[:match[0]]
					line = fmt.Sprintf("%s%s(...)\n%s\t%s:%d", prefix, loc.Function, prefix, loc.File, loc.Line)
					replacePosition = true
				}
			}
		}
		if _, err := fmt.Fprintln(out, line); err != nil {
			return err
		}
	}
	return scan.Err()
}
