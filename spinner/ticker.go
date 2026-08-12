package spinner

import (
	"sync"
	"time"
)

// Ticker is the clock a [Spinner] animates on. The animation loop selects on
// C() and calls Stop() exactly once, when the loop exits.
//
// It is an interface, and injectable through [Config.NewTicker], because a
// spinner that could only be driven by time.Ticker would force every test to
// sleep for real: the assertions would be about wall-clock luck rather than
// about the frames. Tests hand in [ManualTicker] and step the animation one
// frame at a time.
type Ticker interface {
	// C returns the channel ticks arrive on. A closed channel ends the
	// animation, which is how a fake clock can retire a spinner.
	C() <-chan time.Time
	// Stop releases the ticker's resources. It is called once, from the
	// animation goroutine, as that goroutine exits.
	Stop()
}

// Acker is an optional [Ticker] extension. When a ticker implements it the
// spinner calls Acked after it has finished painting the frame for a tick, so a
// synchronous fake clock can know the frame is already on Out before its own
// Tick call returns. Without this handshake a test would deliver a tick and then
// have to guess when the goroutine got round to writing it, which is exactly the
// flake this package is trying not to ship.
type Acker interface {
	Acked()
}

// realTicker is the production clock: a plain time.Ticker.
type realTicker struct{ t *time.Ticker }

func newRealTicker(d time.Duration) Ticker { return &realTicker{t: time.NewTicker(d)} }

func (r *realTicker) C() <-chan time.Time { return r.t.C }
func (r *realTicker) Stop()               { r.t.Stop() }

// ManualTicker is a [Ticker] whose ticks are delivered by hand, for
// deterministic tests. Pass its New method as [Config.NewTicker]:
//
//	mt := spinner.NewManualTicker()
//	s := spinner.New(spinner.Config{Out: &buf, NewTicker: mt.New, Mode: spinner.Animate})
//	s.Start()
//	mt.Tick() // returns once the next frame has been written to buf
//	s.Stop()
//
// Tick is synchronous: it blocks until the spinner has painted the frame, or
// until the spinner's goroutine has gone away, so a test never sleeps and never
// races the animation.
type ManualTicker struct {
	mu       sync.Mutex
	interval time.Duration
	done     chan struct{}

	ch  chan time.Time
	ack chan struct{}
}

// NewManualTicker returns a ManualTicker ready to be handed to a Spinner.
func NewManualTicker() *ManualTicker {
	return &ManualTicker{
		done: make(chan struct{}),
		ch:   make(chan time.Time),
		ack:  make(chan struct{}),
	}
}

// New records d as the requested interval and returns the ticker itself, so the
// method satisfies [Config.NewTicker].
func (m *ManualTicker) New(d time.Duration) Ticker {
	m.mu.Lock()
	m.interval = d
	m.mu.Unlock()
	return m
}

// Interval reports the interval the spinner asked for, which is how a test
// asserts on interval resolution without waiting for a single tick.
func (m *ManualTicker) Interval() time.Duration {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.interval
}

// C implements [Ticker].
func (m *ManualTicker) C() <-chan time.Time { return m.ch }

// Stop implements [Ticker]. It also unblocks any in-flight Tick: the spinner
// calls Stop as its goroutine exits, and a test blocked in Tick at that moment
// would otherwise hang forever.
func (m *ManualTicker) Stop() {
	m.mu.Lock()
	defer m.mu.Unlock()
	select {
	case <-m.done:
	default:
		close(m.done)
	}
}

// Reset makes a stopped ManualTicker usable again, for tests that restart a
// spinner after stopping it.
func (m *ManualTicker) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	select {
	case <-m.done:
		m.done = make(chan struct{})
	default:
	}
}

// Tick delivers one tick and waits for the frame to be painted. It reports
// whether the tick was consumed: false means the spinner is no longer
// animating, which is a normal outcome after Stop rather than an error.
func (m *ManualTicker) Tick() bool {
	m.mu.Lock()
	done, ch, ack := m.done, m.ch, m.ack
	m.mu.Unlock()

	select {
	case ch <- time.Now():
	case <-done:
		return false
	}
	select {
	case <-ack:
		return true
	case <-done:
		return false
	}
}

// Acked implements [Acker].
func (m *ManualTicker) Acked() {
	m.mu.Lock()
	done, ack := m.done, m.ack
	m.mu.Unlock()

	select {
	case ack <- struct{}{}:
	case <-done:
	}
}
