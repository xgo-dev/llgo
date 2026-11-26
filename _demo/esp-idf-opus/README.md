# llgo × ESP-IDF Opus codec acceptance (ESP32-S31)

A self-contained pipeline that drives the Opus codec from Go code compiled by
llgo: encode a synthetic tone into Opus packets, decode them back, compare
segment by segment (SNR / max error / RMS, aligned by the codec's own
lookahead), exercise packet-loss concealment and the encoder ctls, and gate
the whole thing on the firmware's own PASS verdict. The Go binding lives in
the llgo repository (`pkg/esp/opus`, consumed through the module require +
replace in `go.mod`); this directory owns the acceptance program and the IDF
project.

No peer process, no network, no board: the acceptance run is one esp-emu
boot with a self-contained probe, so this is the smallest full-stack gate of
the pkg/esp audio family.

## What is under test

| segment | what it proves |
| --- | --- |
| `tone48k` | 48 kHz mono, music application, 48 kbps, 50 × 20 ms frames: encode→decode SNR 39.3 dB (gate 25 dB), max sample error 399 of a 12000-amplitude sine, plus packet-loss concealment (a nil frame fills exactly one frame and the stream resyncs) |
| `voip16k` | 16 kHz mono, voip application, 16 kbps: SNR 13.4 dB (gate 10 dB) - SILK's perceptual noise shaping is tuned for speech formants and legitimately loses dB on a pure tone; the gate catches misalignment/broken decode (those read < 5 dB with max error at the sine amplitude), plus a max-error guard at half amplitude on both segments |
| ctls | complexity 11 refuses with the codec's own OPUS_BAD_ARG, frame-duration table moves Encode's consumption and refuses non-Opus durations and short frames |

The decoded stream is compared against the encode input **shifted by the
codec's lookahead** (`opus.Encoder.Lookahead()`, 312 samples at 48 kHz,
104 at 16 kHz = 6.5 ms): comparing in place reads SNR 0 against a perfectly
good codec - the first attempt of this probe did exactly that.

## Layout

```
esp-idf-opus/
├── go.mod                  Go module (module llgoopus; requires + replaces
│                           github.com/xgo-dev/llgo, the binding package's home)
├── goapp/main.go           Acceptance probe: validation → state sizes →
│                           tone48k (+PLC) → voip16k → ctl checks → PASS
├── goapp/hosted.go         Host-build push-linkname stubs, so `go build` links
├── goapp/log.go            errText: esperr rendering + the package's plain-
│                           language ErrText sentences
├── golden/emu-opus-probe.txt  Golden for the [go] opus lines (+ floater rule)
└── idfproj/                Minimal ESP-IDF project
    ├── CMakeLists.txt      (generated from _demo/common/)
    ├── sdkconfig.defaults  PSRAM (16 KiB+ mallocs prefer it: the codec
    │                       states land there), 240 MHz, 3.9 MiB app partition
    ├── partitions.csv      (generated)
    ├── main/
    │   ├── CMakeLists.txt  Sources + WHOLE_ARCHIVE + REQUIRES 78__esp-opus
    │   ├── idf_component.yml  the codec: managed component 78/esp-opus 1.0.5
    │   │                   (vanilla upstream Opus, compiled fixed-point
    │   │                   -DFIXED_POINT=1 -DDISABLE_FLOAT_API; the registry
    │   │                   has no "espressif/opus" component)
    │   ├── demo_glue.c     C-side heap ledger + the fflush(stdout) console
    │   │                   promptness; checked in
    │   ├── opus_glue.c     pkg/esp/opus glue template, synced at build time
    │   │                   (gitignored): handle table + opus_* translations
    │   └── app_main.c, llgo_dram.ld  (generated from _demo/common/)
    └── host/
        ├── make-artifact.sh  (symlink to the shared pipeline)
        ├── artifact.conf   GLUES=opus, 128 KiB GC window,
        │                   -goroutine-stack-size 16384 (see Stacks below)
        └── llgo_syms.S     (generated; also provides llgoFixStreams)
```

## Build and run

Prerequisites: an llgo compiler (`/tmp/llgo-opus` below, built with
`LLVM_CONFIG=/opt/homebrew/opt/llvm/bin/llvm-config go build -o /tmp/llgo-opus ./cmd/llgo`),
Docker image `espressif/idf:latest` (pins the IDF v6.3-dev API), esp-emu
v1.2.0.

```
# 1. Compile the Go probe into the IDF-linkable object (also syncs the glue
#    and the scaffold into idfproj/main/, so run it before idf.py).
cd _demo/esp-idf-opus
LLGO_ROOT=$(cd ../.. && pwd) bash idfproj/host/make-artifact.sh /tmp/llgo-opus

# 2. IDF firmware (set-target pins the chip on the first build; merge-bin
#    lands the image at build/merged-binary.bin).
docker run --rm -v "$PWD/idfproj":/project -w /project \
    espressif/idf:latest bash -c \
    'idf.py set-target esp32s31 && idf.py build && idf.py merge-bin -o merged-binary.bin'

# 3. Acceptance run under the emulator (the probe exits on its done marker).
timeout 150 ~/.local/bin/esp-emu --chip esp32s31 \
    --firmware idfproj/build/merged-binary.bin --console uart0 \
    --psram-size 16M --timeout 120s --exit-on "[go] opus done"

# 4. Golden check.
grep -E '^\[go\] opus' <emu log> | tr -d '\r' \
    | sed -E 's/enc = [0-9]+/enc = */; s/snr = [0-9.]+/snr = */; \
              s/max = [0-9]+/max = */; s/rms = [0-9]+/rms = */' \
    | diff - <(grep -v '^#' golden/emu-opus-probe.txt)
```

Expected console output (interleaved with the IDF boot log; `[c]` lines are
the C side's heap marks):

```
[go] opus start
[c] opus heap boot internal=274759 psram=16774636
[go] opus validation ok rate44100+ch3-refused
[go] opus states enc1 = 24716 dec1 = 18136 enc2 = 29628 dec2 = 26856
[go] opus seg tone48k frames = 50 enc = 6101 dec = 48000 snr = 39.3 dB max = 399 rms = 91 delay = 312
[go] opus plc conceal = 960 of 960
[go] opus plc ok resync = 960
[go] opus seg tone48k PASS
[go] opus seg voip16k frames = 50 enc = 2069 dec = 16000 snr = 13.4 dB max = 5208 rms = 1772 delay = 104
[go] opus seg voip16k PASS
[go] opus ctl ok bad-arg frame-table length-guard
[go] opus PASS
[go] opus done
```

## Memory ledger (measured, esp-emu, 128 KiB GC window)

| stage | internal free | psram free | note |
| --- | --- | --- | --- |
| boot | 274,759 B | 16,774,636 B | 128 KiB static GC window placed in internal SRAM |
| seg1 done | 274,759 B | 15,707,620 B | -1,067,016 B of PSRAM: the two codec states (24.7 + 18.1 KiB, IDF-malloc'd, PSRAM-backed at this sdkconfig) plus the collector's first PSRAM segment (the Go heap grew for the probe's stream buffer and the 64 KiB main fiber) |

Codec C state sizes (arithmetic, `opus_*_get_size`, mono): encoder 24,716 B,
decoder 18,136 B; stereo: 29,628 B / 26,856 B. Go-side per segment: the pcm
frame (1.9 KiB), the output frame buffer (2.9 KiB), the whole-input stream
copy (50 frames: 96 KiB at 48 kHz) — the stream copy exists only for the
lookahead-aligned comparison; a streaming consumer compares shifted-in-place
and never holds the whole input.

## Stacks (the one measured trap)

opus_encode/opus_decode are deep C calls (alloca scratch in the tens of
KiB). The probe's goroutine runs on a fiber stack from the GC heap — the
64 KiB `CONFIG_ESP_MAIN_TASK_STACK_SIZE` everyone reaches for first is the
FreeRTOS task's stack and has nothing to do with it. The first firmware of
this demo (default stacks: main fiber 16 KiB) died exactly that way: the
deep call wrote below the fiber floor unchecked (the fiber guard only fires
at cooperative swap points) and corrupted the allocator's globals; the next
Go allocation panicked with `gc: invalid heap block`. The fix is
`-goroutine-stack-size 16384` in `host/artifact.conf` (main fiber = 4x =
64 KiB) paired with a 128 KiB GC window big enough to hold that fiber. The
stack discipline is a pkg/esp/opus contract, documented in the package docs.

## Known limitations

- The registry component is `78/esp-opus` 1.0.5, not "espressif/opus" (which
  does not exist); it is vanilla upstream Opus compiled always fixed-point.
- The binding exposes integer-millisecond frame durations only ({5,10,20,
  40,60} ms; Opus also allows 2.5 ms).
- Two encoder and two decoder slots in the glue's handle table; more report
  ESP_ERR_NO_MEM.
- Decode concealment (nil frame) requires pcmOut/channels to be a multiple
  of 2.5 ms in samples (the codec's own rule).
- `fflush(stdout)`-anchored console: the Go print path's streams are
  repointed at the IDF's by `llgoFixStreams` (host/llgo_syms.S), called at
  the top of the probe — a runtime-layer mechanism fix is planned, after
  which the call and the shim are removed (see the TODO in goapp/main.go).
