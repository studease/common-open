package aac

import (
	"fmt"

	"github.com/studease/common/av"
	"github.com/studease/common/av/codec"
	"github.com/studease/common/av/utils"
	"github.com/studease/common/log"
)

// AOT types
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

// Data types
const (
	SPECIFIC_CONFIG = 0x00
	RAW_FRAME_DATA  = 0x01
)

var (
	// Rates in FLV
	Rates = [4]int{5500, 11025, 22050, 44100}
	// SamplingFrequencys which AAC supported
	SamplingFrequencys = [16]uint32{96000, 88200, 64000, 48000, 44100, 32000, 24000, 22050, 16000, 12000, 11025, 8000, 7350}
	// Channels for quick mapping
	Channels = [8]uint16{0, 1, 2, 3, 4, 5, 6, 8}
)

func init() {
	codec.Register(codec.AAC, Context{})
}

// Context implements IMediaContext
type Context struct {
	av.Context

	logger log.ILogger

	// Specific Config
	Golomb                          utils.Golomb
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

// Init this class
func (me *Context) Init(info *av.Information, logger log.ILogger) av.IMediaContext {
	me.Context.Init(info)
	me.logger = logger
	me.MimeType = "audio/mp4"
	me.Flags.IsLeading = 0
	me.Flags.SampleDependsOn = 1
	me.Flags.SampleIsDependedOn = 0
	me.Flags.SampleHasRedundancy = 0
	me.Flags.IsNonSync = 0
	return me
}

// Codec returns the codec ID of this context
func (me *Context) Codec() av.Codec {
	return codec.AAC
}

// Parse an AAC packet
func (me *Context) Parse(p *av.Packet) error {
	if len(p.Payload) < 2 {
		err := fmt.Errorf("data not enough while parsing AAC packet")
		me.logger.Debugf(2, "%v", err)
		return err
	}

	p.Context = me
	info := me.Information()
	info.Timestamp += p.Timestamp

	i := 0

	tmp := p.Payload[i]
	me.Format = tmp & 0xF0
	me.SampleRate = (tmp >> 2) & 0x03
	me.SampleSize = (tmp >> 1) & 0x01
	me.SampleType = tmp & 0x01
	i++

	me.DataType = p.Payload[i]
	i++

	switch me.DataType {
	case SPECIFIC_CONFIG:
		return me.parseSpecificConfig(p.Timestamp, p.Payload[i:])

	case RAW_FRAME_DATA:
		return me.parseRawFrameData(p.Timestamp, p.Payload[i:])

	default:
		err := fmt.Errorf("unrecognized AAC packet type: 0x%02X", me.DataType)
		me.logger.Debugf(2, "%v", err)
		return err
	}
}

func (me *Context) parseSpecificConfig(timestamp uint32, data []byte) error {
	if len(data) < 2 {
		err := fmt.Errorf("data not enough while parsing AAC specific config")
		me.logger.Debugf(2, "%v", err)
		return err
	}

	defer (func() {
		me.Codecs = fmt.Sprintf("mp4a.40.%d", me.AudioObjectType)
	})()

	info := me.Information()

	gb := me.Golomb.Init(data)
	if gb == nil {
		err := fmt.Errorf("failed to init Golomb")
		me.logger.Debugf(2, "%v", err)
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

	me.ChannelConfiguration = uint8(gb.ReadBits(4))
	if me.ChannelConfiguration < 16 {
		me.Channels = Channels[me.ChannelConfiguration]
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

		me.ExtensionAudioObjectType = uint8(gb.ReadBits(5))
		switch me.ExtensionAudioObjectType {
		case AOT_ESCAPE:
			me.ExtensionAudioObjectType = 32 + uint8(gb.ReadBits(6))

		case AOT_ER_BSAC:
			me.ExtensionChannelConfiguration = uint8(gb.ReadBits(4))
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
			me.logger.Debugf(2, "Failed to parse AAC config ALS")
			return err
		}
	}

	info.RefSampleDuration = info.Timescale * 1024 / me.SamplingFrequency

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

func (me *Context) parseRawFrameData(timestamp uint32, data []byte) error {
	info := me.Information()

	me.DTS = info.TimeBase + info.Timestamp
	me.PTS = me.DTS
	me.Data = data
	return nil
}

func (me *Context) parseConfigALS(gb *utils.Golomb) error {
	if gb.Left() < 112 {
		return fmt.Errorf("data not enough while parsing ALS config")
	}

	if gb.ReadBitsLong(32) != 0x414C5300 { // "ALS\0"
		return fmt.Errorf("not ALS\\0")
	}

	// Override AudioSpecificConfig channel configuration and sample rate
	// which are buggy in old ALS conformance files
	me.SamplingFrequency = uint32(gb.ReadBitsLong(32))

	// Skip number of samples
	gb.SkipBits(32)

	// Read number of channels
	me.ChannelConfiguration = 0
	me.Channels = uint16(gb.ReadBits(16)) + 1

	return nil
}
