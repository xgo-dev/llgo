# Demo-local post-sync hook, sourced by _demo/common/make-artifact.sh right
# after the glue sync (the pipeline's working directory is the demo root and
# $MAIN points at the main component).
#
# The A2DP glue needs no local patching anymore: the codec's MCLK pin (GPIO52,
# the ES8311's system clock) is the A2DPConfig.MCLKPin field now, and the
# synced a2dp_glue.c drives it from the config.

# Optional stage tracing inside the classic-stack bring-up: LLGO_BT_TRACE=1
# prints a line around each blocking C call, to locate a hang.
if [[ "${LLGO_BT_TRACE:-0}" == "1" ]]; then
python3 - "$MAIN/bt_glue.c" <<'PYEOF'
import sys
path = sys.argv[1]
src = open(path).read()
assert '#include <stdio.h>' not in src
src = src.replace('#include "esp_bt.h"', '#include <stdio.h>\n#include "esp_bt.h"', 1)
marks = [
    ('    if (s_uninit_sem == NULL) {',
     '    printf("[c] bt trace: nvs ok\\n");\n    if (s_uninit_sem == NULL) {'),
    ('    err = esp_bt_controller_init(&cfg);',
     '    printf("[c] bt trace: ctrl init...\\n");\n    err = esp_bt_controller_init(&cfg);\n    printf("[c] bt trace: ctrl init rc=%d\\n", (int)err);'),
    ('    err = esp_bt_controller_enable(ESP_BT_MODE_BTDM);',
     '    printf("[c] bt trace: ctrl enable BTDM...\\n");\n    err = esp_bt_controller_enable(ESP_BT_MODE_BTDM);\n    printf("[c] bt trace: ctrl enable rc=%d\\n", (int)err);'),
    ('    err = esp_bluedroid_init();',
     '    printf("[c] bt trace: bluedroid init...\\n");\n    err = esp_bluedroid_init();\n    printf("[c] bt trace: bluedroid init rc=%d\\n", (int)err);'),
    ('    err = esp_bluedroid_enable();',
     '    printf("[c] bt trace: bluedroid enable...\\n");\n    err = esp_bluedroid_enable();\n    printf("[c] bt trace: bluedroid enable rc=%d\\n", (int)err);'),
    ('    err = esp_bt_gap_register_callback(gap_cb);',
     '    printf("[c] bt trace: gap register...\\n");\n    err = esp_bt_gap_register_callback(gap_cb);\n    printf("[c] bt trace: gap register rc=%d ready=1\\n", (int)err);'),
]
for old, new in marks:
    assert src.count(old) == 1, "bt trace patch point not found: " + old
    src = src.replace(old, new, 1)
open(path, 'w').write(src)
print("bt_glue.c: LLGO_BT_TRACE stage prints applied")
PYEOF
fi
