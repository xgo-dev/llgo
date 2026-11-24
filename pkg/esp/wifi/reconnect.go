package wifi

// Auto-reconnect: an optional state machine that reconnects the station
// after an unexpected disconnection - the router rebooted, the signal
// dropped, the AP refused a re-auth. Without it, a disconnection is the
// application's problem: the mailbox carries EvStaDisconnected and nothing
// ever calls esp_wifi_connect again.
//
// # Shape
//
//	SetAutoReconnect(true)          // machine reacts from now on
//	...
//	wifi.Disconnect()               // intentional: no retry will follow
//	...
//	// an unexpected EvStaDisconnected now schedules esp_wifi_connect
//	// attempts at 1 s, 2 s, 4 s ... capped at 30 s; the first EvGotIP
//	// resets the delay and ends the episode.
//
// A Disconnect the program itself called never triggers a retry: it arms an
// intentional-disconnect flag that exactly one EvStaDisconnected consumes.
// Everything else that disconnects the station is unexpected and starts the
// backoff-and-retry sequence.
//
// # Events
//
// The machine adds no event kinds. The stack's own story - EvStaDisconnected
// (reason code in Aux), EvStaConnected, EvGotIP - already covers an episode
// end to end, the regular fan-out delivers it to every subscriber, and
// synthesizing a "reconnecting" event the driver never fires would invent
// state the mailbox cannot verify. ReconnectState() reports the machine's
// phase (idle / backoff / connecting) for programs that want to observe the
// retry window itself.
//
// # Scheduling
//
// The machine is one goroutine and a subscriber like any other: it never
// owns the C mailbox and never steals from other subscriptions. Its waits
// follow the package's bounded-quantum discipline - park the FreeRTOS task
// one vTaskDelay quantum (100 ms, which also feeds the idle task and the
// watchdog), hand the scheduler away with runtime.Gosched so the poller can
// drain the C ring, then check its subscription. No C-level wait is held
// across a scheduling point, and nothing busy-spins.
//
// # Cooperation
//
// All machine state changes happen between scheduling points - the llgo
// scheduler is cooperative (goroutines switch only at blocking calls), so
// plain variables are race-free, the same discipline Subscribe and Close
// rely on. While the machine is enabled it owns esp_wifi_connect: a manual
// Connect during a backoff window races the machine's next attempt at the
// driver level (IDF serializes the two with esp_err codes; the loser just
// fails). Scan works during a disconnection (scanning needs no
// association) but a scan racing a connect attempt reports
// ESP_ERR_WIFI_STATE - an error value, never a panic or a hang.
//
// # Verification status
//
// The emu acceptance run (see _demo/esp-idf-wifi) drives every transition
// itself: an intentional disconnect, a manual reconnect, and an unexpected
// disconnection produced by associating to an SSID nobody serves - the
// same EvStaDisconnected a lost router delivers, from the machine's point
// of view indistinguishable from it. What the emulator cannot inject is the
// AP vanishing while the station sits associated (real router power loss);
// that hardware shape remains a pending verification item.

const (
	// reconnectStartMs is the first retry delay after an unexpected
	// disconnection.
	reconnectStartMs = 1000

	// reconnectMaxMs caps the exponential backoff. Once there, attempts
	// keep running at this cadence until a GOT_IP resets the episode.
	reconnectMaxMs = 30000
)

// ReconnectPhase is the auto-reconnect state machine's current phase, as
// reported by ReconnectState.
type ReconnectPhase int32

const (
	// ReconnIdle: the machine is disabled, or enabled with the station
	// connected (or never associated) and nothing pending.
	ReconnIdle ReconnectPhase = 0

	// ReconnBackoff: an unexpected disconnection happened; the machine is
	// waiting out the current retry delay before its next esp_wifi_connect.
	ReconnBackoff ReconnectPhase = 1

	// ReconnConnecting: an esp_wifi_connect the machine issued is in
	// flight; the next EvStaConnected/EvGotIP or EvStaDisconnected decides
	// the episode.
	ReconnConnecting ReconnectPhase = 2
)

// String names the phase ("idle", "backoff", "connecting") for probes and
// logs.
func (p ReconnectPhase) String() string {
	switch p {
	case ReconnBackoff:
		return "backoff"
	case ReconnConnecting:
		return "connecting"
	default:
		return "idle"
	}
}

// Machine state. Every write happens without a scheduling point between a
// read and its dependent write - SetAutoReconnect and Disconnect run on the
// caller's goroutine, the rest on the machine goroutine between quanta - so
// the cooperative scheduler makes the pairs atomic.
var (
	autoReconnect    bool
	reconnectPhase   ReconnectPhase = ReconnIdle
	userDisconnect   bool           // armed by Disconnect, consumed by one DISCONNECTED
	reconnectDelayMs int32          = reconnectStartMs
	reconnectRunning bool           // the machine goroutine exists
)

// SetAutoReconnect enables or disables the automatic reconnect state
// machine. Enabling subscribes the machine (once, synchronously - the
// mailbox is already being watched when the call returns), resets the
// backoff and lets it run; disabling cancels any pending or in-flight
// attempt and parks the machine until enabled again. It is safe to call
// before Setup: the machine simply idles until real events arrive. The
// default is disabled - programs that never call this keep their exact
// current behavior, retries remaining the application's own business.
func SetAutoReconnect(enable bool) {
	if enable == autoReconnect {
		return
	}
	autoReconnect = enable
	if !enable {
		// Mid-episode disable: drop everything the machine was holding.
		reconnectPhase = ReconnIdle
		userDisconnect = false
		return
	}
	reconnectDelayMs = reconnectStartMs
	if reconnectRunning {
		return
	}
	reconnectRunning = true
	// Operands of a go statement evaluate on the calling goroutine, so the
	// subscription exists before SetAutoReconnect returns.
	go runReconnect(Subscribe(EvStaDisconnected, EvGotIP))
}

// ReconnectState reports the machine's phase: ReconnIdle (disabled or
// settled), ReconnBackoff (waiting out the retry delay) or ReconnConnecting
// (the machine's esp_wifi_connect is in flight).
func ReconnectState() ReconnectPhase { return reconnectPhase }

// runReconnect is the machine loop. sub is the subscription SetAutoReconnect
// took for it; the loop parks one quantum (vTaskDelay - the idle task and
// watchdog stay fed), hands the scheduler away (the poller drains the C
// ring into the fan-out), then consumes what arrived and advances the
// countdown. One iteration is one pollQuantumMs step, so the backoff
// countdown runs in 100 ms quanta exactly like every other wait in this
// package.
func runReconnect(sub *Subscription) {
	// quantaLeft counts the retry delay down in pollQuantumMs steps;
	// 0 means nothing is scheduled.
	quantaLeft := int32(0)
	for {
		vTaskDelay(pollQuantumMs / tickMs)
		gosched()

		// Consume everything the fan-out has delivered.
	drain:
		for {
			var ev Event
			select {
			case ev = <-sub.ch:
			default:
				break drain
			}
			switch ev.Kind {
			case EvGotIP:
				// Episode over - connected again, or still
				// connected: backoff reset, nothing pending.
				quantaLeft = 0
				reconnectPhase = ReconnIdle
				reconnectDelayMs = reconnectStartMs
				userDisconnect = false
			case EvStaDisconnected:
				if !autoReconnect {
					continue // disabled: the machine is inert
				}
				if userDisconnect {
					// This program asked for it (Disconnect):
					// consume the flag, schedule nothing.
					userDisconnect = false
					quantaLeft = 0
					reconnectPhase = ReconnIdle
					continue
				}
				// Unexpected: schedule one attempt after the
				// current delay, then grow the delay for the
				// attempt after that.
				quantaLeft = reconnectDelayMs / pollQuantumMs
				if quantaLeft < 1 {
					quantaLeft = 1
				}
				reconnectDelayMs *= 2
				if reconnectDelayMs > reconnectMaxMs {
					reconnectDelayMs = reconnectMaxMs
				}
				reconnectPhase = ReconnBackoff
			}
		}

		if !autoReconnect {
			// Disabled between quanta: nothing pending, idle.
			quantaLeft = 0
			reconnectPhase = ReconnIdle
			continue
		}
		if quantaLeft > 0 {
			quantaLeft--
			if quantaLeft == 0 {
				// The attempt. Connect reads whatever station
				// config the driver holds right now, so credentials
				// updated mid-episode apply from the next attempt
				// on. A failed attempt comes back as the next
				// DISCONNECTED, which re-arms the grown delay; a
				// call error (e.g. ESP_ERR_WIFI_STATE while a scan
				// holds the driver) fires no event, so re-arm here
				// at the same delay instead of idling forever.
				reconnectPhase = ReconnConnecting
				if err := Connect(); err != nil {
					quantaLeft = reconnectDelayMs / pollQuantumMs
					if quantaLeft < 1 {
						quantaLeft = 1
					}
					reconnectPhase = ReconnBackoff
				}
			}
		}
	}
}
