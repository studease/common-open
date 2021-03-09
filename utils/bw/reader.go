package bw

import (
	"io"
	"sync/atomic"
	"time"

	"github.com/studease/common/log"
)

// Reader with bandwidth limited
type Reader struct {
	logger log.ILogger
	mgr    *Manager
	avg    *int32
	src    io.Reader
	cnt    int32
	next   int64
}

// Init this class
func (me *Reader) Init(mgr *Manager, logger log.ILogger) *Reader {
	me.mgr = mgr
	me.logger = logger
	me.avg = &mgr.avgIn

	if avg := atomic.LoadInt32(me.avg); avg > 0 {
		atomic.StoreInt64(&me.next, time.Now().Unix()+int64(time.Second))
	}

	return me
}

// Attach a io.Reader
func (me *Reader) Attach(src io.Reader) {
	me.src = src
}

// Read reads up to len(b) bytes from the src reader
func (me *Reader) Read(p []byte) (n int, err error) {
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

	n, err = me.src.Read(p)
	return
}

// Close removes itseft from the manager
func (me *Reader) Close() error {
	me.src = nil

	me.mgr.mtx.Lock()
	defer me.mgr.mtx.Unlock()

	for i, r := range me.mgr.readers {
		if r == me {
			me.mgr.readers = append(me.mgr.readers[:i], me.mgr.readers[i+1:]...)
		}
	}

	return nil
}
