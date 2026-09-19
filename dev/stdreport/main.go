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
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type nativeShard struct {
	Platform   string
	Lane       string
	GoVersion  string
	ShardIndex string
	ShardTotal string
	Status     string
}

type wasmPackage struct {
	Package string `json:"package"`
	Status  string `json:"status"`
	Reason  string `json:"reason,omitempty"`
}

type wasmReport struct {
	Profile  string        `json:"profile"`
	Result   string        `json:"result"`
	Shard    int           `json:"shard"`
	Shards   int           `json:"shards"`
	Packages []wasmPackage `json:"packages"`
}

type wasmProfileSummary struct {
	Profile       string
	ShardsFound   int
	ShardsTotal   int
	Total         int
	Passed        int
	NotApplicable int
	Failed        int
	FailedPkgs    []wasmPackage
}

func main() {
	if err := run(os.Args[1:], os.Stdout); err != nil {
		fmt.Fprintf(os.Stderr, "stdreport: %v\n", err)
		os.Exit(1)
	}
}

func run(args []string, stdout io.Writer) error {
	flags := flag.NewFlagSet("stdreport", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	nativeDir := flags.String("native-reports", "", "directory containing native summary TSV files")
	wasmDir := flags.String("wasm-reports", "", "directory containing wasm acceptance JSON files")
	runURL := flags.String("run-url", "", "workflow run URL")
	mention := flags.String("mention-on-failure", "", "GitHub user/team to mention on failure")
	outputFile := flags.String("output", "-", "output markdown file path or - for stdout")
	nativeRes := flags.String("native-result", "success", "result of native test matrix")
	wasmRes := flags.String("wasm-result", "success", "result of wasm-std matrix")
	browserRes := flags.String("browser-result", "success", "result of wasm-browser test")
	sentinelsRes := flags.String("sentinels-result", "success", "result of wasm-sentinels test")
	if err := flags.Parse(args); err != nil {
		return err
	}

	var nativeShards []nativeShard
	if *nativeDir != "" {
		var err error
		nativeShards, err = loadNativeShards(*nativeDir)
		if err != nil {
			return fmt.Errorf("load native shards: %w", err)
		}
	}

	var wasmSummaries []wasmProfileSummary
	if *wasmDir != "" {
		var err error
		wasmSummaries, err = loadWasmReports(*wasmDir)
		if err != nil {
			return fmt.Errorf("load wasm reports: %w", err)
		}
	}

	report := generateReport(
		nativeShards,
		wasmSummaries,
		*runURL,
		*mention,
		*nativeRes,
		*wasmRes,
		*browserRes,
		*sentinelsRes,
	)

	if *outputFile == "-" || *outputFile == "" {
		_, err := fmt.Fprint(stdout, report)
		return err
	} else {
		if err := os.WriteFile(*outputFile, []byte(report), 0644); err != nil {
			return fmt.Errorf("write output: %w", err)
		}
	}
	return nil
}

func loadNativeShards(dir string) ([]nativeShard, error) {
	var shards []nativeShard
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(d.Name(), ".tsv") {
			return nil
		}
		f, err := os.Open(path)
		if err != nil {
			return err
		}
		defer f.Close()

		scanner := bufio.NewScanner(f)
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line == "" {
				continue
			}
			parts := strings.Split(line, "\t")
			if len(parts) >= 6 {
				shards = append(shards, nativeShard{
					Platform:   parts[0],
					Lane:       parts[1],
					GoVersion:  parts[2],
					ShardIndex: parts[3],
					ShardTotal: parts[4],
					Status:     parts[5],
				})
			}
		}
		return scanner.Err()
	})
	return shards, err
}

func loadWasmReports(dir string) ([]wasmProfileSummary, error) {
	byProfile := make(map[string]*wasmProfileSummary)
	profileOrder := []string{"J32-GoJS", "J32-Emscripten", "J64-Emscripten", "W32-WASI"}
	for _, p := range profileOrder {
		expectedShards := 3
		if p == "W32-WASI" {
			expectedShards = 5
		}
		byProfile[p] = &wasmProfileSummary{
			Profile:     p,
			ShardsTotal: expectedShards,
		}
	}

	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(d.Name(), ".json") {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		var rep wasmReport
		if err := json.Unmarshal(data, &rep); err != nil {
			return nil
		}
		summary, ok := byProfile[rep.Profile]
		if !ok {
			summary = &wasmProfileSummary{
				Profile:     rep.Profile,
				ShardsTotal: rep.Shards,
			}
			byProfile[rep.Profile] = summary
			profileOrder = append(profileOrder, rep.Profile)
		}
		summary.ShardsFound++
		for _, pkg := range rep.Packages {
			if pkg.Status == "other-shard" {
				continue
			}
			summary.Total++
			switch pkg.Status {
			case "pass":
				summary.Passed++
			case "not-applicable", "separate-suite":
				summary.NotApplicable++
			default:
				summary.Failed++
				summary.FailedPkgs = append(summary.FailedPkgs, pkg)
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	var res []wasmProfileSummary
	for _, p := range profileOrder {
		if s, ok := byProfile[p]; ok {
			sort.Slice(s.FailedPkgs, func(i, j int) bool {
				return s.FailedPkgs[i].Package < s.FailedPkgs[j].Package
			})
			res = append(res, *s)
		}
	}
	return res, nil
}

func generateReport(
	nativeShards []nativeShard,
	wasmSummaries []wasmProfileSummary,
	runURL, mention, nativeRes, wasmRes, browserRes, sentinelsRes string,
) string {
	var out strings.Builder
	out.WriteString("## Standard Library test summary\n\n")
	if runURL != "" {
		fmt.Fprintf(&out, "[Workflow run](%s)\n\n", runURL)
	}

	hasFailures := false

	// Check overall conclusions
	if nativeRes != "success" || wasmRes != "success" || browserRes != "success" || sentinelsRes != "success" {
		hasFailures = true
	}

	// Native table
	if len(nativeShards) > 0 {
		out.WriteString("### Native Standard Library\n\n")
		out.WriteString("| Platform | Lane | Go Version | Shard | Status |\n")
		out.WriteString("|---|---|---:|---:|---:|\n")
		for _, s := range nativeShards {
			statusIcon := "✅"
			if s.Status != "success" {
				statusIcon = "❌"
				hasFailures = true
			}
			shardText := fmt.Sprintf("%s/%s", s.ShardIndex, s.ShardTotal)
			fmt.Fprintf(&out, "| %s | %s | %s | %s | %s %s |\n",
				escapeMarkdown(s.Platform), escapeMarkdown(s.Lane), s.GoVersion, shardText, statusIcon, s.Status)
		}
		out.WriteString("\n")
	}

	// Wasm table
	if len(wasmSummaries) > 0 {
		out.WriteString("### WebAssembly Standard Library Acceptance\n\n")
		out.WriteString("| Profile | Shards | Total | Passed | Not-Applicable | Failed / Unresolved |\n")
		out.WriteString("|---|---:|---:|---:|---:|---:|\n")
		var allFailed []wasmPackage
		for _, w := range wasmSummaries {
			if w.Failed > 0 || (w.ShardsTotal > 0 && w.ShardsFound < w.ShardsTotal) {
				hasFailures = true
			}
			shardStr := fmt.Sprintf("%d/%d", w.ShardsFound, w.ShardsTotal)
			fmt.Fprintf(&out, "| %s | %s | %d | %d | %d | %d |\n",
				w.Profile, shardStr, w.Total, w.Passed, w.NotApplicable, w.Failed)
			for _, pkg := range w.FailedPkgs {
				allFailed = append(allFailed, pkg)
			}
		}
		out.WriteString("\n")

		if len(allFailed) > 0 {
			out.WriteString("**Failed wasm packages:**\n\n")
			for _, f := range allFailed {
				reason := f.Reason
				if reason == "" {
					reason = "unspecified failure"
				}
				fmt.Fprintf(&out, "- `%s`: %s\n", f.Package, escapeMarkdown(reason))
			}
			out.WriteString("\n")
		}
	}

	// Host integration table
	out.WriteString("### WebAssembly Host Integration\n\n")
	out.WriteString("| Suite | Result |\n")
	out.WriteString("|---|---:|\n")
	browserIcon := "✅"
	if browserRes != "success" {
		browserIcon = "❌"
		hasFailures = true
	}
	sentinelsIcon := "✅"
	if sentinelsRes != "success" {
		sentinelsIcon = "❌"
		hasFailures = true
	}
	fmt.Fprintf(&out, "| 🌐 Real-browser GoJS callback (Chrome) | %s %s |\n", browserIcon, browserRes)
	fmt.Fprintf(&out, "| 🏛️ Wasm GOROOT sentinels | %s %s |\n\n", sentinelsIcon, sentinelsRes)

	if hasFailures {
		if mention != "" {
			fmt.Fprintf(&out, "> [!WARNING]\n> Standard library test failures detected.\n> @%s please identify which PR introduced this issue.\n", strings.TrimPrefix(mention, "@"))
		}
	} else {
		out.WriteString("All standard library tests passed successfully.\n")
	}

	return out.String()
}

func escapeMarkdown(value string) string {
	return strings.ReplaceAll(value, "|", "\\|")
}
