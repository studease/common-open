package cmaf

import (
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/studease/common/av"
	"github.com/studease/common/av/codec/aac"
	"github.com/studease/common/av/codec/avc"
	"github.com/studease/common/av/format"
	"github.com/studease/common/events"
	ErrorEvent "github.com/studease/common/events/errorevent"
	Event "github.com/studease/common/events/event"
	MediaEvent "github.com/studease/common/events/mediaevent"
	MediaStreamTrackEvent "github.com/studease/common/events/mediastreamtrackevent"
	"github.com/studease/common/log"
)

func init() {
	format.Register("CMAF", CMAF{})
}

// CMAF MediaStream, implements IDemuxer, IRemuxer.
type CMAF struct {
	format.MediaStream

	Mode        uint32
	logger      log.ILogger
	mtx         sync.RWMutex
	source      av.IMediaStream
	InitSegment *MediaChunk // init segment with all tracks
	readyState  uint32

	addtrackListener    *events.EventListener
	removetrackListener *events.EventListener
	packetListener      *events.EventListener
	errorListener       *events.EventListener
	closeListener       *events.EventListener
}

// Init this class.
func (me *CMAF) Init(mode uint32, logger log.ILogger) av.IRemuxer {
	me.MediaStream.Init(logger)
	me.Mode = mode
	me.logger = logger
	me.readyState = format.RemuxInactive
	me.addtrackListener = events.NewListener(me.onAddTrack, 0)
	me.removetrackListener = events.NewListener(me.onRemoveTrack, 0)
	me.packetListener = events.NewListener(me.onPacket, 0)
	me.errorListener = events.NewListener(me.onError, 0)
	me.closeListener = events.NewListener(me.onClose, 0)
	return me
}

// Append parses buffer.
func (me *CMAF) Append(data []byte) {
	// TODO(spencerlau): Parse cmaf boxes
	me.DispatchEvent(ErrorEvent.New(ErrorEvent.ERROR, me, "NotSupportedError", fmt.Errorf("The operation is not supported")))
}

// Reset clears IDemuxer cache, and closes IMediaStream.
func (me *CMAF) Reset() {
	me.MediaStream.Close()
	me.Init(me.Mode, me.logger)
}

// Source attaches the IMediaStream as input.
func (me *CMAF) Source(ms av.IMediaStream) {
	if ms == nil {
		me.Close()
		return
	}

	me.mtx.Lock()
	defer me.mtx.Unlock()

	me.source = ms
	atomic.StoreUint32(&me.readyState, format.RemuxWaiting)

	onMetaData := ms.GetDataFrame("onMetaData")
	if onMetaData != nil {
		me.SetDataFrame("onMetaData", onMetaData)
	}
	tracks := ms.GetTracks()
	for _, item := range tracks {
		if item.Kind() == format.KindVideo && (me.Mode&av.ModeVideo&av.ModeKeyframe) == 0 || item.Kind() == format.KindAudio && (me.Mode&av.ModeAudio) == 0 {
			continue
		}
		track := item.Clone()
		me.AddTrack(track)
		source := track.Source()
		if infoframe := source.GetInfoFrame(); infoframe != nil {
			me.generateInitSegment(infoframe.Kind, source.Kind(), track)
		}
		source.AddEventListener(MediaEvent.PACKET, me.packetListener)
	}

	ms.AddEventListener(MediaStreamTrackEvent.ADDTRACK, me.addtrackListener)
	ms.AddEventListener(MediaStreamTrackEvent.REMOVETRACK, me.removetrackListener)
	ms.AddEventListener(MediaEvent.PACKET, me.packetListener)
	ms.AddEventListener(ErrorEvent.ERROR, me.errorListener)
	ms.AddEventListener(Event.CLOSE, me.closeListener)
}

func (me *CMAF) onAddTrack(e *MediaStreamTrackEvent.MediaStreamTrackEvent) {
	switch e.Track.Kind() {
	case format.KindVideo:
		if (me.Mode & av.ModeVideo & av.ModeKeyframe) == 0 {
			return
		}
	case format.KindAudio:
		if (me.Mode & av.ModeAudio) == 0 {
			return
		}
	default:
		me.logger.Debugf(2, "Ignored unrecognized track: kind=%s.", e.Track.Kind())
		return
	}

	source := e.Track.Source()
	if me.Attached(source) == nil {
		track := new(MediaStreamTrack).Init(e.Track.Kind(), source, me.logger)
		me.AddTrack(track)
		source.AddEventListener(MediaEvent.PACKET, me.packetListener)
	}
}

func (me *CMAF) onRemoveTrack(e *MediaStreamTrackEvent.MediaStreamTrackEvent) {
	source := e.Track.Source()
	track := me.Attached(source)
	if track != nil {
		source.RemoveEventListener(MediaEvent.PACKET, me.packetListener)
		me.RemoveTrack(track)
	}
}

func (me *CMAF) onPacket(e *MediaEvent.MediaEvent) {
	switch e.Packet.Kind {
	case av.KindAudio:
		me.onAudioPacket(e.Packet)
	case av.KindVideo:
		me.onVideoPacket(e.Packet)
	case av.KindScript:
		me.onDataPacket(e.Packet)
	default:
		me.logger.Errorf("Unrecognized packet: %s", e.Packet.Kind)
	}
}

func (me *CMAF) onDataPacket(pkt *av.Packet) {
	key := pkt.Get("Key").(string)
	me.SetDataFrame(key, pkt)
}

func (me *CMAF) onAudioPacket(pkt *av.Packet) {
	track := me.GetAudioTracks()[0]
	source := track.Source()

	switch pkt.Codec {
	case "AAC":
		switch pkt.Get("DataType").(byte) {
		case aac.SPECIFIC_CONFIG:
			me.generateInitSegment(pkt.Kind, source.Kind(), track)
		case aac.RAW_FRAME_DATA:
			if source.GetInfoFrame() == nil || atomic.LoadUint32(&me.readyState) != format.RemuxPumping {
				return
			}
			me.generateSegment(track, pkt)
		default:
			me.logger.Errorf("Unrecognized AAC packet type: 0x%02X", pkt.Get("DataType").(byte))
		}
	default:
		me.logger.Errorf("Unrecognized codec: %s", pkt.Codec)
	}
}

func (me *CMAF) onVideoPacket(pkt *av.Packet) {
	track := me.GetVideoTracks()[0]
	source := track.Source()

	switch pkt.Codec {
	case "AVC":
		switch pkt.Get("DataType").(byte) {
		case avc.SEQUENCE_HEADER:
			me.generateInitSegment(pkt.Kind, source.Kind(), track)
		case avc.NALU:
			if pkt.Get("Keyframe").(bool) && atomic.CompareAndSwapUint32(&me.readyState, format.RemuxWaiting, format.RemuxPumping) {
				me.Info.TimeBase = pkt.Timestamp
				// If you are willing to use interleaved mode, all of the info frames should be ahead of any media frames.
				audios := me.GetAudioTracks()
				if (me.Mode&av.ModeAudio) == 0 || len(audios) == 0 || (me.Mode&av.ModeInterleaved) != 0 && audios[0].Source().GetInfoFrame() != nil {
					tracks := me.GetTracks()
					me.generateInitSegment(av.KindScript, "", tracks...)
				}
			}
			if source.GetInfoFrame() == nil || atomic.LoadUint32(&me.readyState) != format.RemuxPumping || (me.Mode&av.ModeKeyframe) == av.ModeKeyframe && !pkt.Get("Keyframe").(bool) {
				return
			}
			fallthrough
		case avc.END_OF_SEQUENCE:
			me.generateSegment(track, pkt)
		default:
			me.logger.Errorf("Unrecognized AVC packet type: 0x%02X", pkt.Get("DataType").(byte))
		}
	default:
		me.logger.Errorf("Unrecognized codec: %s", pkt.Codec)
	}
}

func (me *CMAF) onError(e *ErrorEvent.ErrorEvent) {
	me.logger.Debugf(0, "%s: %s", e.Name, e.Message)
	me.Close()
}

func (me *CMAF) onClose(e *Event.Event) {
	me.Close()
}

// generateInitSegment generates an cmaf init segment of the given tracks.
func (me *CMAF) generateInitSegment(kind string, codec string, tracks ...av.IMediaStreamTrack) {
	source := tracks[0].Source()
	infoframe := source.GetInfoFrame()

	ftyp := me.ftyp()
	moov := me.moov(tracks...)
	data := merge(ftyp, moov)

	seg := me.format(kind, codec, 0, data)
	switch kind {
	case av.KindAudio:
		fallthrough
	case av.KindVideo:
		seg.Extends(infoframe)
	}
	me.DispatchEvent(MediaEvent.New(MediaEvent.PACKET, me, seg))
}

// generateSegment generates an cmaf segment of the given track with the packet.
func (me *CMAF) generateSegment(track av.IMediaStreamTrack, pkt *av.Packet) {
	var trk = track.(*MediaStreamTrack)
	var source = track.Source()
	var ctx = source.Context()
	if track.Kind() == format.KindVideo {
		if pkt.Get("Keyframe").(bool) {
			ctx.Flags.SampleDependsOn = 2
			ctx.Flags.SampleIsDependedOn = 1
		} else {
			ctx.Flags.SampleDependsOn = 1
			ctx.Flags.SampleIsDependedOn = 0
		}
	}

	trk.SN++
	prft := me.prft(track)
	moof := me.moof(track, pkt)
	mdat := me.mdat(pkt.Get("Data").([]byte))
	data := merge(prft, moof, mdat)

	seg := me.format(pkt.Kind, pkt.Codec, pkt.Timestamp, data)
	seg.Extends(pkt)
	me.DispatchEvent(MediaEvent.New(MediaEvent.PACKET, me, seg))

	var delta = pkt.Get("DTS").(uint32) - me.Info.TimeBase - trk.Timestamp
	trk.Timestamp += ctx.RefSampleDuration + delta
}

func (me *CMAF) format(kind string, codec string, timestamp uint32, data []byte) *av.Packet {
	seg := new(av.Packet).Init()
	seg.Kind = kind
	seg.Codec = codec
	seg.Length = uint32(len(data))
	seg.Timestamp = timestamp - me.Info.TimeBase
	seg.StreamID = 0
	seg.Position = 0
	seg.Payload = data
	return seg
}

// SetDataFrame stores a data frame with the given key.
func (me *CMAF) SetDataFrame(key string, pkt *av.Packet) {
	me.Info = *me.source.Information()
	me.MediaStream.SetDataFrame(key, pkt)
}

// Close detaches IRemuxer source, and closes IMediaStream.
func (me *CMAF) Close() {
	switch atomic.LoadUint32(&me.readyState) {
	case format.RemuxWaiting:
		fallthrough
	case format.RemuxPumping:
		me.mtx.Lock()
		defer me.mtx.Unlock()

		atomic.StoreUint32(&me.readyState, format.RemuxInactive)
		me.DispatchEvent(Event.New(Event.CLOSE, me))

		tracks := me.GetTracks()
		for _, item := range tracks {
			source := item.Source()
			source.RemoveEventListener(MediaEvent.PACKET, me.packetListener)
		}
		if ms := me.source; ms != nil {
			ms.RemoveEventListener(MediaStreamTrackEvent.ADDTRACK, me.addtrackListener)
			ms.RemoveEventListener(MediaStreamTrackEvent.REMOVETRACK, me.removetrackListener)
			ms.RemoveEventListener(MediaEvent.PACKET, me.packetListener)
			ms.RemoveEventListener(ErrorEvent.ERROR, me.errorListener)
			ms.RemoveEventListener(Event.CLOSE, me.closeListener)
			me.source = nil
		}
		me.MediaStream.Close()
	}
}
