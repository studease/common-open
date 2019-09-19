package bw

import (
	"io"
	"sync/atomic"
	"time"

	"github.com/studease/common/log"
)

// Writer with bandwidth limited
type Writer struct {
	logger log.ILogger
	mgr    *Manager
	avg    *int32
	dst    io.Writer
	cnt    int32
	next   int64
}

// Init this class
func (me *Writer) Init(mgr *Manager, logger log.ILogger) *Writer {
	me.mgr = mgr
	me.logger = logger
	me.avg = &mgr.avgOut

	if avg := atomic.LoadInt32(me.avg); avg > 0 {
		atomic.StoreInt64(&me.next, time.Now().Unix()+int64(time.Second))
	}

	return me
}

// Attach a io.Writer
func (me *Writer) Attach(dst io.Writer) {
	me.dst = dst
}

// Read reads up to len(b) bytes from the src reader
func (me *Writer) Write(p []byte) (n int, err error) {
	avg := atomic.LoadInt32(me.avg)
	if avg > 0 {
		defer func() {
			atomic.AddInt32(&me.cnt, int32(n))
		}()

		for cnt := atomic.LoadInt32(&me.cnt); cnt >= avg; cnt -= avg {
			dur := atomic.LoadInt64(&me.next) - time.Now().Unix()
			if dur > 0 {
				time.Sleep(time.Duration(dur))
				atomic.StoreInt32(&me.cnt, cnt-avg)
				atomic.StoreInt64(&me.next, time.Now().Unix()+int64(time.Second))
			}
		}
	}

	n, err = me.dst.Write(p)
	return
}

// Close removes itseft from the manager
func (me *Writer) Close() error {
	me.dst = nil

	me.mgr.mtx.Lock()
	defer me.mgr.mtx.Unlock()

	for i, w := range me.mgr.writers {
		if w == me {
			me.mgr.writers = append(me.mgr.writers[:i], me.mgr.writers[i+1:]...)
		}
	}

	return nil
}
