package mux

import (
	"fmt"
	"regexp"
	"strings"
	"sync"

	"github.com/studease/common/av"
	"github.com/studease/common/av/format"
	"github.com/studease/common/events"
	MediaEvent "github.com/studease/common/events/mediaevent"
	"github.com/studease/common/log"
)

var (
	pathRe, _ = regexp.Compile("^/([-_.[:alnum:]]+)(?:/([-_.[:alnum:]]+))?/([-_.[:alnum:]]+)$")
)

// ReadableStream is used as the basic IReadableStream
type ReadableStream struct {
	events.EventDispatcher
	format.MediaStream

	logger         log.ILogger
	factory        log.ILoggerFactory
	mtx            sync.RWMutex
	info           av.Information
	path           string
	appName        string
	instName       string
	name           string
	parameters     string
	dataframes     map[string]*av.Packet
	infoFrame      *av.Packet
	audioInfoFrame *av.Packet
	videoInfoFrame *av.Packet
}

// Init this class
func (me *ReadableStream) Init(path string, logger log.ILogger, factory log.ILoggerFactory) *ReadableStream {
	me.EventDispatcher.Init(logger)
	me.MediaStream.Init()
	me.info.Init()
	me.logger = logger
	me.factory = factory
	me.dataframes = make(map[string]*av.Packet)

	arr := strings.Split(path, "?")
	me.path = arr[0]
	if len(arr) > 1 {
		me.parameters = arr[1]
	}

	arr = pathRe.FindStringSubmatch(me.path)
	if arr == nil {
		panic(fmt.Sprintf("bad path format: %s", me.path))
	}

	me.appName = arr[1]
	me.instName = arr[2]
	me.name = arr[3]

	return me
}

// Sink a packet into the stream
func (me *ReadableStream) Sink(pkt *av.Packet) error {
	return nil
}

// AppName returns the application name
func (me *ReadableStream) AppName() string {
	return me.appName
}

// InstName returns the instance name
func (me *ReadableStream) InstName() string {
	return me.instName
}

// Name returns the name
func (me *ReadableStream) Name() string {
	return me.name
}

// Parameters returns the stored parameters
func (me *ReadableStream) Parameters() string {
	return me.parameters
}

// SetDataFrame stores a data frame with the given key
func (me *ReadableStream) SetDataFrame(key string, p *av.Packet) {
	me.mtx.Lock()
	defer me.mtx.Unlock()

	p.Handler = "@setDataFrame"

	me.dataframes[key] = p
	me.DispatchEvent(MediaEvent.New(MediaEvent.DATA, me, p))
}

// ClearDataFrame deletes the data frame by the given key
func (me *ReadableStream) ClearDataFrame(key string) {
	me.mtx.Lock()
	defer me.mtx.Unlock()

	p := me.dataframes[key]
	p.Handler = "@clearDataFrame"

	delete(me.dataframes, key)
	me.DispatchEvent(MediaEvent.New(MediaEvent.DATA, me, p))
}

// GetDataFrame searches for the data frame by the given key
func (me *ReadableStream) GetDataFrame(key string) *av.Packet {
	me.mtx.RLock()
	defer me.mtx.RUnlock()

	return me.dataframes[key]
}

// Information returns the associated Information
func (me *ReadableStream) Information() *av.Information {
	return &me.info
}

// InfoFrame returns the audio info frame
func (me *ReadableStream) InfoFrame() *av.Packet {
	return me.infoFrame
}

// AudioInfoFrame returns the audio info frame
func (me *ReadableStream) AudioInfoFrame() *av.Packet {
	return me.audioInfoFrame
}

// VideoInfoFrame returns the video info frame
func (me *ReadableStream) VideoInfoFrame() *av.Packet {
	return me.videoInfoFrame
}
