# Demo-local post-sync hook, sourced by _demo/common/make-artifact.sh right
# after the common scaffold and glue sync (the pipeline's working directory
# is the demo root and $MAIN points at the main component).
#
# Demo-specific build facts live here:
#
# 1. The partition table. The pipeline writes the shared 4 MB table into
#    every project; this firmware needs the esp-sr model partition on top -
#    the esp-sr S31 CI layout (16 MB flash: factory 8000K, model 7900K) with
#    the standard NVS/PHY sectors ahead of the app.
# 2. The wakenet glue's MCLK pin. The board microphone sits behind the
#    ES8311, which clocks itself from MCLK (GPIO 52 here); the upstream
#    template hardcodes I2S_GPIO_UNUSED. The patch points the template's
#    gpio_cfg at the demo glue's llgo_audiocheck_mclk global (demo_glue.c;
#    Go sets it before wakenet.Init in HW mode, -1 = upstream default).
# 3. The capture-energy hook. The 0-hit diagnosis needs the engine's input
#    energy beside the Go capture's: the patch inserts one call to the demo
#    glue's llgo_ac_chunk_rms (demo_glue.c) into the capture loop, right
#    before detect. EMU (the test source) never reaches it; the Go
#    heartbeat prints and clears it (HW heartbeats only).
# 4. The input-level fix. The board's record path sits 15 dB below the
#    esp-sr reference utterance (chunk rms ~270 vs 1529) and the codec's
#    0x16 digital gain does not deliver here. The patch inserts a saturated
#    x8 (+18 dB) multiply (llgo_ac_chunk_gain, demo_glue.c) at the END of
#    the capture loop's I2S branch - after the energy hook, before detect -
#    so the engine sees the raised level while the raw rms stays printed
#    and the EMU test source never passes through the gain.
# 5. The threshold readback. The threshold comparison experiment sets the
#    engine's detection threshold from Go (ModelConfig.Threshold, 0.50);
#    the patch adds a get_det_threshold readback right after the set, so
#    the log carries the value the engine actually holds (the model's own
#    default when the Go side set nothing - the EMU runs never set it).
#
# All patch points are asserted, so a drifted upstream template fails the
# build here instead of silently losing MCLK, the energy hook, the gain or
# the threshold readback.

cat > "$MAIN/../partitions.csv" <<'CSV'
# Name,   Type, SubType, Offset,  Size, Flags
nvs,      data, nvs,     0x9000,  0x6000,
phy_init, data, phy,     0xf000,  0x1000,
factory,  app,  factory, 0x10000, 8000K,
model,    data, spiffs,  ,        7900K,
CSV

GLUE="$MAIN/wakenet_glue.c"
grep -q '#include "driver/i2s_std.h"' "$GLUE" \
    || { echo "post-sync: wakenet_glue.c lost its i2s_std include; MCLK patch point moved" >&2; exit 1; }
grep -q '\.mclk = I2S_GPIO_UNUSED,' "$GLUE" \
    || { echo "post-sync: wakenet_glue.c gpio_cfg no longer sets '.mclk = I2S_GPIO_UNUSED'; MCLK patch point moved" >&2; exit 1; }
python3 - "$GLUE" <<'EOF'
import sys
path = sys.argv[1]
text = open(path).read()
inc = '#include "driver/i2s_std.h"'
old = '.mclk = I2S_GPIO_UNUSED,'
assert text.count(inc) == 1 and text.count(old) == 1
text = text.replace(
    inc,
    inc + '\n\n/* audiocheck demo hook (post-sync.sh): the ES8311 mic needs MCLK; the\n'
          ' * pin travels through the demo glue (llgo_audiocheck_mclk, demo_glue.c;\n'
          ' * -1 = the upstream unused-pin default). */\n'
          'extern int32_t llgo_audiocheck_mclk;',
    1)
text = text.replace(
    old,
    '.mclk = (llgo_audiocheck_mclk >= 0) ? (gpio_num_t)llgo_audiocheck_mclk\n'
    '                                                    : I2S_GPIO_UNUSED,',
    1)
open(path, 'w').write(text)
EOF
echo "post-sync: wakenet_glue.c MCLK hook patched in"

HOOK="/* detect, with the per-call timing the Go side prints. */"
grep -qF "$HOOK" "$GLUE" \
    || { echo "post-sync: wakenet_glue.c lost its detect comment; chunk-rms patch point moved" >&2; exit 1; }
python3 - "$GLUE" <<'EOF'
import sys
path = sys.argv[1]
text = open(path).read()
hook = '/* detect, with the per-call timing the Go side prints. */'
assert text.count(hook) == 1
text = text.replace(
    hook,
    'extern void llgo_ac_chunk_rms(const int16_t *pcm, int32_t n); /* demo glue\n'
    ' * (demo_glue.c), post-sync.sh */\n'
    'llgo_ac_chunk_rms(s_chunk, s_chunk_samples); /* demo hook: engine-input energy */\n\n'
    + hook,
    1)
open(path, 'w').write(text)
EOF
echo "post-sync: wakenet_glue.c chunk-rms hook patched in"

python3 - "$GLUE" <<'EOF'
import sys
path = sys.argv[1]
text = open(path).read()
gain = '            } else {\n                wn_test_fill'
assert text.count(gain) == 1, 'wakenet_glue.c lost the I2S-branch end; chunk-gain patch point moved'
text = text.replace(
    gain,
    '                extern void llgo_ac_chunk_gain(int16_t *pcm, int32_t n); /* demo\n'
    '                 * glue (demo_glue.c), post-sync.sh */\n'
    '                llgo_ac_chunk_gain(s_chunk, s_chunk_samples); /* demo hook: +18 dB\n'
    '                 * software gain, I2S capture only */\n'
    '            } else {\n'
    '                wn_test_fill',
    1)
open(path, 'w').write(text)
EOF
echo "post-sync: wakenet_glue.c chunk-gain hook patched in"

python3 - "$GLUE" <<'EOF'
import sys
path = sys.argv[1]
text = open(path).read()
thr = '    if (s_threshold_milli > 0) {\n        s_wn->set_det_threshold(s_md, (float)s_threshold_milli / 1000.0f, 1);\n    }'
assert text.count(thr) == 1, 'wakenet_glue.c lost the threshold set; readback patch point moved'
text = text.replace(
    thr,
    thr + '\n'
    '    /* threshold readback (demo hook, post-sync.sh): the log carries the\n'
    '     * value the engine actually holds - the set value, or the model\n'
    '     * default when the Go side set nothing (EMU never sets it). */\n'
    '    {\n'
    '        extern int printf(const char *fmt, ...); /* demo glue, post-sync.sh */\n'
    '        printf("[c] wn threshold = %.2f (set %.3f)\\n",\n'
    '               s_wn->get_det_threshold(s_md, 1),\n'
    '               s_threshold_milli > 0 ? (float)s_threshold_milli / 1000.0f : -1.0f);\n'
    '    }',
    1)
open(path, 'w').write(text)
EOF
echo "post-sync: wakenet_glue.c threshold readback patched in"
