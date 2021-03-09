package m3u8

import (
	"bytes"
	"fmt"
)

const (
	// DEFAULT_VERSION of me lib in use
	DEFAULT_VERSION uint = 9
)

// MediaPlaylist of m3u8
type MediaPlaylist struct {
	BasicTags
	MediaPlaylistTags
	MediaMetadataTags
	MediaSegments []MediaSegmentTags
	chunks        int
	buffer        bytes.Buffer
}

// Init this class
func (me *MediaPlaylist) Init(version uint, chunks int) *MediaPlaylist {
	me.EXTM3U = true
	me.EXT_X_VERSION = version
	me.chunks = chunks
	return me
}

// Marshal returns the INI encoding of m3u8
func (me *MediaPlaylist) Marshal() ([]byte, error) {
	me.buffer.Reset()

	// basic tags
	me.buffer.WriteString("#EXTM3U\n")
	me.buffer.WriteString(fmt.Sprintf("#EXT-X-VERSION:%d\n", me.EXT_X_VERSION))

	// media playlist tags
	if me.EXT_X_INDEPENDENT_SEGMENTS {
		me.buffer.WriteString("#EXT-X-INDEPENDENT-SEGMENTS\n")
	}
	if me.EXT_X_START.TIME_OFFSET > 0 {
		me.buffer.WriteString("#EXT-X-START:")
		me.buffer.WriteString(fmt.Sprintf("TIME-OFFSET=%.5f", me.EXT_X_START.TIME_OFFSET))
		if me.EXT_X_START.PRECISE != "" {
			me.buffer.WriteString(fmt.Sprintf(",PRECISE:%s", me.EXT_X_START.PRECISE))
		}
		me.buffer.WriteString("\n")
	}
	if me.EXT_X_DEFINE.VALUE != "" {
		me.buffer.WriteString("#EXT-X-DEFINE:")
		if me.EXT_X_DEFINE.NAME != "" {
			me.buffer.WriteString(fmt.Sprintf("NAME=%s", me.EXT_X_DEFINE.NAME))
		}
		if me.EXT_X_DEFINE.IMPORT != "" {
			me.buffer.WriteString(fmt.Sprintf("IMPORT=%s", me.EXT_X_DEFINE.IMPORT))
		}
		me.buffer.WriteString(fmt.Sprintf(",VALUE=%s\n", me.EXT_X_DEFINE.VALUE))
	}
	me.buffer.WriteString(fmt.Sprintf("#EXT-X-TARGETDURATION:%d\n", me.EXT_X_TARGETDURATION))
	if me.EXT_X_DISCONTINUITY_SEQUENCE != 0 {
		me.buffer.WriteString(fmt.Sprintf("#EXT-X-DISCONTINUITY-SEQUENCE:%d\n", me.EXT_X_DISCONTINUITY_SEQUENCE))
	}
	if me.EXT_X_PLAYLIST_TYPE != "" {
		me.buffer.WriteString(fmt.Sprintf("#EXT-X-PLAYLIST-TYPE:%s\n", me.EXT_X_PLAYLIST_TYPE))
	}
	if me.EXT_X_I_FRAMES_ONLY {
		me.buffer.WriteString("#EXT-X-I-FRAMES-ONLY\n")
	}
	if me.EXT_X_PART_INF.PART_TARGET > 0 {
		me.buffer.WriteString("#EXT-X-PART-INF:")
		me.buffer.WriteString(fmt.Sprintf("PART-TARGET=%.5f", me.EXT_X_PART_INF.PART_TARGET))
		me.buffer.WriteString("\n")

		me.buffer.WriteString("#EXT-X-SERVER-CONTROL:")
		me.buffer.WriteString(fmt.Sprintf("PART-HOLD-BACK=%.1f", me.EXT_X_SERVER_CONTROL.PART_HOLD_BACK))
		if me.EXT_X_SERVER_CONTROL.CAN_SKIP_UNTIL > 0 {
			me.buffer.WriteString(fmt.Sprintf(",CAN-SKIP-UNTIL=%.1f", me.EXT_X_SERVER_CONTROL.CAN_SKIP_UNTIL))
		}
		if me.EXT_X_SERVER_CONTROL.CAN_SKIP_DATERANGES != "" {
			me.buffer.WriteString(fmt.Sprintf(",CAN-SKIP-DATERANGES=%s", me.EXT_X_SERVER_CONTROL.CAN_SKIP_DATERANGES))
		}
		if me.EXT_X_SERVER_CONTROL.HOLD_BACK > 0 {
			me.buffer.WriteString(fmt.Sprintf(",HOLD-BACK=%.1f", me.EXT_X_SERVER_CONTROL.HOLD_BACK))
		}
		if me.EXT_X_SERVER_CONTROL.CAN_BLOCK_RELOAD != "" {
			me.buffer.WriteString(fmt.Sprintf(",CAN-BLOCK-RELOAD=%s", me.EXT_X_SERVER_CONTROL.CAN_BLOCK_RELOAD))
		}
		me.buffer.WriteString("\n")
	}

	// media metadata tags
	if me.EXT_X_DATERANGE.ID != "" {
		me.buffer.WriteString("#EXT-X-DATERANGE:")
		me.buffer.WriteString(fmt.Sprintf("ID=%s", me.EXT_X_DATERANGE.ID))
		if me.EXT_X_DATERANGE.CLASS != "" {
			me.buffer.WriteString(fmt.Sprintf(",CLASS=%s", me.EXT_X_DATERANGE.CLASS))
		}
		me.buffer.WriteString(fmt.Sprintf(",START-DATE=%s", me.EXT_X_DATERANGE.START_DATE))
		if me.EXT_X_DATERANGE.END_DATE != "" {
			me.buffer.WriteString(fmt.Sprintf(",END-DATE=%s", me.EXT_X_DATERANGE.END_DATE))
		}
		if me.EXT_X_DATERANGE.DURATION > 0 {
			me.buffer.WriteString(fmt.Sprintf(",DURATION=%.3f", me.EXT_X_DATERANGE.DURATION))
		}
		if me.EXT_X_DATERANGE.PLANNED_DURATION > 0 {
			me.buffer.WriteString(fmt.Sprintf(",PLANNED-DURATION=%.3f", me.EXT_X_DATERANGE.PLANNED_DURATION))
		}
		if me.EXT_X_DATERANGE.END_ON_NEXT != "" {
			me.buffer.WriteString(fmt.Sprintf(",END-ON-NEXT=%s", me.EXT_X_DATERANGE.END_ON_NEXT))
		}
		me.buffer.WriteString("\n")
	}
	if me.EXT_X_MEDIA_SEQUENCE != 0 {
		me.buffer.WriteString(fmt.Sprintf("#EXT-X-MEDIA-SEQUENCE:%d\n", me.EXT_X_MEDIA_SEQUENCE))
	}
	if me.EXT_X_SKIP.SKIPPED_SEGMENTS != "" {
		me.buffer.WriteString("#EXT-X-SKIP:")
		me.buffer.WriteString(fmt.Sprintf("SKIPPED-SEGMENTS=%s", me.EXT_X_SKIP.SKIPPED_SEGMENTS))
		if me.EXT_X_SKIP.RECENTLY_REMOVED_DATERANGES != "" {
			me.buffer.WriteString(fmt.Sprintf(",RECENTLY-REMOVED-DATERANGES=%s", me.EXT_X_SKIP.RECENTLY_REMOVED_DATERANGES))
		}
		me.buffer.WriteString("\n")
	}

	// media segment tags
	for _, item := range me.MediaSegments {
		if item.EXT_X_KEY.METHOD != "" && item.EXT_X_KEY.URI != "" {
			me.buffer.WriteString("#EXT-X-KEY:")
			me.buffer.WriteString(fmt.Sprintf("METHOD=%s", item.EXT_X_KEY.METHOD))
			me.buffer.WriteString(fmt.Sprintf(",URI=\"%s\"", item.EXT_X_KEY.URI))
			if item.EXT_X_KEY.IV != "" {
				me.buffer.WriteString(fmt.Sprintf(",IV=%s", item.EXT_X_KEY.IV))
			}
			if item.EXT_X_KEY.KEYFORMAT != "" {
				me.buffer.WriteString(fmt.Sprintf(",KEYFORMAT=%s", item.EXT_X_KEY.KEYFORMAT))
			}
			if item.EXT_X_KEY.KEYFORMATVERSIONS != "" {
				me.buffer.WriteString(fmt.Sprintf(",KEYFORMATVERSIONS=%s", item.EXT_X_KEY.KEYFORMATVERSIONS))
			}
			me.buffer.WriteString("\n")
		}
		if item.EXT_X_DISCONTINUITY {
			me.buffer.WriteString("#EXT-X-DISCONTINUITY\n")
		}
		if item.EXT_X_PROGRAM_DATE_TIME != "" {
			me.buffer.WriteString(fmt.Sprintf("#EXT-X-PROGRAM-DATE-TIME:%s\n", item.EXT_X_PROGRAM_DATE_TIME))
		}
		if item.EXT_X_MAP.URI != "" {
			me.buffer.WriteString("#EXT-X-MAP:")
			me.buffer.WriteString(fmt.Sprintf("URI=\"%s\"", item.EXT_X_MAP.URI))
			if item.EXT_X_MAP.BYTERANGE != "" {
				me.buffer.WriteString(fmt.Sprintf(",BYTERANGE=%s", item.EXT_X_MAP.BYTERANGE))
			}
			me.buffer.WriteString("\n")
		}
		if item.EXT_X_BYTERANGE.N > 0 {
			me.buffer.WriteString(fmt.Sprintf("#EXT-X-BYTERANGE:%d", item.EXT_X_BYTERANGE.N))
			if item.EXT_X_BYTERANGE.O > 0 {
				me.buffer.WriteString(fmt.Sprintf("@%d", item.EXT_X_BYTERANGE.O))
			}
			me.buffer.WriteString("\n")
		}
		if item.EXT_X_GAP {
			me.buffer.WriteString("#EXT-X-GAP\n")
		}
		if item.EXT_X_BITRATE > 0 {
			me.buffer.WriteString(fmt.Sprintf("#EXT-X-BITRATE:%d\n", item.EXT_X_BITRATE))
		}

		for _, part := range item.EXT_X_PART {
			if part.DURATION == 0 || part.URI == "" {
				continue
			}
			me.buffer.WriteString(fmt.Sprintf("#EXT-X-PART:DURATION=%.5f,URI=\"%s\"", part.DURATION, part.URI))
			if part.INDEPENDENT != "" {
				me.buffer.WriteString(fmt.Sprintf(",INDEPENDENT=%s", part.INDEPENDENT))
			}
			if part.BYTERANGE.N > 0 {
				me.buffer.WriteString(fmt.Sprintf(",BYTERANGE=%d", part.BYTERANGE.N))
				if part.BYTERANGE.O > 0 {
					me.buffer.WriteString(fmt.Sprintf("@%d", part.BYTERANGE.O))
				}
			}
			if part.GAP != "" {
				me.buffer.WriteString(fmt.Sprintf(",GAP=%s", part.GAP))
			}
			me.buffer.WriteString("\n")
		}
		if len(item.EXT_X_PART) >= me.chunks {
			me.buffer.WriteString(fmt.Sprintf("#EXTINF:%.5f,%s\n%s\n", item.EXTINF.Duration, item.EXTINF.Title, item.URI))
		}
	}

	if !me.EXT_X_ENDLIST && me.EXT_X_PRELOAD_HINT.TYPE != "" && me.EXT_X_PRELOAD_HINT.URI != "" {
		me.buffer.WriteString("#EXT-X-PRELOAD-HINT:")
		me.buffer.WriteString(fmt.Sprintf("TYPE=%s", me.EXT_X_PRELOAD_HINT.TYPE))
		me.buffer.WriteString(fmt.Sprintf(",URI=\"%s\"", me.EXT_X_PRELOAD_HINT.URI))
		if me.EXT_X_PRELOAD_HINT.BYTERANGE_START != 0 {
			me.buffer.WriteString(fmt.Sprintf(",BYTERANGE-START=%d", me.EXT_X_PRELOAD_HINT.BYTERANGE_START))
		}
		if me.EXT_X_PRELOAD_HINT.BYTERANGE_LENGTH != 0 {
			me.buffer.WriteString(fmt.Sprintf(",BYTERANGE-LENGTH=%d", me.EXT_X_PRELOAD_HINT.BYTERANGE_LENGTH))
		}
		me.buffer.WriteString("\n")
	}

	for i, item := range me.EXT_X_RENDITION_REPORT {
		if item.URI == "" {
			continue
		}
		if i == 0 {
			me.buffer.WriteString("\n")
		}
		me.buffer.WriteString("#EXT-X-RENDITION-REPORT:")
		me.buffer.WriteString(fmt.Sprintf("URI=\"%s\"", item.URI))
		me.buffer.WriteString(fmt.Sprintf(",LAST-MSN=%d", item.LAST_MSN))
		me.buffer.WriteString(fmt.Sprintf(",LAST-PART=%d\n", item.LAST_PART))
	}

	if me.EXT_X_ENDLIST {
		me.buffer.WriteString("#EXT-X-ENDLIST\n")
	}

	return me.buffer.Bytes(), nil
}

// BasicTags of m3u8
type BasicTags struct {
	EXTM3U        bool
	EXT_X_VERSION uint
}

// MediaOrMasterPlaylistTags of m3u8
type MediaOrMasterPlaylistTags struct {
	EXT_X_INDEPENDENT_SEGMENTS bool
	EXT_X_START                StartAttributes
	EXT_X_DEFINE               DefineAttributes
}

// StartAttributes defined for #EXT-X-START
type StartAttributes struct {
	TIME_OFFSET float64
	PRECISE     string // 'YES', 'NO'
}

// DefineAttributes defined for #EXT-X-DEFINE
type DefineAttributes struct {
	NAME   string
	VALUE  string
	IMPORT string
}

// MediaPlaylistTags of m3u8
type MediaPlaylistTags struct {
	MediaOrMasterPlaylistTags
	EXT_X_TARGETDURATION         uint
	EXT_X_MEDIA_SEQUENCE         uint
	EXT_X_DISCONTINUITY_SEQUENCE uint
	EXT_X_ENDLIST                bool
	EXT_X_PLAYLIST_TYPE          string // 'EVENT', 'VOD'
	EXT_X_I_FRAMES_ONLY          bool
	EXT_X_PART_INF               PartInfAttributes
	EXT_X_SERVER_CONTROL         ServerControlAttributes
}

// PartInfAttributes defined for #EXT-X-PART-INF
type PartInfAttributes struct {
	PART_TARGET float64
}

// ServerControlAttributes defined for #EXT-X-SERVER-CONTROL
type ServerControlAttributes struct {
	CAN_SKIP_UNTIL      float64
	CAN_SKIP_DATERANGES string // 'YES'
	HOLD_BACK           float64
	PART_HOLD_BACK      float64
	CAN_BLOCK_RELOAD    string // 'YES'
}

// MediaMetadataTags of m3u8
type MediaMetadataTags struct {
	EXT_X_DATERANGE        DateRangeAttributes
	EXT_X_SKIP             SkipAttributes
	EXT_X_PRELOAD_HINT     PreloadHintAttributes
	EXT_X_RENDITION_REPORT []RenditionReportAttributes
}

// DateRangeAttributes defined for #EXT-X-DATERANGE
type DateRangeAttributes struct {
	ID                  string
	CLASS               string
	START_DATE          string
	END_DATE            string
	DURATION            float64
	PLANNED_DURATION    float64
	X_CLIENT_ATTRIBUTES []XClientAttribute
	END_ON_NEXT         string // 'YES'
}

// XClientAttribute defined for X-<client-attribute>
type XClientAttribute struct {
	Key   string
	Value string
}

// SkipAttributes defined for #EXT-X-SKIP
type SkipAttributes struct {
	SKIPPED_SEGMENTS            string
	RECENTLY_REMOVED_DATERANGES string
}

// PreloadHintAttributes defined for #EXT-X-PRELOAD-HINT
type PreloadHintAttributes struct {
	TYPE             string // 'PART', 'MAP'
	URI              string
	BYTERANGE_START  uint
	BYTERANGE_LENGTH uint
}

// RenditionReportAttributes defined for #EXT-X-RENDITION-REPORT
type RenditionReportAttributes struct {
	URI       string
	LAST_MSN  uint
	LAST_PART uint
}

// MediaSegmentTags of m3u8
type MediaSegmentTags struct {
	EXTINF                  InfAttributes
	EXT_X_BYTERANGE         ByteRange
	EXT_X_DISCONTINUITY     bool
	EXT_X_KEY               KeyAttributes
	EXT_X_MAP               MapAttributes
	EXT_X_PROGRAM_DATE_TIME string // 2010-02-19T14:54:23.031+08:00
	EXT_X_GAP               bool
	EXT_X_BITRATE           uint
	EXT_X_PART              []PartAttributes
	URI                     string
}

// InfAttributes defined for #EXTINF
type InfAttributes struct {
	Duration float64
	Title    string
}

// ByteRange defined for #EXT-X-BYTERANGE
type ByteRange struct {
	N uint
	O uint
}

// KeyAttributes defined for #EXT-X-KEY
type KeyAttributes struct {
	METHOD            string // 'NONE', 'AES-128', 'SAMPLE-AES'
	URI               string
	IV                string
	KEYFORMAT         string
	KEYFORMATVERSIONS string
}

// MapAttributes defined for #EXT-X-MAP
type MapAttributes struct {
	URI       string
	BYTERANGE string
}

// PartAttributes defined for #EXT-X-PART
type PartAttributes struct {
	URI         string
	DURATION    float64
	INDEPENDENT string // 'YES'
	BYTERANGE   ByteRange
	GAP         string // 'YES'
}

// MasterPlaylist of m3u8
type MasterPlaylist struct {
	BasicTags
	MasterPlaylistTags
	buffer bytes.Buffer
}

// Init this class
func (me *MasterPlaylist) Init() *MasterPlaylist {
	me.EXTM3U = true
	return me
}

// Marshal returns the INI encoding
func (me *MasterPlaylist) Marshal() ([]byte, error) {
	me.buffer.Reset()

	// basic tags
	me.buffer.WriteString("#EXTM3U\n")

	// master playlist tags
	if me.EXT_X_INDEPENDENT_SEGMENTS {
		me.buffer.WriteString("#EXT-X-INDEPENDENT-SEGMENTS\n")
	}
	if me.EXT_X_START.TIME_OFFSET > 0 {
		me.buffer.WriteString("#EXT-X-START:")
		me.buffer.WriteString(fmt.Sprintf("TIME-OFFSET=%.5f", me.EXT_X_START.TIME_OFFSET))
		if me.EXT_X_START.PRECISE != "" {
			me.buffer.WriteString(fmt.Sprintf(",PRECISE:%s", me.EXT_X_START.PRECISE))
		}
		me.buffer.WriteString("\n")
	}
	if me.EXT_X_DEFINE.VALUE != "" {
		me.buffer.WriteString("#EXT-X-DEFINE:")
		if me.EXT_X_DEFINE.NAME != "" {
			me.buffer.WriteString(fmt.Sprintf("NAME=%s", me.EXT_X_DEFINE.NAME))
		}
		if me.EXT_X_DEFINE.IMPORT != "" {
			me.buffer.WriteString(fmt.Sprintf("IMPORT=%s", me.EXT_X_DEFINE.IMPORT))
		}
		me.buffer.WriteString(fmt.Sprintf(",VALUE=%s\n", me.EXT_X_DEFINE.VALUE))
	}

	for i, item := range me.EXT_X_MEDIA {
		if item.TYPE == "" || item.GROUP_ID == "" || item.NAME == "" {
			continue
		}
		if i == 0 {
			me.buffer.WriteString("\n")
		}
		me.buffer.WriteString("#EXT-X-MEDIA:")
		me.buffer.WriteString(fmt.Sprintf("TYPE=%s", item.TYPE))
		me.buffer.WriteString(fmt.Sprintf(",GROUP-ID=\"%s\"", item.GROUP_ID))
		me.buffer.WriteString(fmt.Sprintf(",NAME=\"%s\"", item.NAME))
		if item.LANGUAGE != "" {
			me.buffer.WriteString(fmt.Sprintf(",LANGUAGE=\"%s\"", item.LANGUAGE))
		}
		if item.ASSOC_LANGUAGE != "" {
			me.buffer.WriteString(fmt.Sprintf(",ASSOC-LANGUAGE=\"%s\"", item.ASSOC_LANGUAGE))
		}
		if item.DEFAULT != "" {
			me.buffer.WriteString(fmt.Sprintf(",DEFAULT=%s", item.DEFAULT))
		}
		if item.AUTOSELECT != "" {
			me.buffer.WriteString(fmt.Sprintf(",AUTOSELECT=%s", item.AUTOSELECT))
		}
		if item.FORCED != "" {
			me.buffer.WriteString(fmt.Sprintf(",FORCED=%s", item.FORCED))
		}
		if item.INSTREAM_ID != "" && item.TYPE == "CLOSED-CAPTIONS" {
			me.buffer.WriteString(fmt.Sprintf(",INSTREAM-ID=%s", item.INSTREAM_ID))
		}
		if item.CHARACTERISTICS != "" {
			me.buffer.WriteString(fmt.Sprintf(",CHARACTERISTICS=%s", item.CHARACTERISTICS))
		}
		if item.CHANNELS != "" {
			me.buffer.WriteString(fmt.Sprintf(",CHANNELS=%s", item.CHANNELS))
		}
		if item.URI != "" {
			me.buffer.WriteString(fmt.Sprintf(",URI=\"%s\"", item.URI))
		}
		me.buffer.WriteString("\n")
	}

	for i, item := range me.EXT_X_STREAM_INF {
		if i == 0 {
			me.buffer.WriteString("\n")
		}
		me.buffer.WriteString("#EXT-X-STREAM-INF:")
		me.buffer.WriteString(fmt.Sprintf("BANDWIDTH=%d", item.BANDWIDTH))
		me.buffer.WriteString(fmt.Sprintf(",CODECS=\"%s\"", item.CODECS))
		if item.AVERAGE_BANDWIDTH > 0 {
			me.buffer.WriteString(fmt.Sprintf(",AVERAGE-BANDWIDTH=%d", item.AVERAGE_BANDWIDTH))
		}
		if item.RESOLUTION != "" {
			me.buffer.WriteString(fmt.Sprintf(",RESOLUTION=%s", item.RESOLUTION))
		}
		if item.FRAME_RATE > 0 {
			me.buffer.WriteString(fmt.Sprintf(",FRAME-RATE=%.3f", item.FRAME_RATE))
		}
		if item.HDCP_LEVEL != "" {
			me.buffer.WriteString(fmt.Sprintf(",HDCP-LEVEL=%s", item.HDCP_LEVEL))
		}
		if item.ALLOWED_CPC != "" {
			me.buffer.WriteString(fmt.Sprintf(",ALLOWED-CPC=%s", item.ALLOWED_CPC))
		}
		if item.VIDEO_RANGE != "" {
			me.buffer.WriteString(fmt.Sprintf(",VIDEO-RANGE=%s", item.VIDEO_RANGE))
		}
		if item.AUDIO != "" {
			me.buffer.WriteString(fmt.Sprintf(",AUDIO=\"%s\"", item.AUDIO))
		}
		if item.VIDEO != "" {
			me.buffer.WriteString(fmt.Sprintf(",VIDEO=\"%s\"", item.VIDEO))
		}
		if item.SUBTITLES != "" {
			me.buffer.WriteString(fmt.Sprintf(",SUBTITLES=\"%s\"", item.SUBTITLES))
		}
		if item.CLOSED_CAPTIONS != "" {
			me.buffer.WriteString(fmt.Sprintf(",CLOSED-CAPTIONS=%s", item.CLOSED_CAPTIONS))
		}
		me.buffer.WriteString(fmt.Sprintf("\n%s\n", item.URI))
		if len(me.EXT_X_I_FRAME_STREAM_INF) > i {
			me.buffer.WriteString("#EXT-X-I-FRAME-STREAM-INF:")
			me.buffer.WriteString(fmt.Sprintf("BANDWIDTH=%d", item.BANDWIDTH))
			me.buffer.WriteString(fmt.Sprintf(",URI=\"%s\"\n", me.EXT_X_I_FRAME_STREAM_INF[i].URI))
		}
	}

	for i, item := range me.EXT_X_SESSION_DATA {
		if i == 0 {
			me.buffer.WriteString("\n")
		}
		me.buffer.WriteString("#EXT-X-SESSION-DATA:")
		me.buffer.WriteString(fmt.Sprintf("DATA-ID=\"%s\"", item.DATA_ID))
		if item.LANGUAGE != "" {
			me.buffer.WriteString(fmt.Sprintf(",LANGUAGE=\"%s\"", item.LANGUAGE))
		}
		if item.VALUE != "" {
			me.buffer.WriteString(fmt.Sprintf(",VALUE=\"%s\"", item.VALUE))
		}
		if item.URI != "" {
			me.buffer.WriteString(fmt.Sprintf(",URI=\"%s\"", item.URI))
		}
	}

	for i, item := range me.EXT_X_SESSION_KEY {
		if item.METHOD == "" || item.URI == "" {
			continue
		}
		if i == 0 {
			me.buffer.WriteString("\n")
		}
		me.buffer.WriteString("#EXT-X-SESSION-KEY:")
		me.buffer.WriteString(fmt.Sprintf("METHOD=%s", item.METHOD))
		me.buffer.WriteString(fmt.Sprintf(",URI=%s", item.URI))
		if item.IV != "" {
			me.buffer.WriteString(fmt.Sprintf(",IV=%s", item.IV))
		}
		if item.KEYFORMAT != "" {
			me.buffer.WriteString(fmt.Sprintf(",KEYFORMAT=%s", item.KEYFORMAT))
		}
		if item.KEYFORMATVERSIONS != "" {
			me.buffer.WriteString(fmt.Sprintf(",KEYFORMATVERSIONS=%s", item.KEYFORMATVERSIONS))
		}
		me.buffer.WriteString("\n")
	}

	return me.buffer.Bytes(), nil
}

// MasterPlaylistTags of m3u8
type MasterPlaylistTags struct {
	MediaOrMasterPlaylistTags
	EXT_X_MEDIA              []MediaAttributes
	EXT_X_STREAM_INF         []StreamInfAttributes
	EXT_X_I_FRAME_STREAM_INF []IFrameStreamInfAttributes
	EXT_X_SESSION_DATA       []SessionDataAttributes
	EXT_X_SESSION_KEY        []SessionKeyAttributes
}

// MediaAttributes defined for #EXT-X-MEDIA
type MediaAttributes struct {
	TYPE            string // 'AUDIO', 'VIDEO', 'SUBTITLES', 'CLOSED-CAPTIONS'
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

// StreamInfAttributes defined for #EXT-X-STREAM-INF
type StreamInfAttributes struct {
	BANDWIDTH         uint
	AVERAGE_BANDWIDTH uint
	CODECS            string // mp4a.40.2,avc1.4d401e
	RESOLUTION        string
	FRAME_RATE        float64 // 30.000
	HDCP_LEVEL        string  // 'TYPE-0', 'TYPE-1', 'NONE'
	ALLOWED_CPC       string
	VIDEO_RANGE       string // 'SDR', 'HLG', 'PQ'
	AUDIO             string
	VIDEO             string
	SUBTITLES         string
	CLOSED_CAPTIONS   string // 'NONE'
	URI               string
}

// IFrameStreamInfAttributes defined for #EXT-X-I-FRAME-STREAM-INF
type IFrameStreamInfAttributes struct {
	URI string
}

// SessionDataAttributes defined for #EXT-X-SESSION-DATA
type SessionDataAttributes struct {
	DATA_ID  string
	VALUE    string
	URI      string
	LANGUAGE string
}

// SessionKeyAttributes defined for #EXT-X-SESSION-KEY
type SessionKeyAttributes KeyAttributes
