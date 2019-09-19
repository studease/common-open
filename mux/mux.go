package mux

import (
	"strings"

	"github.com/studease/common/av"
	"github.com/studease/common/av/utils/amf"
	"github.com/studease/common/events"
	"github.com/studease/common/log"
	"github.com/studease/common/utils"
)

// Muxer types
const (
	TYPE_FLV  = "FLV"
	TYPE_FMP4 = "FMP4"
	TYPE_RTP  = "RTP"
)

// Muxer modes
const (
	MODE_NONE        uint32 = 0x00
	MODE_AUDIO       uint32 = 0x01
	MODE_VIDEO       uint32 = 0x02
	MODE_ALL         uint32 = 0x03
	MODE_KEYFRAME    uint32 = 0x06
	MODE_ADVANCED    uint32 = 0x08
	MODE_LOW_LATENCY uint32 = 0x10
	MODE_MANUAL      uint32 = 0x40000000
	MODE_OFF         uint32 = 0x80000000
)

// Muxer states
const (
	STATE_INITIALIZED uint32 = 0x00
	STATE_DETECTING   uint32 = 0x01
	STATE_ALIVE       uint32 = 0x02
	STATE_CLOSING     uint32 = 0x03
	STATE_CLOSED      uint32 = 0x04
)

var (
	r = utils.NewRegister()

	modes = map[string]uint32{
		"audio":       MODE_AUDIO,
		"video":       MODE_VIDEO,
		"all":         MODE_ALL,
		"keyframe":    MODE_KEYFRAME,
		"advanced":    MODE_ADVANCED,
		"low-latency": MODE_LOW_LATENCY,
		"manual":      MODE_MANUAL,
		"off":         MODE_OFF,
	}
)

// IMuxer defines methods to attach an IReadableStream
type IMuxer interface {
	events.IEventDispatcher

	Init(mode uint32, logger log.ILogger, factory log.ILoggerFactory) IMuxer
	Attach(stream av.IReadableStream)
	Append(data []byte) (int, error)
	Information() *av.Information
	Stream() av.IReadableStream
	Mode() uint32
	ReadyState() uint32
	Close()
}

// Register an IMuxer with the given name
func Register(name string, muxer interface{}) {
	r.Add(name, muxer)
}

// New creates a registered IMuxer by the name
func New(name string, mode uint32, factory log.ILoggerFactory) IMuxer {
	if m := r.New(name); m != nil {
		return m.(IMuxer).Init(mode, factory.NewLogger(name), factory)
	}

	return nil
}

// Mode parse the given string into modes
func Mode(s string, sep string) uint32 {
	var (
		mode uint32
	)

	arr := strings.Split(s, sep)

	for _, v := range arr {
		n, ok := modes[v]
		if ok {
			mode |= n
		}
	}

	return mode
}

func onMetaData(info *av.Information, o *amf.Value) {
	if v := o.Get("duration"); v != nil {
		info.Duration = uint32(v.Double() * 1000)
	}

	if v := o.Get("filesize"); v != nil {
		info.Size = int64(v.Double() * 1000)
	}

	if v := o.Get("width"); v != nil {
		info.Width = uint32(v.Double())
	}

	if v := o.Get("height"); v != nil {
		info.Height = uint32(v.Double())
	}

	if v := o.Get("audiodatarate"); v != nil {
		info.AudioDataRate = uint32(v.Double())
		info.BitRate += info.AudioDataRate
	}

	if v := o.Get("videodatarate"); v != nil {
		info.VideoDataRate = uint32(v.Double())
		info.BitRate += info.VideoDataRate
	}

	if v := o.Get("framerate"); v != nil {
		info.FrameRate.Num = v.Double()
		info.FrameRate.Den = 1
	}

	if v := o.Get("audiosamplerate"); v != nil {
		info.SampleRate = uint32(v.Double())
	}

	if v := o.Get("audiosamplesize"); v != nil {
		info.SampleSize = uint32(v.Double())
	}

	if v := o.Get("audiochannels"); v != nil {
		info.Channels = uint32(v.Double())
	}
}
