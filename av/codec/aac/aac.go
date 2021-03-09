package aac

import (
	"fmt"

	"github.com/studease/common/av"
	"github.com/studease/common/av/codec"
	"github.com/studease/common/av/utils"
	"github.com/studease/common/events"
	MediaEvent "github.com/studease/common/events/mediaevent"
	"github.com/studease/common/log"
)

// AOT types.
const (
	AOT_NULL            uint8 = 0
	AOT_AAC_MAIN        uint8 = 1  // Main
	AOT_AAC_LC          uint8 = 2  // Low Complexity
	AOT_AAC_SSR         uint8 = 3  // Scalable Sample Rate
	AOT_AAC_LTP         uint8 = 4  // Long Term Prediction
	AOT_SBR             uint8 = 5  // Spectral Band Replication
	AOT_AAC_SCALABLE    uint8 = 6  // Scalable
	AOT_TWINVQ          uint8 = 7  // Twin Vector Quantizer
	AOT_CELP            uint8 = 8  // Code Excited Linear Prediction
	AOT_HVXC            uint8 = 9  // Harmonic Vector eXcitation Coding
	AOT_TTSI            uint8 = 12 // Text-To-Speech Interface
	AOT_MAINSYNTH       uint8 = 13 // Main Synthesis
	AOT_WAVESYNTH       uint8 = 14 // Wavetable Synthesis
	AOT_MIDI            uint8 = 15 // General MIDI
	AOT_SAFX            uint8 = 16 // Algorithmic Synthesis and Audio Effects
	AOT_ER_AAC_LC       uint8 = 17 // Error Resilient Low Complexity
	AOT_ER_AAC_LTP      uint8 = 19 // Error Resilient Long Term Prediction
	AOT_ER_AAC_SCALABLE uint8 = 20 // Error Resilient Scalable
	AOT_ER_TWINVQ       uint8 = 21 // Error Resilient Twin Vector Quantizer
	AOT_ER_BSAC         uint8 = 22 // Error Resilient Bit-Sliced Arithmetic Coding
	AOT_ER_AAC_LD       uint8 = 23 // Error Resilient Low Delay
	AOT_ER_CELP         uint8 = 24 // Error Resilient Code Excited Linear Prediction
	AOT_ER_HVXC         uint8 = 25 // Error Resilient Harmonic Vector eXcitation Coding
	AOT_ER_HILN         uint8 = 26 // Error Resilient Harmonic and Individual Lines plus Noise
	AOT_ER_PARAM        uint8 = 27 // Error Resilient Parametric
	AOT_SSC             uint8 = 28 // SinuSoidal Coding
	AOT_PS              uint8 = 29 // Parametric Stereo
	AOT_SURROUND        uint8 = 30 // MPEG Surround
	AOT_ESCAPE          uint8 = 31 // Escape Value
	AOT_L1              uint8 = 32 // Layer 1
	AOT_L2              uint8 = 33 // Layer 2
	AOT_L3              uint8 = 34 // Layer 3
	AOT_DST             uint8 = 35 // Direct Stream Transfer
	AOT_ALS             uint8 = 36 // Audio LosslesS
	AOT_SLS             uint8 = 37 // Scalable LosslesS
	AOT_SLS_NON_CORE    uint8 = 38 // Scalable LosslesS (non core)
	AOT_ER_AAC_ELD      uint8 = 39 // Error Resilient Enhanced Low Delay
	AOT_SMR_SIMPLE      uint8 = 40 // Symbolic Music Representation Simple
	AOT_SMR_MAIN        uint8 = 41 // Symbolic Music Representation Main
	AOT_USAC_NOSBR      uint8 = 42 // Unified Speech and Audio Coding (no SBR)
	AOT_SAOC            uint8 = 43 // Spatial Audio Object Coding
	AOT_LD_SURROUND     uint8 = 44 // Low Delay MPEG Surround
	AOT_USAC            uint8 = 45 // Unified Speech and Audio Coding
)

// Data types.
const (
	SPECIFIC_CONFIG = 0x00
	RAW_FRAME_DATA  = 0x01
)

var (
	// Rates used in FLV.
	Rates = [4]int{5500, 11025, 22050, 44100}
	// SamplingFrequencys which AAC supports.
	SamplingFrequencys = [16]uint32{96000, 88200, 64000, 48000, 44100, 32000, 24000, 22050, 16000, 12000, 11025, 8000, 7350}
	// Channels for quick mapping.
	Channels = [8]uint16{0, 1, 2, 3, 4, 5, 6, 8}
	// SilentFrames of AAC.
	SilentFrames = [][]byte{
		[]byte{0x00, 0xc8, 0x00, 0x80, 0x23, 0x80},
		[]byte{0x21, 0x00, 0x49, 0x90, 0x02, 0x19, 0x00, 0x23, 0x80},
		[]byte{0x00, 0xc8, 0x00, 0x80, 0x20, 0x84, 0x01, 0x26, 0x40, 0x08, 0x64, 0x00, 0x8e},
		[]byte{0x00, 0xc8, 0x00, 0x80, 0x20, 0x84, 0x01, 0x26, 0x40, 0x08, 0x64, 0x00, 0x80, 0x2c, 0x80, 0x08, 0x02, 0x38},
		[]byte{0x00, 0xc8, 0x00, 0x80, 0x20, 0x84, 0x01, 0x26, 0x40, 0x08, 0x64, 0x00, 0x82, 0x30, 0x04, 0x99, 0x00, 0x21, 0x90, 0x02, 0x38},
		[]byte{0x00, 0xc8, 0x00, 0x80, 0x20, 0x84, 0x01, 0x26, 0x40, 0x08, 0x64, 0x00, 0x82, 0x30, 0x04, 0x99, 0x00, 0x21, 0x90, 0x02, 0x00, 0xb2, 0x00, 0x20, 0x08, 0xe0},
	}
)

func init() {
	codec.Register("AAC", AAC{})
}

// AAC IMediaStreamTrackSource.
type AAC struct {
	events.EventDispatcher

	logger    log.ILogger
	info      *av.Information
	infoframe *av.Packet
	ctx       av.Context

	// Specific Config
	AudioObjectType                 uint8 // 5 bits
	SamplingFrequencyIndex          uint8 // 4 bits
	SamplingFrequency               uint32
	ChannelConfiguration            uint8 // 4 bits
	Channels                        uint16
	ExtensionAudioObjectType        uint8  // 5 bits
	ExtensionSamplingFrequencyIndex uint8  // 4 bits
	ExtensionSamplingFrequency      uint32 // 24 bits
	ExtensionChannelConfiguration   uint8  // 4 bits
	Config                          []byte
}

// Init this class.
func (me *AAC) Init(info *av.Information, logger log.ILogger) av.IMediaStreamTrackSource {
	me.EventDispatcher.Init(logger)
	me.logger = logger
	me.info = info
	me.infoframe = nil
	me.ctx.MimeType = "audio/mp4"
	me.ctx.Codec = ""
	me.ctx.RefSampleDuration = me.info.Timescale * 1024 / 44100
	me.ctx.Flags.IsLeading = 0
	me.ctx.Flags.SampleDependsOn = 1
	me.ctx.Flags.SampleIsDependedOn = 0
	me.ctx.Flags.SampleHasRedundancy = 0
	me.ctx.Flags.IsNonSync = 0
	return me
}

// Kind returns the source name.
func (me *AAC) Kind() string {
	return "AAC"
}

// Context returns the source context.
func (me *AAC) Context() *av.Context {
	return &me.ctx
}

// SetInfoFrame stores the info frame for decoding.
func (me *AAC) SetInfoFrame(pkt *av.Packet) {
	me.infoframe = pkt
}

// GetInfoFrame returns the info frame.
func (me *AAC) GetInfoFrame() *av.Packet {
	return me.infoframe
}

// Sink a packet into the source.
func (me *AAC) Sink(pkt *av.Packet) {
	me.DispatchEvent(MediaEvent.New(MediaEvent.PACKET, me, pkt))
}

// Parse an AAC packet.
func (me *AAC) Parse(pkt *av.Packet) error {
	if pkt.Left() < 1 {
		err := fmt.Errorf("data not enough while parsing AAC packet")
		me.logger.Errorf("%v", err)
		return err
	}

	me.info.Timestamp = pkt.Timestamp

	pkt.Set("DataType", pkt.Payload[pkt.Position])
	pkt.Position++
	pkt.Set("CTS", uint32(0))

	switch pkt.Get("DataType").(byte) {
	case SPECIFIC_CONFIG:
		return me.parseSpecificConfig(pkt)
	case RAW_FRAME_DATA:
		return me.parseRawFrameData(pkt)
	default:
		err := fmt.Errorf("unrecognized AAC packet type: 0x%02X", pkt.Get("DataType").(byte))
		me.logger.Errorf("%v", err)
		return err
	}
}

func (me *AAC) parseSpecificConfig(pkt *av.Packet) error {
	if pkt.Left() < 2 {
		err := fmt.Errorf("data not enough while parsing AAC specific config")
		me.logger.Errorf("%v", err)
		return err
	}

	me.infoframe = pkt
	defer (func() {
		me.ctx.Codec = fmt.Sprintf("mp4a.40.%d", me.AudioObjectType)
		me.info.Codecs = append(me.info.Codecs, me.ctx.Codec)
	})()

	gb := new(utils.Golomb).Init(pkt.Payload[pkt.Position:])
	if gb == nil {
		err := fmt.Errorf("failed to init Golomb while parsing AAC specific config")
		me.logger.Errorf("%v", err)
		return err
	}

	me.AudioObjectType = uint8(gb.ReadBits(5))
	if me.AudioObjectType == AOT_ESCAPE {
		me.AudioObjectType = 32 + uint8(gb.ReadBits(6))
	}

	me.SamplingFrequencyIndex = uint8(gb.ReadBits(4))
	if me.SamplingFrequencyIndex == 0xF {
		me.SamplingFrequency = uint32(gb.ReadBits(24))
	} else {
		me.SamplingFrequency = SamplingFrequencys[me.SamplingFrequencyIndex]
	}
	me.info.SampleRate = me.SamplingFrequency

	me.ChannelConfiguration = uint8(gb.ReadBits(4))
	if me.ChannelConfiguration < 16 {
		me.Channels = Channels[me.ChannelConfiguration]
		me.info.Channels = uint32(me.Channels)
	}

	if me.AudioObjectType == AOT_SBR || (me.AudioObjectType == AOT_PS &&
		// Check for W6132 Annex YYYY draft MP3onMP4
		(gb.ShowBits(3)&0x03) == 0 && (gb.ShowBits(9)&0x3F) == 0) {
		me.ExtensionSamplingFrequencyIndex = uint8(gb.ReadBits(4))
		if me.ExtensionSamplingFrequencyIndex == 0xF {
			me.ExtensionSamplingFrequency = uint32(gb.ReadBits(24))
		} else {
			me.ExtensionSamplingFrequency = SamplingFrequencys[me.ExtensionSamplingFrequencyIndex]
		}
		me.info.SampleRate = me.ExtensionSamplingFrequency

		me.ExtensionAudioObjectType = uint8(gb.ReadBits(5))
		switch me.ExtensionAudioObjectType {
		case AOT_ESCAPE:
			me.ExtensionAudioObjectType = 32 + uint8(gb.ReadBits(6))
		case AOT_ER_BSAC:
			me.ExtensionChannelConfiguration = uint8(gb.ReadBits(4))
			me.Channels = Channels[me.ExtensionChannelConfiguration]
			me.info.Channels = uint32(me.ExtensionChannelConfiguration)
		}
	} else {
		me.ExtensionAudioObjectType = AOT_NULL
		me.ExtensionSamplingFrequency = 0
	}

	if me.AudioObjectType == AOT_ALS {
		gb.ShowBits(5)
		if gb.ShowBitsLong(24) != 0x00414C53 { // "\0ALS"
			gb.SkipBits(24)
		}

		err := me.parseConfigALS(gb)
		if err != nil {
			me.logger.Errorf("Failed to parse AAC config ALS")
			return err
		}
	}

	me.ctx.RefSampleDuration = me.info.Timescale * 1024 / me.SamplingFrequency

	// Force to AOT_SBR
	me.AudioObjectType = AOT_SBR
	me.ExtensionSamplingFrequencyIndex = me.SamplingFrequencyIndex
	if me.SamplingFrequencyIndex >= 6 {
		me.ExtensionSamplingFrequencyIndex -= 3
	} else if me.ChannelConfiguration == 1 { // Mono channel
		me.AudioObjectType = AOT_AAC_LC
	}

	if me.AudioObjectType == AOT_SBR {
		me.Config = []byte{
			me.AudioObjectType<<3 | me.SamplingFrequencyIndex>>1,
			me.SamplingFrequencyIndex<<7 | me.ChannelConfiguration<<3 | me.ExtensionSamplingFrequencyIndex>>1,
			me.ExtensionSamplingFrequencyIndex<<7 | 0x08,
			0x00,
		}
	} else {
		me.Config = []byte{
			me.AudioObjectType<<3 | me.SamplingFrequencyIndex>>1,
			me.SamplingFrequencyIndex<<7 | me.ChannelConfiguration<<3,
		}
	}
	return nil
}

func (me *AAC) parseRawFrameData(pkt *av.Packet) error {
	pkt.Set("DTS", me.info.Timestamp)
	pkt.Set("PTS", pkt.Get("DTS").(uint32))
	pkt.Set("Data", pkt.Payload[pkt.Position:])
	return nil
}

func (me *AAC) parseConfigALS(gb *utils.Golomb) error {
	if gb.Left() < 112 {
		return fmt.Errorf("data not enough while parsing ALS config")
	}

	if gb.ReadBitsLong(32) != 0x414C5300 { // "ALS\0"
		return fmt.Errorf("not ALS\\0")
	}

	// Override AudioSpecificConfig channel configuration and sample rate
	// which are buggy in old ALS conformance files
	me.SamplingFrequency = uint32(gb.ReadBitsLong(32))
	me.info.SampleRate = me.SamplingFrequency

	// Skip number of samples
	gb.SkipBits(32)

	// Read number of channels
	me.ChannelConfiguration = 0
	me.Channels = uint16(gb.ReadBits(16)) + 1
	me.info.Channels = uint32(me.Channels)
	return nil
}
