package monitor

import (
	"debug/dwarf"
	"debug/elf"
	"debug/macho"
	"debug/pe"
	"errors"
	"fmt"
	"go/token"
	"io"
	"os"
	"os/signal"
	"regexp"
	"strconv"
	"time"

	"github.com/mattn/go-tty"
	"github.com/xgo-dev/llgo/internal/flash"
	"go.bug.st/serial"
)

// MonitorConfig contains configuration for the monitor
type MonitorConfig struct {
	Port       string   // Serial port device
	Target     string   // Target name for crosscompile config
	BaudRate   int      // Baudrate of serial monitor
	Executable string   // Optional path to executable for debug info
	WaitTime   int      // Wait time for port connection (ms)
	SerialPort []string // List of serial ports to use
}

// Monitor starts serial monitoring with the given configuration
func Monitor(config MonitorConfig, verbose bool) error {
	// Set defaults
	if config.BaudRate == 0 {
		config.BaudRate = 115200
	}
	if config.WaitTime == 0 {
		config.WaitTime = 300
	}

	// Resolve port using flash.GetPort
	port, err := flash.GetPort(config.Port, config.SerialPort)
	if err != nil {
		return fmt.Errorf("failed to find port: %w", err)
	}
	config.Port = port

	if verbose {
		fmt.Fprintf(os.Stderr, "Connecting to %s at %d baud\n", config.Port, config.BaudRate)
	}

	// Open serial port with retry
	var serialConn serial.Port
	for i := 0; i <= config.WaitTime; i++ {
		serialConn, err = serial.Open(config.Port, &serial.Mode{BaudRate: config.BaudRate})
		if err != nil {
			if i < config.WaitTime {
				time.Sleep(10 * time.Millisecond)
				continue
			}
			return fmt.Errorf("failed to open port %s: %w", config.Port, err)
		}
		break
	}
	defer serialConn.Close()

	// Open TTY for input
	tty, err := tty.Open()
	if err != nil {
		return fmt.Errorf("failed to open TTY: %w", err)
	}
	defer tty.Close()

	// Setup signal handling
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt)
	defer signal.Stop(sig)

	// Create output writer with optional debug info
	var writer *outputWriter
	if config.Executable != "" {
		writer = newOutputWriter(os.Stdout, config.Executable)
	} else {
		writer = newOutputWriter(os.Stdout, "")
	}

	fmt.Printf("Connected to %s. Press Ctrl-C to exit.\n", config.Port)

	errCh := make(chan error, 2)

	// Goroutine for reading from serial port
	go func() {
		buf := make([]byte, 100*1024)
		for {
			n, err := serialConn.Read(buf)
			if err != nil {
				errCh <- fmt.Errorf("serial read error: %w", err)
				return
			}
			writer.Write(buf[:n])
		}
	}()

	// Goroutine for reading from TTY and writing to serial port
	go func() {
		for {
			r, err := tty.ReadRune()
			if err != nil {
				errCh <- fmt.Errorf("TTY read error: %w", err)
				return
			}
			if r == 0 {
				continue
			}
			serialConn.Write([]byte(string(r)))
		}
	}()

	// Wait for signal or error
	select {
	case <-sig:
		if verbose {
			fmt.Fprintf(os.Stderr, "\nDisconnected from %s\n", config.Port)
		}
		return nil
	case err := <-errCh:
		return err
	}
}

var addressMatch = regexp.MustCompile(`^panic: runtime error at 0x([0-9a-f]+): `)

// Extract the address from the "panic: runtime error at" message.
func extractPanicAddress(line []byte) uint64 {
	matches := addressMatch.FindSubmatch(line)
	if matches != nil {
		address, err := strconv.ParseUint(string(matches[1]), 16, 64)
		if err == nil {
			return address
		}
	}
	return 0
}

// Convert an address in the binary to a source address location.
func addressToLine(executable string, address uint64) (token.Position, error) {
	data, err := readDWARF(executable)
	if err != nil {
		return token.Position{}, err
	}
	r := data.Reader()

	for {
		e, err := r.Next()
		if err != nil {
			return token.Position{}, err
		}
		if e == nil {
			break
		}
		switch e.Tag {
		case dwarf.TagCompileUnit:
			r.SkipChildren()
			lr, err := data.LineReader(e)
			if err != nil {
				return token.Position{}, err
			}
			var lineEntry = dwarf.LineEntry{
				EndSequence: true,
			}
			for {
				// Read the next .debug_line entry.
				prevLineEntry := lineEntry
				err := lr.Next(&lineEntry)
				if err != nil {
					if err == io.EOF {
						break
					}
					return token.Position{}, err
				}

				if prevLineEntry.EndSequence && lineEntry.Address == 0 {
					// Tombstone value. This symbol has been removed, for
					// example by the --gc-sections linker flag. It is still
					// here in the debug information because the linker can't
					// just remove this reference.
					// Read until the next EndSequence so that this sequence is
					// skipped.
					// For more details, see (among others):
					// https://reviews.llvm.org/D84825
					for {
						err := lr.Next(&lineEntry)
						if err != nil {
							return token.Position{}, err
						}
						if lineEntry.EndSequence {
							break
						}
					}
				}

				if !prevLineEntry.EndSequence {
					// The chunk describes the code from prevLineEntry to
					// lineEntry.
					if prevLineEntry.Address <= address && lineEntry.Address > address {
						return token.Position{
							Filename: prevLineEntry.File.Name,
							Line:     prevLineEntry.Line,
							Column:   prevLineEntry.Column,
						}, nil
					}
				}
			}
		}
	}

	return token.Position{}, nil // location not found
}

// Read the DWARF debug information from a given file (in various formats).
func readDWARF(executable string) (*dwarf.Data, error) {
	f, err := os.Open(executable)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	if file, err := elf.NewFile(f); err == nil {
		return file.DWARF()
	} else if file, err := macho.NewFile(f); err == nil {
		return file.DWARF()
	} else if file, err := pe.NewFile(f); err == nil {
		return file.DWARF()
	} else {
		return nil, errors.New("unknown binary format")
	}
}

type outputWriter struct {
	out        io.Writer
	executable string
	line       []byte
}

// newOutputWriter returns an io.Writer that will intercept panic addresses and
// will try to insert a source location in the output if the source location can
// be found in the executable.
func newOutputWriter(out io.Writer, executable string) *outputWriter {
	return &outputWriter{
		out:        out,
		executable: executable,
	}
}

func (w *outputWriter) Write(p []byte) (n int, err error) {
	start := 0
	for i, c := range p {
		if c == '\n' {
			w.out.Write(p[start : i+1])
			start = i + 1
			if w.executable != "" {
				address := extractPanicAddress(w.line)
				if address != 0 {
					loc, err := addressToLine(w.executable, address)
					if err == nil && loc.Filename != "" {
						fmt.Printf("[llgo: panic at %s]\n", loc.String())
					}
				}
			}
			w.line = w.line[:0]
		} else {
			w.line = append(w.line, c)
		}
	}
	w.out.Write(p[start:])
	n = len(p)
	return
}
