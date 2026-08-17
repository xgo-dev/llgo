package flash

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/xgo-dev/llgo/internal/env"
	"github.com/xgo-dev/llgo/internal/shellparse"
	"go.bug.st/serial"
	"go.bug.st/serial/enumerator"
)

// Device contains all flashing/debugging configuration for a device
type Device struct {
	Serial     string   // Serial communication settings
	SerialPort []string // Available serial ports
	Flash      Flash    // Flash configuration for device programming
	MSD        MSD      // Mass Storage Device configuration
	OpenOCD    OpenOCD  // OpenOCD configuration for debugging/flashing
}

// Flash contains configuration for device flashing
type Flash struct {
	Method            string // Flash method: "command", "openocd", "msd", "bmp"
	Command           string // Flash command template
	Flash1200BpsReset bool   // Whether to use 1200bps reset
}

// MSD contains configuration for Mass Storage Device flashing
type MSD struct {
	VolumeName   []string // Names of the volumes
	FirmwareName string   // Firmware file name pattern
}

// OpenOCD contains configuration for OpenOCD debugging/flashing
type OpenOCD struct {
	Interface string // Interface configuration (e.g., "stlink")
	Transport string // Transport protocol (e.g., "swd", "jtag")
	Target    string // Target configuration (e.g., "stm32f4x")
}

// From tinygo/main.go getDefaultPort
// GetPort returns the default serial port depending on the operating system and USB interfaces.
func GetPort(portFlag string, usbInterfaces []string) (string, error) {
	portCandidates := strings.FieldsFunc(portFlag, func(c rune) bool { return c == ',' })
	if len(portCandidates) == 1 {
		return portCandidates[0], nil
	}

	var ports []string
	var err error
	switch runtime.GOOS {
	case "freebsd":
		ports, err = filepath.Glob("/dev/cuaU*")
	case "darwin", "linux", "windows":
		var portsList []*enumerator.PortDetails
		portsList, err = enumerator.GetDetailedPortsList()
		if err != nil {
			return "", err
		}

		var preferredPortIDs [][2]uint16
		for _, s := range usbInterfaces {
			parts := strings.Split(s, ":")
			if len(parts) != 2 {
				return "", fmt.Errorf("could not parse USB VID/PID pair %q", s)
			}
			vid, err := strconv.ParseUint(parts[0], 16, 16)
			if err != nil {
				return "", fmt.Errorf("could not parse USB vendor ID %q: %w", parts[0], err)
			}
			pid, err := strconv.ParseUint(parts[1], 16, 16)
			if err != nil {
				return "", fmt.Errorf("could not parse USB product ID %q: %w", parts[1], err)
			}
			preferredPortIDs = append(preferredPortIDs, [2]uint16{uint16(vid), uint16(pid)})
		}

		var primaryPorts []string   // ports picked from preferred USB VID/PID
		var secondaryPorts []string // other ports (as a fallback)
		for _, p := range portsList {
			if !p.IsUSB {
				continue
			}
			if p.VID != "" && p.PID != "" {
				foundPort := false
				vid, vidErr := strconv.ParseUint(p.VID, 16, 16)
				pid, pidErr := strconv.ParseUint(p.PID, 16, 16)
				if vidErr == nil && pidErr == nil {
					for _, id := range preferredPortIDs {
						if uint16(vid) == id[0] && uint16(pid) == id[1] {
							primaryPorts = append(primaryPorts, p.Name)
							foundPort = true
							break
						}
					}
				}
				if foundPort {
					continue
				}
			}

			secondaryPorts = append(secondaryPorts, p.Name)
		}
		if len(primaryPorts) == 1 {
			return primaryPorts[0], nil
		} else if len(primaryPorts) > 1 {
			ports = primaryPorts
		} else {
			ports = secondaryPorts
		}
	default:
		return "", errors.New("unable to search for a default USB device to be flashed on this OS")
	}

	if err != nil {
		return "", err
	} else if ports == nil {
		return "", errors.New("unable to locate a serial port")
	} else if len(ports) == 0 {
		return "", errors.New("no serial ports available")
	}

	if len(portCandidates) == 0 {
		if len(usbInterfaces) > 0 {
			return "", errors.New("unable to search for a default USB device - use -port flag, available ports are " + strings.Join(ports, ", "))
		} else if len(ports) == 1 {
			return ports[0], nil
		} else {
			return "", errors.New("multiple serial ports available - use -port flag, available ports are " + strings.Join(ports, ", "))
		}
	}

	for _, ps := range portCandidates {
		for _, p := range ports {
			if p == ps {
				return p, nil
			}
		}
	}

	return "", errors.New("port you specified '" + strings.Join(portCandidates, ",") + "' does not exist, available ports are " + strings.Join(ports, ", "))
}

// From tinygo/main.go touchSerialPortAt1200bps
// touchSerialPortAt1200bps triggers Arduino-compatible devices to enter bootloader mode.
// This function implements the Arduino auto-reset mechanism used before flashing firmware.
//
// Working principle:
// 1. Opens serial port at 1200 baud rate (special reset baudrate for Arduino)
// 2. Sets DTR (Data Terminal Ready) signal to false
// 3. This triggers the device to reset and enter bootloader mode for firmware upload
//
// Usage scenarios:
// - Required for Arduino Uno, Leonardo, Micro and other compatible devices
// - Executed when target config has "flash-1200-bps-reset": "true"
// - Ensures device is in correct state to receive new firmware
//
// Retry mechanism:
// - Retries up to 3 times due to potential temporary serial port access issues
// - Windows special handling: InvalidSerialPort error during bootloader transition is normal
func touchSerialPortAt1200bps(port string) (err error) {
	retryCount := 3
	for i := 0; i < retryCount; i++ {
		// Open port at 1200bps to trigger Arduino reset
		p, e := serial.Open(port, &serial.Mode{BaudRate: 1200})
		if e != nil {
			if runtime.GOOS == `windows` {
				se, ok := e.(*serial.PortError)
				if ok && se.Code() == serial.InvalidSerialPort {
					// InvalidSerialPort error occurs when transitioning to boot
					return nil
				}
			}
			time.Sleep(1 * time.Second)
			err = e
			continue
		}
		defer p.Close()

		// Set DTR to false to trigger reset
		p.SetDTR(false)
		return nil
	}
	return fmt.Errorf("opening port: %s", err)
}

// FlashDevice flashes firmware to a device based on the device configuration
func FlashDevice(device Device, envMap map[string]string, port string, verbose bool) error {
	method := device.Flash.Method
	if method == "" {
		method = "command"
	}

	// Resolve port for methods that need it (all except openocd)
	if method != "openocd" {
		var err error
		port, err = GetPort(port, device.SerialPort)
		if err != nil {
			return fmt.Errorf("failed to find port: %w", err)
		}
	}

	if verbose {
		fmt.Fprintf(os.Stderr, "Flashing using method: %s\n", method)
		fmt.Fprintf(os.Stderr, "Using port: %s\n", port)
	}

	// Execute 1200bps reset before flashing if needed (except for openocd)
	if method != "openocd" && device.Flash.Flash1200BpsReset {
		if verbose {
			fmt.Fprintf(os.Stderr, "Triggering 1200bps reset on port: %s\n", port)
		}
		if err := touchSerialPortAt1200bps(port); err != nil {
			return fmt.Errorf("failed to trigger 1200bps reset: %w", err)
		}
		// Wait a bit for device to enter bootloader mode
		time.Sleep(2 * time.Second)
	}

	switch method {
	case "command":
		return flashCommand(device.Flash, envMap, port, verbose)
	case "openocd":
		return flashOpenOCD(device.OpenOCD, envMap, verbose)
	case "msd":
		return flashMSD(device.MSD, envMap, verbose)
	case "bmp":
		return flashBMP(envMap, port, verbose)
	default:
		return fmt.Errorf("unsupported flash method: %s", method)
	}
}

// flashCommand handles command-based flashing
func flashCommand(flash Flash, envMap map[string]string, port string, verbose bool) error {
	if flash.Command == "" {
		return fmt.Errorf("flash command not specified")
	}

	// Build environment map for template variable expansion
	envs := buildFlashEnvMap(envMap, port)

	// Expand template variables in command
	expandedCommand := env.ExpandEnvWithDefault(flash.Command, envs)

	if verbose {
		fmt.Fprintf(os.Stderr, "Flash command: %s\n", expandedCommand)
	}

	// Split command into parts for exec - safely handling quoted arguments
	parts, err := shellparse.Parse(expandedCommand)
	if err != nil {
		return fmt.Errorf("failed to parse flash command: %w", err)
	}
	if len(parts) == 0 {
		return fmt.Errorf("empty flash command after expansion")
	}

	// Execute flash command
	cmd := exec.Command(parts[0], parts[1:]...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

// flashOpenOCD handles OpenOCD-based flashing
func flashOpenOCD(openocd OpenOCD, envMap map[string]string, verbose bool) error {
	if openocd.Interface == "" {
		return fmt.Errorf("OpenOCD interface not specified")
	}

	args := []string{
		"-f", "interface/" + openocd.Interface + ".cfg",
	}

	if openocd.Transport != "" {
		args = append(args, "-c", "transport select "+openocd.Transport)
	}

	if openocd.Target != "" {
		args = append(args, "-f", "target/"+openocd.Target+".cfg")
	}

	// Add programming commands
	args = append(args,
		"-c", "init",
		"-c", "reset init",
		"-c", fmt.Sprintf("flash write_image erase %s", envMap["elf"]),
		"-c", "reset",
		"-c", "shutdown",
	)

	if verbose {
		fmt.Fprintf(os.Stderr, "OpenOCD command: openocd %s\n", strings.Join(args, " "))
	}

	cmd := exec.Command("openocd", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

// flashMSD handles Mass Storage Device flashing
func flashMSD(msd MSD, envMap map[string]string, verbose bool) error {
	if len(msd.VolumeName) == 0 {
		return fmt.Errorf("MSD volume names not specified")
	}

	if msd.FirmwareName == "" {
		return fmt.Errorf("MSD firmware name not specified")
	}

	// Find the MSD volume
	var mountPoint string
	for _, volumeName := range msd.VolumeName {
		// Try platform-specific mount points
		var candidates []string
		switch runtime.GOOS {
		case "darwin":
			candidates = []string{
				filepath.Join("/Volumes", volumeName),
			}
		case "linux":
			candidates = []string{
				filepath.Join("/media", os.Getenv("USER"), volumeName),
				filepath.Join("/mnt", volumeName),
			}
		case "windows":
			candidates = []string{
				volumeName + ":",
			}
		default:
			candidates = []string{
				filepath.Join("/Volumes", volumeName),                  // macOS
				filepath.Join("/media", os.Getenv("USER"), volumeName), // Linux
				filepath.Join("/mnt", volumeName),                      // Linux alternative
				volumeName + ":",                                       // Windows
			}
		}

		for _, candidate := range candidates {
			if _, err := os.Stat(candidate); err == nil {
				mountPoint = candidate
				break
			}
		}
		if mountPoint != "" {
			break
		}
	}

	if mountPoint == "" {
		return fmt.Errorf("MSD volume not found. Expected volumes: %v", msd.VolumeName)
	}

	// Copy firmware to MSD
	destPath := filepath.Join(mountPoint, msd.FirmwareName)

	if verbose {
		fmt.Fprintf(os.Stderr, "Copying %s to %s\n", envMap["uf2"], destPath)
	}

	return copyFile(envMap["uf2"], destPath)
}

// flashBMP handles Black Magic Probe flashing
func flashBMP(envMap map[string]string, port string, verbose bool) error {
	// BMP typically uses GDB for flashing
	args := []string{
		"-ex", "target extended-remote " + port,
		"-ex", "monitor swdp_scan",
		"-ex", "attach 1",
		"-ex", "load",
		"-ex", "compare-sections",
		"-ex", "kill",
		"-ex", "quit",
		envMap["elf"],
	}

	if verbose {
		fmt.Fprintf(os.Stderr, "BMP command: arm-none-eabi-gdb %s\n", strings.Join(args, " "))
	}

	cmd := exec.Command("arm-none-eabi-gdb", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

// buildFlashEnvMap creates environment map for template expansion
func buildFlashEnvMap(envMap map[string]string, port string) map[string]string {
	envs := make(map[string]string)

	// Basic paths
	envs["root"] = env.LLGoROOT()
	envs["tmpDir"] = os.TempDir()

	// Port information
	if port != "" {
		envs["port"] = port
	}

	// Copy all format paths from envMap
	for key, value := range envMap {
		if value != "" {
			envs[key] = value
		}
	}

	return envs
}

// copyFile copies a file from src to dst
func copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = destFile.ReadFrom(sourceFile)
	return err
}
