package m3u8

import (
	"fmt"
)

const (
	// DEFAULT_VERSION of me lib in use
	DEFAULT_VERSION uint = 7
)

// M3U8 defines a m3u8 file
type M3U8 struct {
	BasicTags
	MediaSegmentTags
	MediaPlaylistTags
	MasterPlaylistTags
	CommonPlaylistTags
}

// Init this class
func (me *M3U8) Init(version uint) *M3U8 {
	me.EXTM3U = true
	me.EXT_X_VERSION = version
	return me
}

// Marshal returns the INI encoding of M3U8
func (me *M3U8) Marshal() ([]byte, error) {
	tmp := "#EXTM3U\n"
	tmp += fmt.Sprintf("#EXT-X-VERSION:%d\n", me.EXT_X_VERSION)
	tmp += fmt.Sprintf("#EXT-X-TARGETDURATION:%d\n", me.EXT_X_TARGETDURATION)
	tmp += fmt.Sprintf("#EXT-X-MEDIA-SEQUENCE:%d\n", me.EXT_X_MEDIA_SEQUENCE)
	tmp += fmt.Sprintf("#EXT-X-MAP:URI=\"%s\"\n", me.EXT_X_MAP[0].URI)
	for _, inf := range me.EXTINF {
		tmp += fmt.Sprintf("#EXTINF:%.3f,\n%s\n", inf.Duration, inf.Title)
	}

	return []byte(tmp), nil
}

// BasicTags defines #EXTM3U, #EXT-X-VERSION
type BasicTags struct {
	EXTM3U        bool
	EXT_X_VERSION uint
}

// MediaSegmentTags of M3U8
type MediaSegmentTags struct {
	EXTINF                  []InfAttributes
	EXT_X_BYTERANGE         ByteRange
	EXT_X_DISCONTINUITY     bool
	EXT_X_KEY               []KeyAttributes
	EXT_X_MAP               []MapAttributes
	EXT_X_PROGRAM_DATE_TIME string // 2010-02-19T14:54:23.031+08:00
	EXT_X_DATERANGE         []DateRangeAttributes
}

// InfAttributes defines #EXTINF
type InfAttributes struct {
	Duration float64
	Title    string
}

// ByteRange defines #EXT-X-BYTERANGE
type ByteRange struct {
	N float64
	O float64
}

// KeyAttributes defines #EXT-X-KEY
type KeyAttributes struct {
	METHOD            string
	URI               string
	IV                string
	KEYFORMAT         string
	KEYFORMATVERSIONS string
}

// MapAttributes defines #EXT-X-MAP
type MapAttributes struct {
	URI       string
	BYTERANGE string
}

// DateRangeAttributes defines #EXT-X-DATERANGE
type DateRangeAttributes struct {
	ID               string
	CLASS            string
	START_DATE       string
	END_DATE         string
	DURATION         float64 // seconds
	PLANNED_DURATION float64 // seconds
	END_ON_NEXT      string  // 'YES'
}

// MediaPlaylistTags of M3U8
type MediaPlaylistTags struct {
	EXT_X_TARGETDURATION         uint
	EXT_X_MEDIA_SEQUENCE         uint
	EXT_X_DISCONTINUITY_SEQUENCE uint
	EXT_X_ENDLIST                bool
	EXT_X_PLAYLIST_TYPE          string // 'EVENT', 'VOD'
	EXT_X_I_FRAMES_ONLY          bool
}

// MasterPlaylistTags of M3U8
type MasterPlaylistTags struct {
	EXT_X_MEDIA              []MediaAttributes
	EXT_X_STREAM_INF         []StreamInfAttributes
	EXT_X_I_FRAME_STREAM_INF []IFrameStreamInfAttributes
	EXT_X_SESSION_DATA       []SessionDataAttributes
	EXT_X_SESSION_KEY        []KeyAttributes
}

// MediaAttributes defines #EXT-X-MEDIA
type MediaAttributes struct {
	TYPE            string
	URI             string
	GROUP_ID        string
	LANGUAGE        string
	ASSOC_LANGUAGE  string
	NAME            string
	DEFAULT         string // 'YES', 'NO'
	AUTOSELECT      string // 'YES', 'NO'
	FORCED          string // 'YES', 'NO'
	INSTREAM_ID     string // 'CC[1-4]', 'SERVICE[1-63]'
	CHARACTERISTICS string
	CHANNELS        string
}

// StreamInfAttributes defines #EXT-X-STREAM-INF
type StreamInfAttributes struct {
	BANDWIDTH         uint
	AVERAGE_BANDWIDTH uint
	CODECS            string
	RESOLUTION        string
	FRAME_RATE        float64
	HDCP_LEVEL        string // 'TYPE-0', 'NONE'
	AUDIO             string
	VIDEO             string
	SUBTITLES         string
	CLOSED_CAPTIONS   string
}

// IFrameStreamInfAttributes defines #EXT-X-I-FRAME-STREAM-INF
type IFrameStreamInfAttributes struct {
	URI string
}

// SessionDataAttributes defines #EXT-X-SESSION-DATA
type SessionDataAttributes struct {
	DATA_ID  string
	VALUE    string
	URI      string
	LANGUAGE string
}

// CommonPlaylistTags of M3U8
type CommonPlaylistTags struct {
	EXT_X_INDEPENDENT_SEGMENTS bool
	EXT_X_START                []StartAttributes
}

// StartAttributes defines #EXT-X-START
type StartAttributes struct {
	TIME_OFFSET float64
	PRECISE     string // 'YES', 'NO'
}
