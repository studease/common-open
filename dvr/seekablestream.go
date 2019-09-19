package dvr

import (
	"sync"
)

// SeekableStream is used as the basic ISeekableStream
type SeekableStream struct {
	mtx    sync.RWMutex
	times  []uint32
	values []interface{}
}

// Init this class
func (me *SeekableStream) Init() *SeekableStream {
	return me
}

// AddPoint adds a seekable point
func (me *SeekableStream) AddPoint(timestamp uint32, value interface{}) {
	me.mtx.Lock()
	defer me.mtx.Unlock()

	me.times = append(me.times, timestamp)
	me.values = append(me.values, value)
}

// GetValue returns the value of the given timestamp
func (me *SeekableStream) GetValue(timestamp uint32, n int) []interface{} {
	me.mtx.RLock()
	defer me.mtx.RUnlock()

	var (
		index  = 0
		last   = len(me.times) - 1
		mid    = 0
		lbound = 0
		ubound = last
	)

	if last == -1 {
		return []interface{}{}
	}

	for lbound <= ubound {
		mid = lbound + int((ubound-lbound)/2)

		if mid == last || timestamp >= me.times[mid] && timestamp < me.times[mid+1] {
			index = mid
			break
		}

		if me.times[mid] < timestamp {
			lbound = mid + 1
		} else {
			ubound = mid - 1
		}
	}

	i := index
	j := i + n

	if j > last {
		j = last
	}

	if j-i < n {
		i = j - n
	}

	if i < 0 {
		i = 0
	}

	return me.values[i:j]
}

// Clear the stored data
func (me *SeekableStream) Clear() {
	me.times = make([]uint32, 0)
	me.values = make([]interface{}, 0)
}
