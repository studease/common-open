package support

// Flag values for the audioCodecs property
const (
	SND_NONE    = 0x0001
	SND_ADPCM   = 0x0002
	SND_MP3     = 0x0004
	SND_INTEL   = 0x0008
	SND_UNUSED  = 0x0010
	SND_NELLY8  = 0x0020
	SND_NELLY   = 0x0040
	SND_G711A   = 0x0080
	SND_G711U   = 0x0100
	SND_NELLY16 = 0x0200
	SND_AAC     = 0x0400
	SND_SPEEX   = 0x0800
	SND_ALL     = 0x0FFF
)

// Flag values for the videoCodecs property
const (
	VID_UNUSED    = 0x0001
	VID_JPEG      = 0x0002
	VID_SORENSON  = 0x0004
	VID_HOMEBREW  = 0x0008
	VID_VP6       = 0x0010
	VID_VP6ALPHA  = 0x0020
	VID_HOMEBREWV = 0x0040
	VID_H264      = 0x0080
	VID_ALL       = 0x00FF
)
