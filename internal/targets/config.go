package targets

// Config represents a complete target configuration after inheritance resolution
type Config struct {
	// Target identification
	Name string `json:"-"`

	// LLVM configuration
	LLVMTarget string `json:"llvm-target"`
	CPU        string `json:"cpu"`
	Features   string `json:"features"`

	// Build configuration
	BuildTags []string `json:"build-tags"`
	GOOS      string   `json:"goos"`
	GOARCH    string   `json:"goarch"`

	// Compiler and linker configuration
	Libc         string   `json:"libc"`
	RTLib        string   `json:"rtlib"`
	JSLib        string   `json:"jslib"` // JavaScript engine library (e.g., "mquickjs")
	Linker       string   `json:"linker"`
	LinkerScript string   `json:"linkerscript"`
	CFlags       []string `json:"cflags"`
	LDFlags      []string `json:"ldflags"`
	ExtraFiles   []string `json:"extra-files"`

	// Code generation configuration
	CodeModel       string `json:"code-model"`
	TargetABI       string `json:"target-abi"`
	RelocationModel string `json:"relocation-model"`

	// Binary and firmware configuration
	BinaryFormat string `json:"binary-format"`
	// UF2 configuration
	UF2FamilyID string `json:"uf2-family-id"`

	// Flash and deployment configuration
	FlashMethod       string `json:"flash-method"`  // values: command, openocd, msd
	FlashCommand      string `json:"flash-command"` // used when FlashMethod == "command"
	Flash1200BpsReset string `json:"flash-1200-bps-reset"`

	// Serial configuration
	Serial     string   `json:"serial"`      // Serial communication type (e.g., "usb")
	SerialPort []string `json:"serial-port"` // Serial port identifiers (e.g., vendor:product IDs)

	// Mass storage device configuration
	MSDVolumeName   []string `json:"msd-volume-name"`
	MSDFirmwareName string   `json:"msd-firmware-name"`

	// Device-specific configuration
	RP2040BootPatch bool `json:"rp2040-boot-patch"`

	// Debug and emulation configuration
	Emulator string   `json:"emulator"`
	GDB      []string `json:"gdb"`

	// OpenOCD configuration
	OpenOCDInterface string `json:"openocd-interface"`
	OpenOCDTransport string `json:"openocd-transport"`
	OpenOCDTarget    string `json:"openocd-target"`
}

// RawConfig represents the raw JSON configuration before inheritance resolution
type RawConfig struct {
	Inherits []string `json:"inherits"`
	Config
}

// IsEmpty returns true if the config appears to be uninitialized
func (c *Config) IsEmpty() bool {
	return c.Name == "" && c.LLVMTarget == "" && c.GOOS == "" && c.GOARCH == ""
}

func (c *Config) FormatDetail() string {
	if c.BinaryFormat == "uf2" {
		return c.UF2FamilyID
	}
	return ""
}

// HasInheritance returns true if this config inherits from other configs
func (rc *RawConfig) HasInheritance() bool {
	return len(rc.Inherits) > 0
}

// GetInherits returns the list of configs this config inherits from
func (rc *RawConfig) GetInherits() []string {
	return rc.Inherits
}
