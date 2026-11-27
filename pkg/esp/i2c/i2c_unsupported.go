// The "this build has no driver behind it" code the hosted stub returns
// (i2c_stub.go). The baremetal backend never reports it: it is compiled only
// for chips that have a register table.
//
// The value lies between the real esp_err ranges so it can never be mistaken
// for one the ESP-IDF driver would return.
package i2c

const errUnsupported = 0x7F00
