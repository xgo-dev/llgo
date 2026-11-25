# llgo × ESP-SR WakeNet pipeline (ESP32-S31): wake-word detection from Go

A self-contained pipeline that drives Espressif's ESP-SR WakeNet engine from
Go code compiled by llgo: the model partition is enumerated, the wn9_hilexin
engine is loaded, a dedicated FreeRTOS task feeds it audio and runs the
detections, and wake events arrive on a Go channel. The acceptance run needs
no microphone: the engine's input is the esp-sr test app's "Hi, Lexin"
utterance, replayed by the glue's built-in test source at real-time pace. The
Go bindings live in the llgo repository (`pkg/esp/wakenet`, consumed through
the module require + replace in `go.mod`); this directory owns the acceptance
program, the IDF project, and the pure-C smoke build that answers the same
question without llgo in the picture.

## Layout

```
esp-idf-wakenet/
├── go.mod                  Go module (module llgowakenet; requires + replaces
│                           github.com/xgo-dev/llgo, the binding package's home)
├── goapp/main.go           Acceptance probe: Init → test-PCM injection → Start →
│                           12 s of events/stats consumption → Stop → Close → PASS
├── goapp/hosted.go         Host-build push-linkname stubs, so `go build` links
├── csmoke/                 P0 pure-C smoke project (no llgo): the same engine
│                           against the same utterance, built and run before the
│                           binding is trusted
└── idfproj/                Minimal ESP-IDF project
    ├── sdkconfig.defaults  wn9_hilexin + SPIRAM + 240 MHz + 16 MB flash, the
    │                       esp-sr S31 CI switches on top of the demo base
    ├── partitions.csv      generated; host/post-sync.sh writes the model
    │                       partition (factory 8000K + model 7900K) over the
    │                       common 4 MB table
    ├── host/
    │   ├── make-artifact.sh  symlink to the shared pipeline; syncs
    │   │                     pkg/esp/wakenet/glue into main/ (artifact.conf)
    │   └── post-sync.sh      writes this demo's partition table
    └── main/
        ├── CMakeLists.txt  Sources + REQUIRES esp-sr + WHOLE_ARCHIVE + link options
        ├── idf_component.yml  espressif/esp-sr ^2.6.2 (engine + model packaging)
        ├── demo_glue.c     Demo-local: heap marks, test-source PCM injection
        ├── wakenet_pcm.c/h The "Hi, Lexin" utterance as int16 PCM (from the
        │                   esp-sr test app's samples, Espressif MIT license)
        └── wakenet_glue.c  Package glue, synced from
                            github.com/xgo-dev/llgo/pkg/esp/wakenet/glue (gitignored)
```

## The architecture, in one paragraph

WakeNet `detect` costs 3–6.4 ms per 32 ms audio chunk - thirty heavy
synchronous C calls per second. The llgo fiber scheduler is single-task
cooperative state, so none of those may run on a goroutine: the glue
(`pkg/esp/wakenet/glue/wakenet_glue.c`) owns the whole audio path on its own
FreeRTOS task (pinned to CPU1) - I2S RX or the test source → one chunk →
detect (timed) → event ring - and Go only polls events and counters
(`wakenet.Events()`, `wakenet.Stats()`). One model instance, no AEC/NS/VAD:
a pure-WakeNet instance is ~16 KB internal RAM + ~330 KB PSRAM (weights,
DET_MODE_95), the configuration the memory ledger says fits next to WiFi.

## Build and run

Prerequisites: an llgo compiler (`/tmp/llgo-idf` below, built with
`LLVM_CONFIG=... go build -o /tmp/llgo-idf ./cmd/llgo`), Docker image
`espressif/idf:latest`, esp-emu for the emulator run.

```sh
cd idfproj
./host/make-artifact.sh /tmp/llgo-idf     # Go → llgo_out.o, glue sync
docker run --rm -v "$PWD":/project -w /project \
    -v "$HOME/.espressif":/root/.espressif --user root \
    espressif/idf:latest bash -c "idf.py set-target esp32s31 && idf.py build && \
    idf.py merge-bin -o merged-binary.bin"
esp-emu --chip esp32s31 --firmware build/merged-binary.bin
```

The probe gates on `[go] wakenet PASS`: the model loaded, the detect task
processed ≥300 frames (~9.6 s of audio) next to the Go scheduler, and the
final line carries the frames/avg/max numbers. `DETECTED` lines mean the
emulator (or board) executed the model's compute well enough to trigger -
end-to-end proof; their absence with a PASS still proves the loop (see the
emu notes in the run log).

The P0 smoke build skips llgo entirely (first thing to run on a new
esp-sr/IDF combination, and the reference for the numbers the Go probe
prints):

```sh
cd csmoke
docker run --rm -v "$PWD":/project -w /project -v "$HOME/.espressif":/root/.espressif \
    --user root espressif/idf:latest bash -c "idf.py set-target esp32s31 && idf.py build"
```

## Flash-size note

The checked-in sdkconfig and partition table assume 16 MB flash (the esp-sr
S31 CI layout: factory 8000K + model 7900K). A 4 MB board needs both shrunk:
`CONFIG_ESPTOOLPY_FLASHSIZE_4MB=y` and a table like `nvs 0x6000 / phy 0x1000
/ factory 2.5 MB / model 512 KB` - wn9_hilexin is 284 KB, so the model
partition has to be ≥ that plus margin, and the app has to fit in what is
left. The numbers have not been validated on a 4 MB board; treat them as a
starting point, not a promise.
