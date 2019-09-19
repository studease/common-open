package timer

import (
	"sync/atomic"
	"time"

	"github.com/studease/common/events"
	TimerEvent "github.com/studease/common/events/timerevent"
	"github.com/studease/common/log"
)

// Timer states
const (
	STATE_INITIALIZED int32 = 0x00
	STATE_RUNNING     int32 = 0x01
)

// Timer is the interface to timers, which let you run code on a specified time sequence
type Timer struct {
	events.EventDispatcher

	logger       log.ILogger
	ticker       *time.Ticker
	delay        time.Duration
	repeatCount  int32
	currentCount int32
	state        int32
}

// Init this class. If the repeat count is set to 0, the timer continues indefinitely
func (me *Timer) Init(delay time.Duration, repeatCount int32, logger log.ILogger) *Timer {
	me.EventDispatcher.Init(logger)
	me.delay = delay
	me.repeatCount = repeatCount
	me.logger = logger
	me.currentCount = 0
	me.state = STATE_INITIALIZED
	return me
}

// Start the timer, if it is not already running
func (me *Timer) Start() {
	if atomic.CompareAndSwapInt32(&me.state, STATE_INITIALIZED, STATE_RUNNING) {
		me.ticker = time.NewTicker(me.delay)
		go me.wait()
	}
}

func (me *Timer) wait() {
	for {
		select {
		case <-me.ticker.C:
			if atomic.LoadInt32(&me.state) != STATE_RUNNING {
				return
			}

			n := atomic.AddInt32(&me.currentCount, 1)
			me.DispatchEvent(TimerEvent.New(TimerEvent.TIMER, me))

			if me.repeatCount > 0 && n == me.repeatCount {
				me.DispatchEvent(TimerEvent.New(TimerEvent.COMPLETE, me))
				me.Stop()
			}
		}
	}
}

// Stop the timer
func (me *Timer) Stop() {
	if atomic.CompareAndSwapInt32(&me.state, STATE_RUNNING, STATE_INITIALIZED) {
		me.ticker.Stop()
	}
}

// Reset stops the timer, if it is running, and sets the currentCount property back to 0, like the reset button of a stopwatch
func (me *Timer) Reset() {
	me.Stop()
	atomic.StoreInt32(&me.currentCount, 0)
}

// Running returns the timer's current state; true if the timer is running, otherwise false
func (me *Timer) Running() bool {
	return atomic.LoadInt32(&me.state) == STATE_RUNNING
}

// New creates a Timer with the given arguments
func New(delay time.Duration, repeatCount int32, logger log.ILogger) *Timer {
	return new(Timer).Init(delay, repeatCount, logger)
}
