package build

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/xgo-dev/llgo/internal/pclnpost"
)

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
		if err != nil || analysis.TextEnd <= analysis.TextStart ||
			analysis.TextEnd-analysis.TextStart > uint64(^uint32(0)) || uint64(len(analysis.EntrySites)) > uint64(^uint32(0)) {
			return "", false, fmt.Errorf("invalid symbol entry capacity or text range")
		}
		buckets := (analysis.TextEnd-analysis.TextStart)/4096 + 2
		need := uint64(32) + (uint64(len(analysis.EntrySites))+1)*8 + buckets*36
		if capacity >= need {
			return ir, false, nil
		}
		capacity = (need + 4095) &^ uint64(4095)
		lines[i] = symbolEntryArrayType.ReplaceAllString(line, fmt.Sprintf("[%d x i8]", capacity))
		return strings.Join(lines, "\n"), true, nil
	}
	return ir, false, nil
}
