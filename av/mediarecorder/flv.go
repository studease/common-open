package mediarecorder

import (
	"fmt"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/studease/common/av"
	"github.com/studease/common/av/format/flv"
	"github.com/studease/common/events"
	ErrorEvent "github.com/studease/common/events/errorevent"
	Event "github.com/studease/common/events/event"
	MediaEvent "github.com/studease/common/events/mediaevent"
	MediaRecorderEvent "github.com/studease/common/events/mediarecorderevent"
	"github.com/studease/common/log"
)

func init() {
	Register("FLV", FLV{})
}

// FLV implementions IMediaRecorder.
type FLV struct {
	flv.FLV

	constraints *av.MediaRecorderConstraints
	logger      log.ILogger
	mtx         sync.RWMutex
	source      av.IMediaStream
	file        *os.File
	readyState  uint32

	packetListener *events.EventListener
	errorListener  *events.EventListener
	closeListener  *events.EventListener
}

// Init this class.
func (me *FLV) Init(constraints *av.MediaRecorderConstraints, logger log.ILogger) av.IMediaRecorder {
	me.FLV.Init(constraints.Mode, logger)
	me.constraints = constraints
	me.logger = logger
	me.readyState = StateInactive
	me.packetListener = events.NewListener(me.onPacket, 0)
	me.errorListener = events.NewListener(me.onError, 0)
	me.closeListener = events.NewListener(me.onClose, 0)
	return me
}

// Source attaches the IMediaStream as input.
func (me *FLV) Source(ms av.IMediaStream) {
	if ms == nil {
		me.Stop()
		return
	}

	me.mtx.Lock()
	defer me.mtx.Unlock()

	err := os.MkdirAll(me.constraints.Directory, os.ModePerm)
	if err != nil {
		panic(fmt.Sprintf("%v", err))
	}

	filename := me.constraints.FileName
	if me.constraints.Unique {
		now := time.Now()
		filename += fmt.Sprintf("-%d.flv", now.Unix())
	} else {
		filename += ".flv"
	}

	perm := os.O_RDWR | os.O_CREATE
	if me.constraints.Append {
		perm |= os.O_APPEND
	} else {
		perm |= os.O_TRUNC
	}

	f, err := os.OpenFile(me.constraints.Directory+"/"+filename, perm, 0666)
	if err != nil {
		panic(fmt.Sprintf("%v", err))
	}
	me.file = f

	if !me.constraints.Append {
		me.file.Write(flv.Header(me.Mode))
	}

	me.source = ms
	me.AddEventListener(MediaEvent.PACKET, me.packetListener)
	me.AddEventListener(ErrorEvent.ERROR, me.errorListener)
	me.AddEventListener(Event.CLOSE, me.closeListener)
}

func (me *FLV) onPacket(e *MediaEvent.MediaEvent) {
	if atomic.LoadUint32(&me.readyState) == StateRecording {
		_, err := me.file.Write(e.Packet.Payload)
		if err != nil {
			me.logger.Debugf(3, "MediaRecorder failed to write: %v", err)
			me.Stop()
		}
	}
}

func (me *FLV) onError(e *ErrorEvent.ErrorEvent) {
	me.logger.Debugf(0, "%s: %s", e.Name, e.Message)
	me.Stop()
}

func (me *FLV) onClose(e *Event.Event) {
	me.Stop()
}

// Start begins recording the source media stream.
func (me *FLV) Start() {
	if !atomic.CompareAndSwapUint32(&me.readyState, StateInactive, StateRecording) {
		me.DispatchEvent(ErrorEvent.New(ErrorEvent.ERROR, me, "InvalidStateError", fmt.Errorf("The MediaRecorder is not in the inactive state")))
		return
	}

	me.mtx.Lock()
	defer me.mtx.Unlock()

	// Note: If the observer decides to reject this event, just panic in its handler
	// rather than calling any other interfaces, which will cause a deadlock. Then
	// catch the exception outside and deal with that.
	me.DispatchEvent(MediaRecorderEvent.New(MediaRecorderEvent.START, me))
	me.FLV.Source(me.source)
}

// Pause is used to pause recording the source media stream.
func (me *FLV) Pause() {
	switch atomic.LoadUint32(&me.readyState) {
	case StateInactive:
		me.DispatchEvent(ErrorEvent.New(ErrorEvent.ERROR, me, "InvalidStateError", fmt.Errorf("The MediaRecorder can't be paused while it's not active")))
		return
	case StateRecording:
		me.mtx.Lock()
		defer me.mtx.Unlock()

		atomic.StoreUint32(&me.readyState, StatePaused)
		me.DispatchEvent(MediaRecorderEvent.New(MediaRecorderEvent.PAUSE, me))
	case StatePaused:
		me.logger.Debugf(3, "MediaRecorder is already paused.")
	}
}

// Resume is used to resume recording after when it has been previously paused.
func (me *FLV) Resume() {
	switch atomic.LoadUint32(&me.readyState) {
	case StateInactive:
		me.DispatchEvent(ErrorEvent.New(ErrorEvent.ERROR, me, "InvalidStateError", fmt.Errorf("The MediaRecorder can't be resumed while it's not paused")))
		return
	case StateRecording:
		me.logger.Debugf(3, "MediaRecorder is already paused.")
	case StatePaused:
		me.mtx.Lock()
		defer me.mtx.Unlock()

		atomic.StoreUint32(&me.readyState, StateRecording)
		me.DispatchEvent(MediaRecorderEvent.New(MediaRecorderEvent.RESUME, me))
	}
}

// Stop is used to stop recording the source media stream.
func (me *FLV) Stop() {
	switch atomic.LoadUint32(&me.readyState) {
	case StateInactive:
		me.DispatchEvent(ErrorEvent.New(ErrorEvent.ERROR, me, "InvalidStateError", fmt.Errorf("The MediaRecorder can't be stopped while it's not active")))
		return
	case StateRecording:
		fallthrough
	case StatePaused:
		me.mtx.Lock()
		defer me.mtx.Unlock()

		atomic.StoreUint32(&me.readyState, StateInactive)

		me.Close()
		me.RemoveEventListener(MediaEvent.PACKET, me.packetListener)
		me.RemoveEventListener(ErrorEvent.ERROR, me.errorListener)
		me.RemoveEventListener(Event.CLOSE, me.closeListener)
		if me.file != nil {
			me.file.Close()
		}
		me.DispatchEvent(MediaRecorderEvent.New(MediaRecorderEvent.STOP, me))
	}
}

// ReadyState returns ready state of this MediaRecorder.
func (me *FLV) ReadyState() uint32 {
	return atomic.LoadUint32(&me.readyState)
}
