package time_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"testing"
	"time"
)

func TestTimeDurations(t *testing.T) {
	d, err := time.ParseDuration("1.5s")
	if err != nil {
		t.Fatalf("ParseDuration failed: %v", err)
	}
	if d != 1500*time.Millisecond {
		t.Fatalf("unexpected duration: %v", d)
	}

	mixed := d + 250*time.Microsecond + 10*time.Nanosecond
	if mixed.Abs() <= 0 {
		t.Fatalf("Abs should be positive, got %v", mixed.Abs())
	}
	if mixed.Hours() <= 0 {
		t.Fatalf("Hours should be positive")
	}
	if mixed.Minutes() <= 0 {
		t.Fatalf("Minutes should be positive")
	}
	if mixed.Seconds() <= 0 {
		t.Fatalf("Seconds should be positive")
	}
	if mixed.Milliseconds() <= 0 {
		t.Fatalf("Milliseconds should be positive")
	}
	if mixed.Microseconds() <= 0 {
		t.Fatalf("Microseconds should be positive")
	}
	if mixed.Nanoseconds() <= 0 {
		t.Fatalf("Nanoseconds should be positive")
	}
	if mixed.String() == "" {
		t.Fatalf("Duration String should not be empty")
	}
	if mixed.Truncate(time.Second) != time.Second {
		t.Fatalf("Truncate expected 1s, got %v", mixed.Truncate(time.Second))
	}
	if mixed.Round(time.Second) != 2*time.Second {
		t.Fatalf("Round expected 2s, got %v", mixed.Round(time.Second))
	}

	before := time.Now().Add(-25 * time.Millisecond)
	if time.Since(before) <= 0 {
		t.Fatalf("Since should be positive")
	}
	future := time.Now().Add(25 * time.Millisecond)
	if time.Until(future) <= 0 {
		t.Fatalf("Until should be positive")
	}
}

func TestTimeCreationAndFormatting(t *testing.T) {
	base := time.Date(2024, time.November, 11, 9, 8, 7, 654321000, time.UTC)
	if base.Year() != 2024 || base.Month() != time.November || base.Day() != 11 {
		t.Fatalf("unexpected date components: %v", base)
	}
	if base.Hour() != 9 || base.Minute() != 8 || base.Second() != 7 || base.Nanosecond() != 654321000 {
		t.Fatalf("unexpected clock components: %v", base)
	}
	y, m, d := base.Date()
	if y != 2024 || m != time.November || d != 11 {
		t.Fatalf("Date mismatch: %d %v %d", y, m, d)
	}
	h, min, s := base.Clock()
	if h != 9 || min != 8 || s != 7 {
		t.Fatalf("Clock mismatch: %d:%d:%d", h, min, s)
	}
	year, week := base.ISOWeek()
	if year < 2024 || week == 0 {
		t.Fatalf("unexpected ISO week: %d-%d", year, week)
	}
	if base.YearDay() == 0 {
		t.Fatalf("YearDay should be positive")
	}
	if base.Weekday().String() == "" {
		t.Fatalf("Weekday string empty")
	}
	if time.Saturday.String() == "" || time.December.String() == "" {
		t.Fatalf("enum String should not be empty")
	}

	if base.IsZero() {
		t.Fatalf("base should not be zero")
	}
	if !base.Equal(base) {
		t.Fatalf("Equal should be true for same instant")
	}

	local := base.Local()
	if local.Location() == nil {
		t.Fatalf("Local should preserve location")
	}
	if time.Local == nil {
		t.Fatalf("time.Local should not be nil")
	}

	plus := base.Add(2*time.Hour + 30*time.Minute)
	if delta := plus.Sub(base); delta != 150*time.Minute {
		t.Fatalf("unexpected delta: %v", delta)
	}
	plus = plus.AddDate(0, 1, 1)
	if !plus.After(base) || !base.Before(plus) {
		t.Fatalf("After/Before relationship violated")
	}
	if comp := base.Compare(plus); comp >= 0 {
		t.Fatalf("Compare expected negative, got %d", comp)
	}

	utc := base.UTC()
	if utc.Location() != time.UTC {
		t.Fatalf("UTC location expected")
	}
	fixed := time.FixedZone("Fixed", 3600)
	inFixed := base.In(fixed)
	if inFixed.Location().String() != "Fixed" {
		t.Fatalf("unexpected fixed zone: %v", inFixed.Location())
	}
	if inFixed.IsDST() {
		t.Fatalf("FixedZone should not observe DST")
	}
	if loc := time.Local; loc == nil {
		t.Fatalf("time.Local should be non-nil")
	}

	if formatted := base.Format(time.RFC3339Nano); formatted == "" {
		t.Fatalf("Format expected non-empty")
	}
	appended := base.AppendFormat([]byte("prefix:"), time.Kitchen)
	if !bytes.HasPrefix(appended, []byte("prefix:")) {
		t.Fatalf("AppendFormat prefix missing: %s", appended)
	}
	textAppended, err := base.AppendText(nil)
	if err != nil || len(textAppended) == 0 {
		t.Fatalf("AppendText failed: %v", err)
	}
	binAppended, err := base.AppendBinary(nil)
	if err != nil || len(binAppended) == 0 {
		t.Fatalf("AppendBinary failed: %v", err)
	}

	marshaledBinary, err := base.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary failed: %v", err)
	}
	var unmarshaled time.Time
	if err := unmarshaled.UnmarshalBinary(marshaledBinary); err != nil {
		t.Fatalf("UnmarshalBinary failed: %v", err)
	}

	marshaledJSON, err := base.MarshalJSON()
	if err != nil {
		t.Fatalf("MarshalJSON failed: %v", err)
	}
	if err := unmarshaled.UnmarshalJSON(marshaledJSON); err != nil {
		t.Fatalf("UnmarshalJSON failed: %v", err)
	}

	marshaledText, err := base.MarshalText()
	if err != nil {
		t.Fatalf("MarshalText failed: %v", err)
	}
	if err := unmarshaled.UnmarshalText(marshaledText); err != nil {
		t.Fatalf("UnmarshalText failed: %v", err)
	}

	gobData, err := base.GobEncode()
	if err != nil {
		t.Fatalf("GobEncode failed: %v", err)
	}
	if err := (&unmarshaled).GobDecode(gobData); err != nil {
		t.Fatalf("GobDecode failed: %v", err)
	}

	if base.GoString() == "" || base.String() == "" {
		t.Fatalf("GoString/String should not be empty")
	}

	if base.Round(time.Minute).Sub(base.Truncate(time.Minute)) > time.Minute {
		t.Fatalf("Round/Truncate difference too large")
	}

	layouts := []string{
		time.ANSIC,
		time.UnixDate,
		time.RubyDate,
		time.RFC822,
		time.RFC822Z,
		time.RFC850,
		time.RFC1123,
		time.RFC1123Z,
		time.RFC3339,
		time.RFC3339Nano,
		time.Kitchen,
		time.Layout,
		time.Stamp,
		time.StampMilli,
		time.StampMicro,
		time.StampNano,
		time.DateTime,
		time.DateOnly,
		time.TimeOnly,
	}
	for _, layout := range layouts {
		str := base.Format(layout)
		if str == "" {
			t.Fatalf("layout %q produced empty string", layout)
		}
		if _, err := time.Parse(layout, str); err != nil {
			t.Fatalf("Parse failed for layout %q: %v", layout, err)
		}
	}

	unix := base.Unix()
	if unix != time.Unix(unix, 0).Unix() {
		t.Fatalf("Unix round trip failed")
	}
	if base.UnixMilli() != time.UnixMilli(base.UnixMilli()).UnixMilli() {
		t.Fatalf("UnixMilli round trip failed")
	}
	if base.UnixMicro() != time.UnixMicro(base.UnixMicro()).UnixMicro() {
		t.Fatalf("UnixMicro round trip failed")
	}
	if base.UnixNano() != time.Unix(0, base.UnixNano()).UnixNano() {
		t.Fatalf("UnixNano round trip failed")
	}

	start, end := inFixed.ZoneBounds()
	if !(start.Before(end) || start.Equal(end)) {
		t.Fatal("ZoneBounds returned invalid order")
	}
	if name, offset := inFixed.Zone(); name == "" || offset != 3600 {
		t.Fatalf("Zone information incorrect: %s %d", name, offset)
	}
}

func TestTimeParsingAndLocations(t *testing.T) {
	locUTC, err := time.LoadLocation("UTC")
	if err != nil {
		t.Fatalf("LoadLocation UTC failed: %v", err)
	}
	if locUTC.String() != "UTC" {
		t.Fatalf("expected UTC string, got %s", locUTC)
	}
	if _, err := time.LoadLocationFromTZData("Invalid", []byte("TZif")); err == nil {
		t.Fatalf("expected error for invalid tz data")
	}

	parsed, err := time.Parse(time.RFC3339, "2024-11-11T09:08:00Z")
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	parsedInLoc, err := time.ParseInLocation(time.RFC822, "11 Nov 24 09:08 UTC", locUTC)
	if err != nil {
		t.Fatalf("ParseInLocation failed: %v", err)
	}
	if !parsed.Equal(parsedInLoc) {
		t.Fatalf("expected parsed times to equal")
	}

	if _, err := time.Parse("2006", "invalid"); err == nil {
		t.Fatalf("expected parse error")
	} else {
		var parseErr *time.ParseError
		if !errors.As(err, &parseErr) {
			t.Fatalf("expected ParseError, got %T", err)
		}
		if parseErr.Error() == "" {
			t.Fatalf("ParseError error string empty")
		}
	}
}

func TestTimeTimersAndTickers(t *testing.T) {
	timer := time.NewTimer(50 * time.Millisecond)
	if !timer.Stop() {
		t.Fatalf("expected Stop to return true before expiry")
	}
	if timer.Reset(10 * time.Millisecond) {
		t.Fatalf("Reset should report false on stopped timer")
	}
	select {
	case <-timer.C:
	case <-time.After(100 * time.Millisecond):
		t.Fatalf("timer did not fire after reset")
	}
	timer.Stop()

	ticker := time.NewTicker(5 * time.Millisecond)
	defer ticker.Stop()
	select {
	case <-ticker.C:
	case <-time.After(50 * time.Millisecond):
		t.Fatalf("ticker did not tick")
	}
	ticker.Reset(5 * time.Millisecond)
	select {
	case <-ticker.C:
	case <-time.After(50 * time.Millisecond):
		t.Fatalf("ticker did not tick after reset")
	}

	tickCh := time.Tick(5 * time.Millisecond)
	select {
	case <-tickCh:
	case <-time.After(50 * time.Millisecond):
		t.Fatalf("tick channel did not deliver")
	}

	afterCh := time.After(5 * time.Millisecond)
	select {
	case <-afterCh:
	case <-time.After(50 * time.Millisecond):
		t.Fatalf("After did not deliver")
	}

	done := make(chan struct{})
	timerFunc := time.AfterFunc(5*time.Millisecond, func() { close(done) })
	select {
	case <-done:
	// A loaded host or virtual machine can resume the timer loop after both
	// deadlines have elapsed. Keep enough separation for the AfterFunc
	// goroutine to start while still detecting a lost callback promptly.
	case <-time.After(time.Second):
		t.Fatalf("AfterFunc did not execute")
	}
	timerFunc.Stop()

	time.Sleep(5 * time.Millisecond)
	if since := time.Since(time.Now().Add(-2 * time.Millisecond)); since <= 0 {
		t.Fatalf("Since expected positive")
	}
}

func TestTimerResetWakesEarlierDeadline(t *testing.T) {
	head := time.NewTimer(2 * time.Second)
	defer head.Stop()
	target := time.NewTimer(time.Hour)
	defer target.Stop()
	// Let the timer scheduler observe the long-lived head before resetting a
	// different timer ahead of it. This makes the test exercise the wakeup from
	// an existing timed wait instead of racing scheduler initialization.
	time.Sleep(10 * time.Millisecond)

	start := time.Now()
	if !target.Reset(20 * time.Millisecond) {
		t.Fatal("Reset reported an active timer as stopped")
	}
	select {
	case <-target.C:
		if elapsed := time.Since(start); elapsed >= time.Second {
			t.Fatalf("earlier reset was not scheduled promptly: %v", elapsed)
		}
	case <-head.C:
		t.Fatal("earlier reset did not wake the timer scheduler")
	}
}

func TestTimerStartWakesAfterStoppingHead(t *testing.T) {
	head := time.NewTimer(2 * time.Second)
	defer head.Stop()
	// Give the scheduler time to enter the long wait and then remove its timer.
	time.Sleep(10 * time.Millisecond)
	if !head.Stop() {
		t.Fatal("the long timer had already fired")
	}

	start := time.Now()
	next := time.NewTimer(20 * time.Millisecond)
	defer next.Stop()
	select {
	case <-next.C:
		if elapsed := time.Since(start); elapsed >= time.Second {
			t.Fatalf("replacement timer did not wake the scheduler promptly: %v", elapsed)
		}
	case <-time.After(time.Second):
		t.Fatal("replacement timer did not fire")
	}
}

func TestTimeJSONInterop(t *testing.T) {
	payload := struct {
		When time.Time     `json:"when"`
		Span time.Duration `json:"span"`
	}{
		When: time.Date(2023, time.June, 1, 12, 0, 0, 0, time.UTC),
		Span: 42 * time.Second,
	}
	data, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("json.Marshal failed: %v", err)
	}
	var decoded struct {
		When time.Time     `json:"when"`
		Span time.Duration `json:"span"`
	}
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("json.Unmarshal failed: %v", err)
	}
	if !decoded.When.Equal(payload.When) || decoded.Span != payload.Span {
		t.Fatalf("JSON round trip mismatch: %+v vs %+v", decoded, payload)
	}
}

func TestTimeGlobalFunctions(t *testing.T) {
	now := time.Now()
	if now.IsZero() {
		t.Fatalf("Now returned zero")
	}
	unix := time.Unix(now.Unix(), int64(now.Nanosecond()))
	if !unix.Round(time.Nanosecond).UTC().Equal(time.Unix(now.Unix(), int64(now.Nanosecond())).UTC()) {
		t.Fatalf("Unix reconstruction mismatch")
	}
	unixMilli := time.UnixMilli(now.UnixMilli())
	if unixMilli.UnixMilli() != now.UnixMilli() {
		t.Fatalf("UnixMilli round trip mismatch")
	}
	unixMicro := time.UnixMicro(now.UnixMicro())
	if unixMicro.UnixMicro() != now.UnixMicro() {
		t.Fatalf("UnixMicro round trip mismatch")
	}

	future := now.Add(10 * time.Millisecond)
	if time.Until(future) <= 0 {
		t.Fatalf("Until should be positive")
	}

	if time.Nanosecond == 0 || time.Second == 0 || time.Hour == 0 {
		t.Fatalf("time unit constants should be non-zero")
	}
}
