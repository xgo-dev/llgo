package build

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/xgo-dev/llgo/internal/pclnpost"
)

// LLVM must print the reservation's type and initializer as [N x i8] on its
// global declaration line. A printer format change must fail closed below.
var symbolEntryArrayType = regexp.MustCompile(`\[([0-9]+) x i8\]`)

// The linked text range bounds the bucket count even when foreign code lies
// between Go functions. Grow only the data reservation; no function references
// or instructions are added by the relink.
func growSymbolEntryReserve(ir string, analysis pclnpost.ExternalAnalysis) (string, bool, error) {
	lines := strings.Split(ir, "\n")
	for i, line := range lines {
		if !strings.HasPrefix(line, "@__llgo_funcinfo_entry_reserve = ") {
			continue
		}
		match := symbolEntryArrayType.FindStringSubmatch(line)
		if len(match) != 2 {
			return "", false, fmt.Errorf("invalid symbol entry reservation")
		}
		capacity, err := strconv.ParseUint(match[1], 10, 64)
		if err != nil {
			return "", false, fmt.Errorf("invalid symbol entry capacity: %w", err)
		}
		need, err := pclnpost.SymbolEntryReserveSize(analysis)
		if err != nil {
			return "", false, err
		}
		if capacity >= need {
			return ir, false, nil
		}
		capacity = (need + 4095) &^ uint64(4095)
		lines[i] = symbolEntryArrayType.ReplaceAllString(line, fmt.Sprintf("[%d x i8]", capacity))
		return strings.Join(lines, "\n"), true, nil
	}
	return ir, false, nil
}
