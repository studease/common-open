package log

// ILogger is the basic logger interface
type ILogger interface {
	Trace(s string)
	Tracef(format string, args ...interface{})
	Debug(n uint32, s string)
	Debugf(n uint32, format string, args ...interface{})
	Info(s string)
	Infof(format string, args ...interface{})
	Warn(s string)
	Warnf(format string, args ...interface{})
	Error(s string)
	Errorf(format string, args ...interface{})
}

// ILoggerFactory is the basic logger factory interface
type ILoggerFactory interface {
	NewLogger(scope string) ILogger
}

// Trace emits the preformatted message if the logger is at or below trace-level
func Trace(s string) {
	std.Trace(s)
}

// Tracef formats and emits a message if the logger is at or below trace-level
func Tracef(format string, args ...interface{}) {
	std.Tracef(format, args...)
}

// Debug emits the preformatted message if the logger is at or below debug-level
func Debug(n uint32, s string) {
	std.Debug(n, s)
}

// Debugf formats and emits a message if the logger is at or below debug-level
func Debugf(n uint32, format string, args ...interface{}) {
	std.Debugf(n, format, args...)
}

// Info emits the preformatted message if the logger is at or below info-level
func Info(s string) {
	std.Info(s)
}

// Infof formats and emits a message if the logger is at or below info-level
func Infof(format string, args ...interface{}) {
	std.Infof(format, args...)
}

// Warn emits the preformatted message if the logger is at or below warn-level
func Warn(s string) {
	std.Warn(s)
}

// Warnf formats and emits a message if the logger is at or below warn-level
func Warnf(format string, args ...interface{}) {
	std.Warnf(format, args...)
}

// Error emits the preformatted message if the logger is at or below error-level
func Error(s string) {
	std.Error(s)
}

// Errorf formats and emits a message if the logger is at or below error-level
func Errorf(format string, args ...interface{}) {
	std.Errorf(format, args...)
}
