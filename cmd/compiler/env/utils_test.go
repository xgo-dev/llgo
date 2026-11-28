package env

import "testing"

func TestExpandEnvSlice(t *testing.T) {
	envs := map[string]string{
		"root": "/usr/local/llgo",
		"port": "/dev/ttyUSB0",
	}

	input := []string{
		"-I{root}/include",
		"-DPORT={port}",
		"static-flag",
	}

	expected := []string{
		"-I/usr/local/llgo/include",
		"-DPORT=/dev/ttyUSB0",
		"static-flag",
	}

	result := ExpandEnvSlice(input, envs)

	if len(result) != len(expected) {
		t.Fatalf("expandEnvSlice length mismatch: got %d, want %d", len(result), len(expected))
	}

	for i, exp := range expected {
		if result[i] != exp {
			t.Errorf("expandEnvSlice[%d] = %q, want %q", i, result[i], exp)
		}
	}
}

func TestExpandEnvWithDefault(t *testing.T) {
	envs := map[string]string{
		"port": "/dev/ttyUSB0",
		"hex":  "firmware.hex",
		"bin":  "firmware.bin",
		"img":  "image.img",
	}

	tests := []struct {
		template     string
		defaultValue string
		expected     string
	}{
		{
			"simavr {}",
			"firmware.hex",
			"simavr firmware.hex",
		},
		{
			"simavr {}",
			"custom.elf", // Explicit default
			"simavr custom.elf",
		},
		{
			"qemu -kernel {}",
			"vmlinux", // Custom kernel
			"qemu -kernel vmlinux",
		},
		{
			"no braces here",
			"ignored",
			"no braces here",
		},
	}

	for i, test := range tests {
		var result string
		if test.defaultValue == "" {
			result = ExpandEnvWithDefault(test.template, envs)
		} else {
			result = ExpandEnvWithDefault(test.template, envs, test.defaultValue)
		}

		if result != test.expected {
			t.Errorf("Test %d: expandEnvWithDefault(%q, envs, %q) = %q, want %q",
				i, test.template, test.defaultValue, result, test.expected)
		}
	}
}
