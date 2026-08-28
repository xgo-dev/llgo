package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestRun(t *testing.T) {
	for _, args := range [][]string{nil, []string{"-major", "21"}} {
		var stdout, stderr bytes.Buffer
		if code := run(args, &stdout, &stderr); code != 0 {
			t.Fatalf("run(%q) exit code = %d, stderr = %q", args, code, stderr.String())
		}
		for _, want := range []string{
			"LLGO_LLVM_MAJOR=21\n",
			"ESP_CLANG_LLVM_MAJOR=21\n",
			"ESP_CLANG_VERSION=21.1.3_20260816\n",
			"ESP_CLANG_BASE_URL=https://github.com/goplus/espressif-llvm-project-prebuilt/releases/download/21.1.3_20260816\n",
			"ESP_CLANG_SHA256_DARWIN_AMD64=21159a4edb8948d83e1f73dfef394bca6941d0c4035da02f8c90ac59799893fa\n",
			"ESP_CLANG_SHA256_DARWIN_ARM64=a8c46104501c38a8a7359ec24bc4e9d646f9fec2bdb2b122cbbee78e060400d1\n",
			"ESP_CLANG_SHA256_LINUX_AMD64=582b787057c9e36e7d4db20aaed7bbba74c7ad0481489f034f09476703befbd5\n",
			"ESP_CLANG_SHA256_LINUX_ARM64=77f49d832e5f309ecd6baaf169c62e3b064b27f9bee5aedddb6e66c981d56f44\n",
		} {
			if !strings.Contains(stdout.String(), want) {
				t.Errorf("run(%q) output does not contain %q:\n%s", args, want, stdout.String())
			}
		}
		if stderr.Len() != 0 {
			t.Errorf("run(%q) stderr = %q", args, stderr.String())
		}
	}
}

func TestRunErrors(t *testing.T) {
	tests := []struct {
		args []string
		want string
	}{
		{args: []string{"-major", "19"}, want: "no LLGo LLVM payload for major version 19"},
		{args: []string{"-major", "invalid"}, want: "invalid value"},
	}
	for _, test := range tests {
		var stdout, stderr bytes.Buffer
		if code := run(test.args, &stdout, &stderr); code == 0 {
			t.Fatalf("run(%q) unexpectedly succeeded", test.args)
		}
		if stdout.Len() != 0 {
			t.Errorf("run(%q) stdout = %q", test.args, stdout.String())
		}
		if !strings.Contains(stderr.String(), test.want) {
			t.Errorf("run(%q) stderr = %q, want substring %q", test.args, stderr.String(), test.want)
		}
	}
}
