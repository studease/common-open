package log

import (
	"io"
	"strings"
	"time"

	"github.com/studease/common/log/internal/level"
	"github.com/studease/common/utils"
)

var (
	levels = map[string]level.Level{
		"none":   level.NONE,
		"trace":  level.TRACE,
		"debug":  level.DEBUG,
		"debug0": level.DEBUG0,
		"debug1": level.DEBUG1,
		"debug2": level.DEBUG2,
		"debug3": level.DEBUG3,
		"debug4": level.DEBUG4,
		"debug5": level.DEBUG5,
		"debug6": level.DEBUG6,
		"debug7": level.DEBUG7,
		"info":   level.INFO,
		"warn":   level.WARN,
		"error":  level.ERROR,
	}
)

// DefaultLoggerFactory creates new DefaultLogger
type DefaultLoggerFactory struct {
	level level.Level
	out   io.Writer
}

// Init this class
func (me *DefaultLoggerFactory) Init(n level.Level, out io.Writer) *DefaultLoggerFactory {
	me.level = n
	me.out = out
	return me
}

// NewLogger returns a configured ILogger for the given scope
func (me *DefaultLoggerFactory) NewLogger(scope string) ILogger {
	return NewDefaultLogger(me.out, strings.ToUpper(scope), me.level, DEFAULT_DEPTH)
}

// NewDefaultLoggerFactory creates a new DefaultLoggerFactory
func NewDefaultLoggerFactory(path string, level string) *DefaultLoggerFactory {
	path = time.Now().Format(path)

	f, err := utils.Create(path)
	if err != nil {
		Errorf("Failed to create log: %s", err)
		return nil
	}

	return new(DefaultLoggerFactory).Init(levels[level], f)
}
