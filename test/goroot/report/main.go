/*
 * Copyright (c) 2026 The XGo Authors (xgo.dev). All rights reserved.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
)

var knownResults = map[string]string{
	"pass":               "✅",
	"expected-fail":      "🟡",
	"unexpected-fail":    "❌",
	"unexpected-pass":    "⚠️",
	"flaky-pass":         "🔀✅",
	"flaky-fail":         "🔀❌",
	"not-applicable":     "➖",
	"host-skip":          "⏭️",
	"resource-fail":      "💥",
	"configuration-fail": "🛠️",
}

var platformLabels = map[string]string{
	"darwin/arm64":        "🍎 🦾",
	"linux/amd64":         "🐧 🖥️",
	"windows-msvc/amd64":  "🪟 🟦 MSVC 🖥️",
	"windows-msvc/arm64":  "🪟 🟦 MSVC 🦾",
	"windows-msvc/386":    "🪟 🟦 MSVC 💾",
	"windows-mingw/amd64": "🪟 🦬 GNU 🖥️",
	"windows-mingw/arm64": "🪟 🦬 GNU 🦾",
	"windows-mingw/386":   "🪟 🦬 GNU 💾",
	"js/wasm":             "🕸️ 📜 JS",
	"wasip1/wasm":         "🕸️ 🔌 WASI",
}

var mismatches = map[string]bool{
	"unexpected-fail":    true,
	"unexpected-pass":    true,
	"resource-fail":      true,
	"configuration-fail": true,
}

type lane struct {
	platform string
	version  string
}

type caseResult struct {
	lane
	shard     string
	casePath  string
	directive string
	result    string
}

func main() {
	input := flag.String("input", "-", "tab-separated case result file, or - for stdin")
	platforms := flag.String("platforms", "", "comma-separated platforms in report order")
	versions := flag.String("versions", "", "comma-separated exact Go versions in report order")
	runURL := flag.String("run-url", "", "workflow run URL")
	mentionOnMismatch := flag.String("mention-on-mismatch", "", "GitHub user/team to mention when expectation mismatches occur")
	flag.Parse()

	if *platforms == "" || *versions == "" {
		fmt.Fprintln(os.Stderr, "goroot-report: -platforms and -versions are required")
		os.Exit(2)
	}
	var reader io.Reader = os.Stdin
	if *input != "-" {
		file, err := os.Open(*input)
		if err != nil {
			fmt.Fprintf(os.Stderr, "goroot-report: %v\n", err)
			os.Exit(1)
		}
		defer file.Close()
		reader = file
	}
	records, err := parseResults(reader)
	if err != nil {
		fmt.Fprintf(os.Stderr, "goroot-report: %v\n", err)
		os.Exit(1)
	}
	report, err := renderReport(records, splitList(*platforms), splitList(*versions), *runURL, *mentionOnMismatch)
	if err != nil {
		fmt.Fprintf(os.Stderr, "goroot-report: %v\n", err)
		os.Exit(1)
	}
	fmt.Print(report)
}

func parseResults(r io.Reader) ([]caseResult, error) {
	var results []caseResult
	scanner := bufio.NewScanner(r)
	for line := 1; scanner.Scan(); line++ {
		text := strings.TrimSpace(scanner.Text())
		if text == "" {
			continue
		}
		fields := strings.Split(text, "\t")
		if len(fields) != 6 {
			return nil, fmt.Errorf("line %d: got %d fields, want 6", line, len(fields))
		}
		if _, ok := knownResults[fields[5]]; !ok {
			return nil, fmt.Errorf("line %d: unknown result %q", line, fields[5])
		}
		results = append(results, caseResult{
			lane:      lane{platform: fields[0], version: strings.TrimPrefix(fields[1], "go")},
			shard:     fields[2],
			casePath:  fields[3],
			directive: fields[4],
			result:    fields[5],
		})
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return results, nil
}

func renderReport(results []caseResult, platforms, versions []string, runURL, mentionOnMismatch string) (string, error) {
	if len(platforms) == 0 || len(versions) == 0 {
		return "", errors.New("platform and version lists must not be empty")
	}
	lanes := make([]lane, 0, len(platforms)*len(versions))
	allowed := make(map[lane]bool, len(platforms)*len(versions))
	for _, platform := range platforms {
		for _, version := range versions {
			key := lane{platform: platform, version: strings.TrimPrefix(version, "go")}
			lanes = append(lanes, key)
			allowed[key] = true
		}
	}

	byCase := make(map[string]map[lane]caseResult)
	mismatchCases := make(map[string]bool)
	for _, result := range results {
		if !allowed[result.lane] {
			return "", fmt.Errorf("result for unconfigured lane %s Go %s", result.platform, result.version)
		}
		laneResults := byCase[result.casePath]
		if laneResults == nil {
			laneResults = make(map[lane]caseResult)
			byCase[result.casePath] = laneResults
		}
		if previous, ok := laneResults[result.lane]; ok {
			return "", fmt.Errorf("duplicate result for %s on %s Go %s (shards %s and %s)", result.casePath, result.platform, result.version, previous.shard, result.shard)
		}
		laneResults[result.lane] = result
		if mismatches[result.result] {
			mismatchCases[result.casePath] = true
		}
	}

	casePaths := make([]string, 0, len(mismatchCases))
	for casePath := range mismatchCases {
		casePaths = append(casePaths, casePath)
	}
	sort.Strings(casePaths)

	var out strings.Builder
	out.WriteString("## GOROOT expectation mismatches\n\n")
	if runURL != "" {
		fmt.Fprintf(&out, "[Workflow run](%s)\n\n", runURL)
	}
	out.WriteString("| Case |")
	for _, laneKey := range lanes {
		fmt.Fprintf(&out, " %s<br>%s |", laneLabel(laneKey), escapeMarkdown(goSeries(laneKey.version)))
	}
	out.WriteString("\n|---|")
	for range lanes {
		out.WriteString("---|")
	}
	out.WriteByte('\n')
	for _, casePath := range casePaths {
		fmt.Fprintf(&out, "| `%s` |", escapeCode(casePath))
		for _, laneKey := range lanes {
			value := "·"
			if result, ok := byCase[casePath][laneKey]; ok {
				value = knownResults[result.result]
			}
			fmt.Fprintf(&out, " %s |", value)
		}
		out.WriteByte('\n')
	}
	if len(casePaths) == 0 {
		out.WriteString("\nNo expected-pass failures or expected-fail passes were reported.\n")
	} else {
		out.WriteString("\n✅ pass · 🟡 expected failure · ❌ unexpected failure · ⚠️ unexpected pass · 🔀 flaky · ➖ not applicable · ⏭️ host skip · 💥 resource guard · 🛠️ configuration · `·` unavailable\n")
		if mentionOnMismatch != "" {
			fmt.Fprintf(&out, "\n> [!WARNING]\n> Expectation mismatches detected (unexpected failure / unexpected pass).\n> @%s please identify which PR introduced this issue.\n", strings.TrimPrefix(mentionOnMismatch, "@"))
		}
	}
	return out.String(), nil
}

func laneLabel(value lane) string {
	if label := platformLabels[value.platform]; label != "" {
		return label
	}
	return escapeMarkdown(value.platform)
}

func goSeries(version string) string {
	parts := strings.Split(strings.TrimPrefix(version, "go"), ".")
	if len(parts) >= 2 {
		return parts[0] + "." + parts[1]
	}
	return strings.TrimPrefix(version, "go")
}

func splitList(value string) []string {
	var values []string
	for _, item := range strings.Split(value, ",") {
		if item = strings.TrimSpace(item); item != "" {
			values = append(values, item)
		}
	}
	return values
}

func escapeMarkdown(value string) string {
	return strings.ReplaceAll(value, "|", "\\|")
}

func escapeCode(value string) string {
	return strings.ReplaceAll(value, "`", "\\`")
}
