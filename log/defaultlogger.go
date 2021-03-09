package log

import (
	"fmt"
	"io"
	"log"
	"os"
	"sync"

	"github.com/studease/common/log/internal/level"
)

const (
	// DEFAULT_DEPTH = this frame + wrapper func + caller
	DEFAULT_DEPTH = 3
)

var (
	std = NewDefaultLogger(os.Stdout, "TONY", level.TRACE, DEFAULT_DEPTH+1)
)

// DefaultLogger encapsulates functionality for providing logging at user-defined levels.
type DefaultLogger struct {
	sync.RWMutex

	scope     string
	level     level.Level
	calldepth int
	trace     *log.Logger
	debug     *log.Logger
	info      *log.Logger
	warn      *log.Logger
	error     *log.Logger
}

// Init this class.
func (me *DefaultLogger) Init(scope string, n level.Level, calldepth int) *DefaultLogger {
	me.scope = scope
	me.level = n
	me.calldepth = calldepth
	return me
}

// WithTrace is a chainable configuration function which sets the trace-level logger.
func (me *DefaultLogger) WithTrace(logger *log.Logger) *DefaultLogger {
	me.level |= level.TRACE
	me.trace = logger
	return me
}

// WithDebug is a chainable configuration function which sets the debug-level logger.
func (me *DefaultLogger) WithDebug(logger *log.Logger) *DefaultLogger {
	me.debug = logger
	return me
}

// WithInfo is a chainable configuration function which sets the info-level logger.
func (me *DefaultLogger) WithInfo(logger *log.Logger) *DefaultLogger {
	me.info = logger
	return me
}

// WithWarn is a chainable configuration function which sets the warn-level logger.
func (me *DefaultLogger) WithWarn(logger *log.Logger) *DefaultLogger {
	me.warn = logger
	return me
}

// WithError is a chainable configuration function which sets the error-level logger.
func (me *DefaultLogger) WithError(logger *log.Logger) *DefaultLogger {
	me.error = logger
	return me
}

func (me *DefaultLogger) log(logger *log.Logger, n level.Level, s string) {
	if me.level.Get() <= n {
		me.Lock()
		defer me.Unlock()

		err := logger.Output(me.calldepth, s)
		if err != nil {
			Warnf("Failed to log: %s", err)
			return
		}

		if (me.level.Get() & level.TRACE) != 0 {
			me.trace.SetPrefix(getPrefix(me.scope, n))
			me.trace.Output(me.calldepth, s)
		}
	}
}

func (me *DefaultLogger) logf(logger *log.Logger, n level.Level, format string, args ...interface{}) {
	if me.level.Get() <= n {
		s := fmt.Sprintf(format, args...)

		me.Lock()
		defer me.Unlock()

		err := logger.Output(me.calldepth, s)
		if err != nil {
			Warnf("Failed to log: %s", err)
			return
		}

		if (me.level.Get() & level.TRACE) != 0 {
			me.trace.SetPrefix(getPrefix(me.scope, n))
			me.trace.Output(me.calldepth, s)
		}
	}
}

// Trace emits the preformatted message if the logger is at or below trace-level.
func (me *DefaultLogger) Trace(s string) {
	if (me.level.Get() & level.TRACE) != 0 {
		me.log(me.trace, level.TRACE, s)
	}
}

// Tracef formats and emits a message if the logger is at or below trace-level.
func (me *DefaultLogger) Tracef(format string, args ...interface{}) {
	if (me.level.Get() & level.TRACE) != 0 {
		me.logf(me.trace, level.TRACE, format, args...)
	}
}

// Debug emits the preformatted message if the logger is at or below debug-level.
func (me *DefaultLogger) Debug(n uint32, s string) {
	if n < 8 {
		me.log(me.debug, level.DEBUG<<n, s)
	}
}

// Debugf formats and emits a message if the logger is at or below debug-level.
func (me *DefaultLogger) Debugf(n uint32, format string, args ...interface{}) {
	if n < 8 {
		me.logf(me.debug, level.DEBUG<<n, format, args...)
	}
}

// Info emits the preformatted message if the logger is at or below info-level.
func (me *DefaultLogger) Info(s string) {
	me.log(me.info, level.INFO, s)
}

// Infof formats and emits a message if the logger is at or below info-level.
func (me *DefaultLogger) Infof(format string, args ...interface{}) {
	me.logf(me.info, level.INFO, format, args...)
}

// Warn emits the preformatted message if the logger is at or below warn-level.
func (me *DefaultLogger) Warn(s string) {
	me.log(me.warn, level.WARN, s)
}

// Warnf formats and emits a message if the logger is at or below warn-level.
func (me *DefaultLogger) Warnf(format string, args ...interface{}) {
	me.logf(me.warn, level.WARN, format, args...)
}

// Error emits the preformatted message if the logger is at or below error-level.
func (me *DefaultLogger) Error(s string) {
	me.log(me.error, level.ERROR, s)
}

// Errorf formats and emits a message if the logger is at or below error-level.
func (me *DefaultLogger) Errorf(format string, args ...interface{}) {
	me.logf(me.error, level.ERROR, format, args...)
}

// NewDefaultLogger returns a configured ILogger.
func NewDefaultLogger(out io.Writer, scope string, n level.Level, calldepth int) *DefaultLogger {
	return new(DefaultLogger).Init(scope, n, calldepth).
		WithTrace(log.New(os.Stdout, getPrefix(scope, level.TRACE), log.LstdFlags|log.Lshortfile)).
		WithDebug(log.New(out, getPrefix(scope, level.DEBUG), log.LstdFlags|log.Lshortfile)).
		WithInfo(log.New(out, getPrefix(scope, level.INFO), log.LstdFlags|log.Lshortfile)).
		WithWarn(log.New(out, getPrefix(scope, level.WARN), log.LstdFlags|log.Lshortfile)).
		WithError(log.New(out, getPrefix(scope, level.ERROR), log.LstdFlags|log.Lshortfile))
}

func getPrefix(scope string, n level.Level) string {
	if n >= level.ERROR {
		return fmt.Sprintf("[ERROR] %s ", scope)
	}
	if n >= level.WARN {
		return fmt.Sprintf("[WARN ] %s ", scope)
	}
	if n >= level.INFO {
		return fmt.Sprintf("[INFO ] %s ", scope)
	}
	if n >= level.DEBUG {
		return fmt.Sprintf("[DEBUG] %s ", scope)
	}
	return fmt.Sprintf("%s ", scope)
}
