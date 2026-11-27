// The "this build cannot drive the pin" code the hosted stub returns
// (gpio_stub.go). The baremetal backend never reports it: it is compiled only
// for chips that have a register table, and its own refusal is
// ESP_ERR_INVALID_ARG for a pin the SoC does not have.
//
// The value lies between the real esp_err ranges so it can never be mistaken
// for one the ESP-IDF driver would return.
package gpio

const errUnsupported = 0x7F00
