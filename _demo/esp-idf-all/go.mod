// Comprehensive verification firmware: one llgo firmware exercising the full
// ESP32-S31 capability set at once - soft AP + HTTP server, A2DP sink audio,
// a BLE HID-keyboard central host and the ES8311 codec self-test - so one
// flash answers every capability question in a single board session.
module llgoidfall

go 1.27.0

require github.com/xgo-dev/llgo v0.0.0

replace github.com/xgo-dev/llgo => ../..
