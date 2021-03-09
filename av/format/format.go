package format

import (
	"reflect"
	"sync"

	"github.com/studease/common/av"
	"github.com/studease/common/events"
	MediaStreamTrackEvent "github.com/studease/common/events/mediastreamtrackevent"
	"github.com/studease/common/log"
	"github.com/studease/common/utils"
)

// Track kinds.
const (
	KindAudio = "audio"
	KindVideo = "video"
	KindExtra = "extra"
)

// IRemuxer states.
const (
	RemuxInactive uint32 = 0x00
	RemuxWaiting  uint32 = 0x01
	RemuxPumping  uint32 = 0x02
)

var (
	r = utils.NewRegister()
)

// MediaStreamTrack is the basic implemention of IMediaStreamTrack.
type MediaStreamTrack struct {
	events.EventDispatcher

	logger    log.ILogger
	id        int
	kind      string
	source    av.IMediaStreamTrackSource
	SN        uint32
	Timestamp uint32
}

// Init this class.
func (me *MediaStreamTrack) Init(kind string, source av.IMediaStreamTrackSource, logger log.ILogger) *MediaStreamTrack {
	me.EventDispatcher.Init(logger)
	me.logger = logger
	me.id = 0
	me.kind = kind
	me.source = source
	me.SN = 0
	me.Timestamp = 0
	return me
}

// ID returns the ID of this track.
func (me *MediaStreamTrack) ID() int {
	return me.id
}

// Kind returns the kind of this track.
func (me *MediaStreamTrack) Kind() string {
	return me.kind
}

// Source returns the source of this track.
func (me *MediaStreamTrack) Source() av.IMediaStreamTrackSource {
	return me.source
}

// Stop detaches from the source.
func (me *MediaStreamTrack) Stop() {
	me.SN = 0
	me.source = nil
}

// Clone clones this track and returns the new one.
func (me *MediaStreamTrack) Clone() av.IMediaStreamTrack {
	return new(MediaStreamTrack).Init(me.kind, me.source, me.logger)
}

// MediaStream is the basic implemention of IMediaStream.
type MediaStream struct {
	events.EventDispatcher

	logger     log.ILogger
	mtx        sync.RWMutex
	index      int
	tracks     []av.IMediaStreamTrack
	dataframes map[string]*av.Packet
	Info       av.Information
	Mode       uint32
}

// Init this class.
func (me *MediaStream) Init(logger log.ILogger) *MediaStream {
	me.EventDispatcher.Init(logger)
	me.Info.Init()
	me.logger = logger
	me.index = 1 // iSO/IEC 14496-12 8.3.2.3: Track IDs are never re-used and cannot be zero.
	me.tracks = make([]av.IMediaStreamTrack, 0)
	me.dataframes = make(map[string]*av.Packet)
	me.Mode = av.ModeAll
	return me
}

// AddTrack adds a new track to the stream.
func (me *MediaStream) AddTrack(track av.IMediaStreamTrack) {
	me.mtx.Lock()
	defer me.mtx.Unlock()

	for _, item := range me.tracks {
		if item == track {
			me.logger.Warnf("IMediaStreamTrack already added.")
			return
		}
	}

	basic, ok := track.(*MediaStreamTrack)
	if !ok {
		basic = reflect.ValueOf(track).Elem().FieldByName("MediaStreamTrack").Addr().Interface().(*MediaStreamTrack)
	}
	basic.id = me.index
	me.index++
	me.tracks = append(me.tracks, track)
	me.DispatchEvent(MediaStreamTrackEvent.New(MediaStreamTrackEvent.ADDTRACK, me, track))
}

// RemoveTrack removes the track from the stream.
func (me *MediaStream) RemoveTrack(track av.IMediaStreamTrack) {
	me.mtx.Lock()
	defer me.mtx.Unlock()

	for i, item := range me.tracks {
		if item == track {
			me.tracks = append(me.tracks[:i], me.tracks[i+1:]...)
			me.DispatchEvent(MediaStreamTrackEvent.New(MediaStreamTrackEvent.REMOVETRACK, me, track))
			break
		}
	}
}

// GetTrackByID eturns a MediaStreamTrack object representing the track with the specified ID string.
func (me *MediaStream) GetTrackByID(id int) av.IMediaStreamTrack {
	me.mtx.RLock()
	defer me.mtx.RUnlock()

	for _, item := range me.tracks {
		basic, ok := item.(*MediaStreamTrack)
		if !ok {
			basic = reflect.ValueOf(item).Elem().FieldByName("MediaStreamTrack").Addr().Interface().(*MediaStreamTrack)
		}
		if basic.id == id {
			return item
		}
	}
	return nil
}

// GetTracks returns a sequence that represents all the MediaStreamTrack objects in this stream.
func (me *MediaStream) GetTracks() []av.IMediaStreamTrack {
	me.mtx.RLock()
	defer me.mtx.RUnlock()

	tracks := make([]av.IMediaStreamTrack, 0)
	for _, item := range me.tracks {
		tracks = append(tracks, item)
	}
	return tracks
}

// GetAudioTracks returns all the MediaStreamTrack objects in this stream where MediaStreamTrack.Kind is audio.
func (me *MediaStream) GetAudioTracks() []av.IMediaStreamTrack {
	me.mtx.RLock()
	defer me.mtx.RUnlock()

	tracks := make([]av.IMediaStreamTrack, 0)
	for _, item := range me.tracks {
		if item.Kind() == KindAudio {
			tracks = append(tracks, item)
		}
	}
	return tracks
}

// GetVideoTracks returns all the MediaStreamTrack objects in this stream where MediaStreamTrack.Kind is video.
func (me *MediaStream) GetVideoTracks() []av.IMediaStreamTrack {
	me.mtx.RLock()
	defer me.mtx.RUnlock()

	tracks := make([]av.IMediaStreamTrack, 0)
	for _, item := range me.tracks {
		if item.Kind() == KindVideo {
			tracks = append(tracks, item)
		}
	}
	return tracks
}

// Attached checks whether the source is attached, and returns the track.
func (me *MediaStream) Attached(source av.IMediaStreamTrackSource) av.IMediaStreamTrack {
	me.mtx.RLock()
	defer me.mtx.RUnlock()

	for i, item := range me.tracks {
		if item.Source() == source {
			return me.tracks[i]
		}
	}
	return nil
}

// Information returns the Information of this stream.
func (me *MediaStream) Information() *av.Information {
	return &me.Info
}

// SetDataFrame stores a data frame with the given key.
func (me *MediaStream) SetDataFrame(key string, pkt *av.Packet) {
	me.mtx.Lock()
	defer me.mtx.Unlock()

	me.dataframes[key] = pkt
}

// GetDataFrame searches for the data frame by the given key.
func (me *MediaStream) GetDataFrame(key string) *av.Packet {
	me.mtx.RLock()
	defer me.mtx.RUnlock()

	return me.dataframes[key]
}

// ClearDataFrame deletes the data frame by the given key.
func (me *MediaStream) ClearDataFrame(key string) {
	me.mtx.Lock()
	defer me.mtx.Unlock()

	delete(me.dataframes, key)
}

// Close stops and removes all the MediaStreamTrack objects in the stream.
func (me *MediaStream) Close() {
	me.mtx.Lock()
	defer me.mtx.Unlock()

	for _, item := range me.tracks {
		item.Stop()
		me.DispatchEvent(MediaStreamTrackEvent.New(MediaStreamTrackEvent.REMOVETRACK, me, item))
	}
	me.index = 1
	me.tracks = make([]av.IMediaStreamTrack, 0)
}

// Register an IRemuxer with the given name.
func Register(name string, remuxer interface{}) {
	r.Add(name, remuxer)
}

// New creates a registered IRemuxer by the name.
func New(name string, mode uint32, factory log.ILoggerFactory) av.IRemuxer {
	if remuxer := r.New(name); remuxer != nil {
		return remuxer.(av.IRemuxer).Init(mode, factory.NewLogger(name))
	}
	return nil
}
