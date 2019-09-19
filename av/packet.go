package av

import "github.com/studease/common/av/utils/amf"

// Packet types
const (
	TYPE_UNKNOWN Type = iota
	TYPE_AUDIO
	TYPE_VIDEO
	TYPE_DATA
)

// RTMP/FLV video frame types
const (
	KEYFRAME               = 0x1
	INTER_FRAME            = 0x2
	DISPOSABLE_INTER_FRAME = 0x3
	GENERATED_KEYFRAME     = 0x4
	INFO_OR_COMMAND_FRAME  = 0x5
)

// Type represents the type of Packet
type Type int

// Codec represents the ID of codecs
type Codec uint32

// Packet in a media stream
type Packet struct {
	Context   IMediaContext
	Type      Type
	Codec     Codec
	Length    uint32
	Timestamp uint32
	StreamID  uint32
	Payload   []byte

	// For RTMP/FLV video frames
	FrameType byte // 0xF0

	// For RTMP/FLV audio frames
	SampleRate byte // 0000 1100
	SampleSize byte // 0000 0010
	SampleType byte // 0000 0001

	DataType byte

	// For RTMP/FLV data frames
	Handler string
	Key     string
	Value   *amf.Value
}

// Init this class
func (me *Packet) Init() *Packet {
	return me
}

// Clone this class
func (me *Packet) Clone(data []byte) *Packet {
	pkt := new(Packet).Init()
	*pkt = *me
	pkt.Length = uint32(len(data))
	pkt.Payload = data
	return pkt
}
