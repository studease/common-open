package mp4

import (
	"encoding/binary"

	"github.com/studease/common/av"
	"github.com/studease/common/av/codec/aac"
	"github.com/studease/common/av/codec/avc"
)

var (
	tmpDREF = []byte{
		0x00, 0x00, 0x00, 0x00, // version(0) + flags
		0x00, 0x00, 0x00, 0x01, // entry_count
		0x00, 0x00, 0x00, 0x0C, // entry_size
		0x75, 0x72, 0x6C, 0x20, // type 'url '
		0x00, 0x00, 0x00, 0x01, // version(0) + flags
	}

	tmpFTYP = []byte{
		0x69, 0x73, 0x6F, 0x6D, // major_brand: isom
		0x0, 0x0, 0x0, 0x1, // minor_version: 0x01
		0x69, 0x73, 0x6F, 0x6D, // isom
		0x61, 0x76, 0x63, 0x31, // avc1
	}

	tmpVideoHDLR = []byte{
		0x00, 0x00, 0x00, 0x00, // version(0) + flags
		0x00, 0x00, 0x00, 0x00, // pre_defined
		0x76, 0x69, 0x64, 0x65, // handler_type: 'vide'
		0x00, 0x00, 0x00, 0x00, // reserved: 3 * 4 bytes
		0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00,
		0x56, 0x69, 0x64, 0x65,
		0x6F, 0x48, 0x61, 0x6E,
		0x64, 0x6C, 0x65, 0x72, 0x00, // name: VideoHandler
	}

	tmpAudioHDLR = []byte{
		0x00, 0x00, 0x00, 0x00, // version(0) + flags
		0x00, 0x00, 0x00, 0x00, // pre_defined
		0x73, 0x6F, 0x75, 0x6E, // handler_type: 'soun'
		0x00, 0x00, 0x00, 0x00, // reserved: 3 * 4 bytes
		0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00,
		0x53, 0x6F, 0x75, 0x6E,
		0x64, 0x48, 0x61, 0x6E,
		0x64, 0x6C, 0x65, 0x72, 0x00, // name: SoundHandler
	}

	tmpSTSD = []byte{
		0x00, 0x00, 0x00, 0x00, // version(0) + flags
		0x00, 0x00, 0x00, 0x01, // entry_count
	}

	tmpSTTS = []byte{
		0x00, 0x00, 0x00, 0x00, // version(0) + flags
		0x00, 0x00, 0x00, 0x00, // entry_count
	}

	tmpSTSC = tmpSTTS
	tmpSTCO = tmpSTTS

	tmpSTSZ = []byte{
		0x00, 0x00, 0x00, 0x00, // version(0) + flags
		0x00, 0x00, 0x00, 0x00, // sample_size
		0x00, 0x00, 0x00, 0x00, // sample_count
	}

	// video media header
	tmpVMHD = []byte{
		0x00, 0x00, 0x00, 0x01, // version(0) + flags
		0x00, 0x00, // graphicsmode: 2 bytes
		0x00, 0x00, 0x00, 0x00, // opcolor: 3 * 2 bytes
		0x00, 0x00,
	}

	// Sound media header
	tmpSMHD = []byte{
		0x00, 0x00, 0x00, 0x00, // version(0) + flags
		0x00, 0x00, 0x00, 0x00, // balance(2) + reserved(2)
	}
)

// Box creates a named MP4 box with the given data
func Box(typ string, args ...[]byte) []byte {
	i := 8
	n := 8

	for _, b := range args {
		n += len(b)
	}

	data := make([]byte, n)
	binary.BigEndian.PutUint32(data[0:4], uint32(n))
	copy(data[4:8], typ)

	for _, b := range args {
		copy(data[i:], b)
		i += len(b)
	}

	return data
}

// FTYP (File Type Box)
func FTYP() []byte {
	return Box(TYPE_FTYP, tmpFTYP)
}

// MOOV (Movie Box)
func MOOV(track *Track) []byte {
	mvhd := MVHD(track.Information())
	trak := TRAK(track)
	mvex := MVEX(track)
	return Box(TYPE_MOOV, mvhd, trak, mvex)
}

// MVHD (Movie Header Box)
func MVHD(info *av.Information) []byte {
	t := info.Timescale
	d := info.Duration

	return Box(TYPE_MVHD, []byte{
		0x00, 0x00, 0x00, 0x00, // version(0) + flags
		0x00, 0x00, 0x00, 0x00, // creation_time
		0x00, 0x00, 0x00, 0x00, // modification_time
		byte(t >> 24), byte(t >> 16), byte(t >> 8), byte(t), // timescale: 4 bytes
		byte(d >> 24), byte(d >> 16), byte(d >> 8), byte(d), // duration: 4 bytes
		0x00, 0x01, 0x00, 0x00, // Preferred rate: 1.0
		0x01, 0x00, 0x00, 0x00, // PreferredVolume(1.0, 2bytes) + reserved(2bytes)
		0x00, 0x00, 0x00, 0x00, // reserved: 4 + 4 bytes
		0x00, 0x00, 0x00, 0x00,
		0x00, 0x01, 0x00, 0x00, // ----begin composition matrix----
		0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00,
		0x00, 0x01, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00,
		0x40, 0x00, 0x00, 0x00, // ----end composition matrix----
		0x00, 0x00, 0x00, 0x00, // ----begin pre_defined 6 * 4 bytes----
		0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, // ----end pre_defined 6 * 4 bytes----
		0xFF, 0xFF, 0xFF, 0xFF, // next_track_ID
	})
}

// TRAK (Track Box)
func TRAK(track *Track) []byte {
	tkhd := TKHD(track)
	mdia := MDIA(track)
	return Box(TYPE_TRAK, tkhd, mdia)
}

// TKHD (Track Header Box)
func TKHD(track *Track) []byte {
	info := track.Information()
	ctx := track.Context().Basic()
	i := track.ID()
	w := info.Width
	h := info.Height
	d := ctx.Duration

	return Box(TYPE_TKHD, []byte{
		0x00, 0x00, 0x00, 0x07, // version(0) + flags
		0x00, 0x00, 0x00, 0x00, // creation_time
		0x00, 0x00, 0x00, 0x00, // modification_time
		byte(i >> 24), byte(i >> 16), byte(i >> 8), byte(i), // track_ID: 4 bytes
		0x00, 0x00, 0x00, 0x00, // reserved: 4 bytes
		byte(d >> 24), byte(d >> 16), byte(d >> 8), byte(d), // duration: 4 bytes
		0x00, 0x00, 0x00, 0x00, // reserved: 2 * 4 bytes
		0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, // layer(2bytes) + alternate_group(2bytes)
		0x00, 0x00, 0x00, 0x00, // volume(2bytes) + reserved(2bytes)
		0x00, 0x01, 0x00, 0x00, // ----begin composition matrix----
		0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00,
		0x00, 0x01, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00,
		0x40, 0x00, 0x00, 0x00, // ----end composition matrix----
		byte(w >> 8), byte(w), // width
		0x00, 0x00,
		byte(h >> 8), byte(h), // height
		0x00, 0x00,
	})
}

// MDIA (Media Box)
func MDIA(track *Track) []byte {
	mdhd := MDHD(track)
	hdlr := HDLR(track)
	minf := MINF(track)
	return Box(TYPE_MDIA, mdhd, hdlr, minf)
}

// MDHD (Media Header Box)
func MDHD(track *Track) []byte {
	info := track.Information()
	ctx := track.Context().Basic()
	t := info.Timescale
	d := ctx.Duration

	return Box(TYPE_MDHD, []byte{
		0x00, 0x00, 0x00, 0x00, // version(0) + flags
		0x00, 0x00, 0x00, 0x00, // creation_time
		0x00, 0x00, 0x00, 0x00, // modification_time
		byte(t >> 24), byte(t >> 16), byte(t >> 8), byte(t), // timescale: 4 bytes
		byte(d >> 24), byte(d >> 16), byte(d >> 8), byte(d), // duration: 4 bytes
		0x55, 0xC4, // language: und (undetermined)
		0x00, 0x00, // pre_defined = 0
	})
}

// HDLR (Handler Reference Box)
func HDLR(track *Track) []byte {
	var (
		data []byte
	)

	if track.Kind() == av.KIND_AUDIO {
		data = tmpAudioHDLR
	} else {
		data = tmpVideoHDLR
	}

	return Box(TYPE_HDLR, data)
}

// MINF (Media Information Box)
func MINF(track *Track) []byte {
	var (
		xmhd []byte
	)

	if track.Kind() == av.KIND_AUDIO {
		xmhd = Box(TYPE_SMHD, tmpSMHD)
	} else {
		xmhd = Box(TYPE_VMHD, tmpVMHD)
	}

	dinf := DINF(track)
	stbl := STBL(track)

	return Box(TYPE_MINF, xmhd, dinf, stbl)
}

// DINF (Data Information Box)
func DINF(track *Track) []byte {
	dref := Box(TYPE_DREF, tmpDREF)
	return Box(TYPE_DINF, dref)
}

// STBL (Sample Table Box)
func STBL(track *Track) []byte {
	stsd := STSD(track)
	stts := Box(TYPE_STTS, tmpSTTS) // Decoding Time to Sample
	stsc := Box(TYPE_STSC, tmpSTSC) // Sample To Chunk
	stsz := Box(TYPE_STSZ, tmpSTSZ) // Sample Size
	stco := Box(TYPE_STCO, tmpSTCO) // Chunk Offset
	return Box(TYPE_STBL, stsd, stts, stsc, stsz, stco)
}

// STSD (Sample Description Box)
func STSD(track *Track) []byte {
	var (
		data []byte
	)

	if track.Kind() == av.KIND_AUDIO {
		data = MP4A(track)
	} else {
		data = AVC1(track)
	}

	return Box(TYPE_STSD, tmpSTSD, data)
}

// MP4A (MPEG-4 Audio Box)
func MP4A(track *Track) []byte {
	ctx := track.Context().(*aac.Context)
	n := ctx.ChannelConfiguration
	r := ctx.SamplingFrequency

	data := []byte{
		0x00, 0x00, 0x00, 0x00, // reserved(4)
		0x00, 0x00, 0x00, 0x01, // reserved(2) + data_reference_index(2)
		0x00, 0x00, 0x00, 0x00, // reserved: 2 * 4 bytes
		0x00, 0x00, 0x00, 0x00,
		0x00, n, // channelCount(2)
		0x00, 0x10, // sampleSize(2)
		0x00, 0x00, 0x00, 0x00, // reserved(4)
		byte(r >> 8), byte(r), // Audio Sample Rate
		0x00, 0x00,
	}

	esds := ESDS(track)

	return Box(TYPE_MP4A, data, esds)
}

// ESDS (Element Stream Descriptors Box)
func ESDS(track *Track) []byte {
	ctx := track.Context().(*aac.Context)
	n := byte(len(ctx.Config))

	data := []byte{
		0x00, 0x00, 0x00, 0x00, // version 0 + flags

		0x03,       // descriptor_type
		0x17 + n,   // length3
		0x00, 0x01, // es_id
		0x00, // stream_priority

		0x04,             // descriptor_type
		0x0F + n,         // length
		0x40,             // codec: mpeg4_audio
		0x15,             // stream_type: Audio
		0x00, 0x00, 0x00, // buffer_size
		0x00, 0x00, 0x00, 0x00, // maxBitrate
		0x00, 0x00, 0x00, 0x00, // avgBitrate

		0x05, // descriptor_type
		n,
	}

	return Box(TYPE_ESDS, data, ctx.Config, []byte{
		0x06, 0x01, 0x02, // GASpecificConfig
	})
}

// AVC1 (AVC Box)
func AVC1(track *Track) []byte {
	info := track.Information()
	ctx := track.Context().(*avc.Context)
	w := info.CodecWidth
	h := info.CodecHeight

	data := []byte{
		0x00, 0x00, 0x00, 0x00, // reserved(4)
		0x00, 0x00, 0x00, 0x01, // reserved(2) + data_reference_index(2)
		0x00, 0x00, 0x00, 0x00, // pre_defined(2) + reserved(2)
		0x00, 0x00, 0x00, 0x00, // pre_defined: 3 * 4 bytes
		0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00,
		byte(w >> 8), byte(w), // width: 2 bytes
		byte(h >> 8), byte(h), // height: 2 bytes
		0x00, 0x48, 0x00, 0x00, // horizresolution: 4 bytes
		0x00, 0x48, 0x00, 0x00, // vertresolution: 4 bytes
		0x00, 0x00, 0x00, 0x00, // reserved: 4 bytes
		0x00, 0x01, // frame_count
		0x0A,                   // strlen
		0x78, 0x71, 0x71, 0x2F, // compressorname: 32 bytes
		0x66, 0x6C, 0x76, 0x2E,
		0x6A, 0x73, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00,
		0x00, 0x18, // depth
		0xFF, 0xFF, // pre_defined = -1
	}

	avcc := Box(TYPE_AVCC, ctx.AVCC)

	return Box(TYPE_AVC1, data, avcc)
}

// MVEX (Movie Extends Box)
func MVEX(track *Track) []byte {
	trex := TREX(track)
	return Box(TYPE_MVEX, trex)
}

// TREX (Track Extends Box)
func TREX(track *Track) []byte {
	i := track.ID()

	data := []byte{
		0x00, 0x00, 0x00, 0x00, // version(0) + flags
		byte(i >> 24), byte(i >> 16), byte(i >> 8), byte(i), // track_ID
		0x00, 0x00, 0x00, 0x01, // default_sample_description_index
		0x00, 0x00, 0x00, 0x00, // default_sample_duration
		0x00, 0x00, 0x00, 0x00, // default_sample_size
		0x00, 0x01, 0x00, 0x01, // default_sample_flags
	}

	return Box(TYPE_TREX, data)
}

// MOOF (Movie Fragment Box)
func MOOF(track *Track) []byte {
	mfhd := MFHD(track)
	traf := TRAF(track)
	return Box(TYPE_MOOF, mfhd, traf)
}

// MFHD (Movie Fragment Header Box)
func MFHD(track *Track) []byte {
	n := track.SequenceNumber

	data := []byte{
		0x00, 0x00, 0x00, 0x00,
		byte(n >> 24), byte(n >> 16), byte(n >> 8), byte(n), // sequence_number: int32
	}

	return Box(TYPE_MFHD, data)
}

// TRAF (Track Fragment Box)
func TRAF(track *Track) []byte {
	ctx := track.Context().Basic()
	i := track.ID()
	d := ctx.DTS

	// Track Fragment Header Box
	tfhd := Box(TYPE_TFHD, []byte{
		0x00, 0x00, 0x00, 0x00, // version(0) & flags
		byte(i >> 24), byte(i >> 16), byte(i >> 8), byte(i), // track_ID
	})

	// Track Fragment Decode Time
	tfdt := Box(TYPE_TFDT, []byte{
		0x00, 0x00, 0x00, 0x00, // version(0) & flags
		byte(d >> 24), byte(d >> 16), byte(d >> 8), byte(d), // baseMediaDecodeTime: int32
	})

	trun := TRUN(track)
	sdtp := SDTP(track)

	return Box(TYPE_TRAF, tfhd, tfdt, trun, sdtp)
}

// TRUN (Track Fragment Run Box)
func TRUN(track *Track) []byte {
	ctx := track.Context().Basic()
	s := len(ctx.Data)
	d := ctx.Duration
	f := ctx.Flags
	t := ctx.CTS

	data := []byte{
		0x00, 0x00, 0x0F, 0x01, // version(0) & flags
		0x00, 0x00, 0x00, 0x01, // sample_count
		0x00, 0x00, 0x00, 0x79, // data_offset
		byte(d >> 24), byte(d >> 16), byte(d >> 8), byte(d), // sample_duration
		byte(s >> 24), byte(s >> 16), byte(s >> 8), byte(s), // sample_size
		(f.IsLeading << 2) | f.SampleDependsOn,
		(f.SampleIsDependedOn << 6) | (f.SampleHasRedundancy << 4) | f.IsNonSync, // sample_flags
		0x00, 0x00, // sample_degradation_priority
		byte(t >> 24), byte(t >> 16), byte(t >> 8), byte(t), // sample_composition_time_offset
	}

	return Box(TYPE_TRUN, data)
}

// SDTP (Sample Dependency Type Box)
func SDTP(track *Track) []byte {
	ctx := track.Context().Basic()
	f := ctx.Flags

	data := []byte{
		0x00, 0x00, 0x00, 0x00, // version(0) + flags
		f.IsLeading<<6 | // is_leading            (2 bits)
			f.SampleDependsOn<<4 | // sample_depends_on     (2 bits)
			f.SampleIsDependedOn<<2 | // sample_is_depended_on (2 bits)
			f.SampleHasRedundancy, // sample_has_redundancy (2 bits)
	}

	return Box(TYPE_SDTP, data)
}

// MDAT (Media Data Box)
func MDAT(data []byte) []byte {
	return Box(TYPE_MDAT, data)
}

// Merge the given boxes into one
func Merge(args ...[]byte) []byte {
	i := 0
	n := 0

	for _, b := range args {
		n += len(b)
	}

	data := make([]byte, n)

	for _, b := range args {
		copy(data[i:], b)
		i += len(b)
	}

	return data
}
