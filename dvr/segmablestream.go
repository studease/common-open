package dvr

import (
	"bytes"
	"os"
	"sync"
	"sync/atomic"

	"github.com/studease/common/av"
)

// SegmableStream is used as the basic ISegmableStream
type SegmableStream struct {
	SeekableStream

	mtx            sync.RWMutex
	buffers        map[av.Codec]av.ISourceBuffer
	sequenceNumber uint32
}

// Init this class
func (me *SegmableStream) Init() *SegmableStream {
	me.buffers = make(map[av.Codec]av.ISourceBuffer)
	return me
}

// AddSourceBuffer creates a new SourceBuffer of the given codec
func (me *SegmableStream) AddSourceBuffer(kind string, codec av.Codec) av.ISourceBuffer {
	me.mtx.Lock()
	defer me.mtx.Unlock()

	b := new(SourceBuffer).Init(&me.sequenceNumber)
	me.buffers[codec] = b

	return b
}

// RemoveSourceBuffer removes the given SourceBuffer
func (me *SegmableStream) RemoveSourceBuffer(b av.ISourceBuffer) {
	me.mtx.Lock()
	defer me.mtx.Unlock()

	for i, v := range me.buffers {
		if v == b {
			delete(me.buffers, i)
			break
		}
	}
}

// GetSegments returns at least n segments if possible around the timestamp
func (me *SegmableStream) GetSegments(timestamp uint32, n int) []interface{} {
	var (
		segments []interface{}
	)

	arr := me.GetValue(timestamp, n)

	for _, e := range arr {
		s := e.(*segEntry)

		if s.video != nil {
			segments = append(segments, s.video)
		}

		if s.audio != nil {
			segments = append(segments, s.audio)
		}
	}

	return segments
}

// SourceBuffer is a chunk of stream
type SourceBuffer struct {
	SeekableStream

	sequenceNumber *uint32
	buffer         bytes.Buffer
	dataType       byte
	timestamp      uint32
	Duration       uint32
	Size           int64
	Frames         int64
}

// Init this class
func (me *SourceBuffer) Init(sn *uint32) *SourceBuffer {
	me.SeekableStream.Init()
	me.sequenceNumber = sn
	return me
}

// Append a packet to the buffer
func (me *SourceBuffer) Append(p *av.Packet) {
	me.buffer.Write(p.Payload)
	me.dataType = p.DataType
	me.Duration += p.Timestamp
	me.Size += int64(p.Length)
	me.Frames++
}

// Dump all of the SourceBuffers to the disk, and add a seekable point
func (me *SourceBuffer) Write(name string, data []byte) (interface{}, error) {
	f, err := os.Create(name)
	if err != nil {
		return nil, err
	}

	defer f.Close()

	_, err = f.Write(me.buffer.Bytes())
	if err != nil {
		return nil, err
	}

	s := new(Segment).Init(name, atomic.LoadUint32(me.sequenceNumber), me.timestamp, me.Duration)
	me.AddPoint(s.Timestamp, s)

	me.buffer.Reset()
	me.timestamp = me.Duration
	me.Duration = 0
	me.Size = 0
	me.Frames = 0

	return s, nil
}

// Timestamp of the buffer
func (me *SourceBuffer) Timestamp() uint32 {
	return me.timestamp
}

// Bytes returns the buffer data
func (me *SourceBuffer) Bytes() []byte {
	return me.buffer.Bytes()
}

// Len returns the number of bytes of the unread portion of the buffer
func (me *SourceBuffer) Len() int {
	return me.buffer.Len()
}

// Segment holds info for seeking
type Segment struct {
	FileName       string
	SequenceNumber uint32
	Timestamp      uint32
	Duration       uint32
}

// Init this class
func (me *Segment) Init(name string, sn uint32, timestamp uint32, duration uint32) *Segment {
	me.FileName = name
	me.SequenceNumber = sn
	me.Timestamp = timestamp
	me.Duration = duration
	return me
}

// segEntry stores a pair of segments
type segEntry struct {
	audio *Segment
	video *Segment
}

// Init this class
func (me *segEntry) Init(audio, video *Segment) *segEntry {
	me.audio = audio
	me.video = video
	return me
}
