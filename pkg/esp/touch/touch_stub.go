//go:build !llgo.idf

// Host build (go build / go vet / gopls): the C glue only exists in the
// ESP-IDF link environment, so every binding reports errUnsupported. It lies
// between the real esp_err ranges so it can never be mistaken for one.
// Nothing here is meant to run on a host.
package touch

const errUnsupported = 0x7F00

func init_(chargeTimes, divNum, powerOnWaitUs, measIntervalUs, maxMeasTimeUs, outMode int32) int32 {
	_, _, _, _, _, _ = chargeTimes, divNum, powerOnWaitUs, measIntervalUs, maxMeasTimeUs, outMode
	return errUnsupported
}
func configFilter(bmMode, jitterStep, denoiseLvl, smooth, hysteresis, debounce int32) int32 {
	_, _, _, _, _, _ = bmMode, jitterStep, denoiseLvl, smooth, hysteresis, debounce
	return errUnsupported
}
func configChannel(chanID int32, activeThresh int32) int32 {
	_, _ = chanID, activeThresh
	return errUnsupported
}
func start() int32 { return errUnsupported }
func stop() int32  { return errUnsupported }
func readChan_(chanID int32, dataType int32, out *uint32) int32 {
	_, _, _ = chanID, dataType, out
	return errUnsupported
}
func calibrate(chanID int32) int32 { _ = chanID; return errUnsupported }
func setThreshold(chanID int32, thresh int32) int32 {
	_, _ = chanID, thresh
	return errUnsupported
}
func pollEvent(outChan *int32, outActive *int32) bool {
	_, _ = outChan, outActive
	return false
}
func deinit() int32 { return errUnsupported }
