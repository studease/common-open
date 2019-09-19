package bw

import (
	"io"
	"sync"
	"sync/atomic"

	"github.com/studease/common/log"
)

// Manager is used to control io bandwidth
type Manager struct {
	in       int32
	out      int32
	mtx      sync.RWMutex
	readers  []io.Reader
	writers  []io.Writer
	avgIn    int32
	avgOut   int32
	bytesIn  int32
	bytesOut int32
}

// Init this class
func (me *Manager) Init(in int32, out int32) *Manager {
	me.in = in / 8
	me.out = out / 8
	return me
}

// NewReader returns an io.Reader with bandwidth limited
func (me *Manager) NewReader(logger log.ILogger) *Reader {
	me.mtx.Lock()
	defer me.mtx.Unlock()

	n := int32(len(me.readers) + 1)
	atomic.StoreInt32(&me.avgIn, me.in/n)

	r := new(Reader).Init(me, logger)
	me.readers = append(me.readers, r)

	return r
}

// NewWriter returns an io.Writer with bandwidth limited
func (me *Manager) NewWriter(logger log.ILogger) *Writer {
	me.mtx.Lock()
	defer me.mtx.Unlock()

	n := int32(len(me.writers) + 1)
	atomic.StoreInt32(&me.avgOut, me.out/n)

	w := new(Writer).Init(me, logger)
	me.writers = append(me.writers, w)

	return w
}
