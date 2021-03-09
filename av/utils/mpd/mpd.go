package mpd

import (
	"encoding/xml"
	"fmt"
	"time"

	"github.com/studease/common/av"
)

// Static constants
const (
	PROFILE_FULL            = "urn:mpeg:dash:profile:full:2011"
	PROFILE_ISOFF_ON_DEMAND = "urn:mpeg:dash:profile:isoff-on-demand:2011"
	PROFILE_ISOFF_MAIN      = "urn:mpeg:dash:profile:isoff-main:2011"
	PROFILE_ISOFF_LIVE      = "urn:mpeg:dash:profile:isoff-live:2011"
	PROFILE_MP2T_MAIN       = "urn:mpeg:dash:profile:mp2t-main:2011"
	PROFILE_MP2T_SIMPLE     = "urn:mpeg:dash:profile:mp2t-simple:2011"

	TYPE_STATIC  = "static"
	TYPE_DYNAMIC = "dynamic"
)

var (
	ratios = [][2]uint32{
		{16, 10}, {16, 9}, {4, 3},
	}
)

// MPD legend:
// For attributes: M=Mandatory, O=Optional, OD=Optional with Default Value, CM=Conditionally Mandatory;
// For elements: <minOccurs>..<maxOccurs> (N=unbounded)
type MPD struct {
	Xmlns                      string               `xml:"xmlns,attr"`
	XmlnsXsi                   string               `xml:"xmlns:xsi,attr"`
	XmlnsXlink                 string               `xml:"xmlns:xlink,attr"`
	XsiSchemaLocation          string               `xml:"xsi:schemaLocation,attr"`
	XlinkType                  string               `xml:"xlink:type,attr,omitempty"`                 // O - 'simple'
	XlinkHref                  string               `xml:"xlink:href,attr,omitempty"`                 // O
	XlinkShow                  string               `xml:"xlink:show,attr,omitempty"`                 // O
	XlinkActuate               string               `xml:"xlink:actuate,attr,omitempty"`              // OD - 'onRequest'(default), 'onLoad'
	ID                         string               `xml:"id,attr,omitempty"`                         // O
	Profiles                   string               `xml:"profiles,attr"`                             // M
	Type                       string               `xml:"type,attr,omitempty"`                       // OD - default: static
	AvailabilityStartTime      DateTime             `xml:"availabilityStartTime,attr,omitempty"`      // CM - @type='dynamic'
	PublishTime                DateTime             `xml:"publishTime,attr,omitempty"`                // OD - @type='dynamic'
	AvailabilityEndTime        DateTime             `xml:"availabilityEndTime,attr,omitempty"`        // O
	MediaPresentationDuration  Duration             `xml:"mediaPresentationDuration,attr,omitempty"`  // O
	MinimumUpdatePeriod        Duration             `xml:"minimumUpdatePeriod,attr,omitempty"`        // O
	MinBufferTime              Duration             `xml:"minBufferTime,attr"`                        // M
	TimeShiftBufferDepth       Duration             `xml:"timeShiftBufferDepth,attr,omitempty"`       // O
	SuggestedPresentationDelay Duration             `xml:"suggestedPresentationDelay,attr,omitempty"` // O
	MaxSegmentDuration         Duration             `xml:"maxSegmentDuration,attr,omitempty"`         // O
	MaxSubsegmentDuration      Duration             `xml:"maxSubsegmentDuration,attr,omitempty"`      // O
	ProgramInformation         []ProgramInformation `xml:"ProgramInformation,omitempty"`              // 0...N
	BaseURL                    []BaseURL            `xml:"BaseURL,omitempty"`                         // 0...N
	Location                   []string             `xml:"Location,omitempty"`                        // 0...N
	Period                     []Period             `xml:"Period"`                                    // 1...N
	Metrics                    []Metrics            `xml:"Metrics,omitempty"`                         // 0...N
}

// Init this class
func (me *MPD) Init(profile string, typ string) *MPD {
	me.Xmlns = "urn:mpeg:dash:schema:mpd:2011"
	me.XmlnsXlink = "http://www.w3.org/1999/xlink"
	me.XmlnsXsi = "http://www.w3.org/2001/XMLSchema-instance"
	me.XsiSchemaLocation = "urn:mpeg:DASH:schema:MPD:2011 http://standards.iso.org/ittf/PubliclyAvailableStandards/MPEG-DASH_schema_files/DASH-MPD.xsd"
	me.Profiles = profile
	me.Type = typ
	return me
}

// Marshal returns the XML encoding of MPD
func (me *MPD) Marshal() ([]byte, error) {
	b, err := xml.MarshalIndent(me, "", "    ")
	return b, err
}

// Period element of MPD
type Period struct {
	XlinkHref          string            `xml:"xlink:href,attr,omitempty"`         // O
	XlinkActuate       string            `xml:"xlink:actuate,attr,omitempty"`      // OD - 'onRequest'(default), 'onLoad'
	Id                 string            `xml:"id,attr,omitempty"`                 // O
	Start              Duration          `xml:"start,attr,omitempty"`              // O
	Duration           Duration          `xml:"duration,attr,omitempty"`           // O
	BitstreamSwitching bool              `xml:"bitstreamSwitching,attr,omitempty"` // OD - default: false
	BaseURL            []BaseURL         `xml:"BaseURL,omitempty"`                 // 0...N
	SegmentBase        []SegmentBase     `xml:"SegmentBase,omitempty"`             // 0...1
	SegmentList        []SegmentList     `xml:"SegmentList,omitempty"`             // 0...1
	SegmentTemplate    []SegmentTemplate `xml:"SegmentTemplate,omitempty"`         // 0...1
	AssetIdentifier    []Descriptor      `xml:"AssetIdentifier,omitempty"`         // 0...1
	EventStream        []EventStream     `xml:"EventStream,omitempty"`             // 0...N
	AdaptationSet      []AdaptationSet   `xml:"AdaptationSet,omitempty"`           // 0...N
	Subset             []Subset          `xml:"Subset,omitempty"`                  // 0...N
}

// AdaptationSet element of Period
type AdaptationSet struct {
	RepresentationBase
	XlinkHref               string             `xml:"xlink:href,attr,omitempty"`    // O
	XlinkActuate            string             `xml:"xlink:actuate,attr,omitempty"` // OD - 'onRequest'(default), 'onLoad'
	Id                      uint               `xml:"id,attr,omitempty"`            // O
	Group                   uint               `xml:"group,attr,omitempty"`         // O
	Lang                    string             `xml:"lang,attr,omitempty"`          // O
	ContentType             string             `xml:"contentType,attr,omitempty"`   // O
	Par                     Ratio              `xml:"par,attr,omitempty"`           // O
	MinBandwidth            uint               `xml:"minBandwidth,attr,omitempty"`  // O
	MaxBandwidth            uint               `xml:"maxBandwidth,attr,omitempty"`  // O
	MinWidth                uint               `xml:"minWidth,attr,omitempty"`      // O
	MaxWidth                uint               `xml:"maxWidth,attr,omitempty"`      // O
	MinHeight               uint               `xml:"minHeight,attr,omitempty"`     // O
	MaxHeight               uint               `xml:"maxHeight,attr,omitempty"`     // O
	MinFrameRate            string             `xml:"minFrameRate,attr,omitempty"`  // O
	MaxFrameRate            string             `xml:"maxFrameRate,attr,omitempty"`  // O
	Bandwidth               int                `xml:"bandwidth,attr,omitempty"`
	SegmentAlignment        bool               `xml:"segmentAlignment,attr,omitempty"`        // OD - default: false
	BitstreamSwitching      bool               `xml:"bitstreamSwitching,attr,omitempty"`      // O
	SubsegmentAlignment     bool               `xml:"subsegmentAlignment,attr,omitempty"`     // OD - default: false
	SubsegmentStartsWithSAP uint               `xml:"subsegmentStartsWithSAP,attr,omitempty"` // OD - 0(default), 1
	Accessibility           []Descriptor       `xml:"Accessibility,omitempty"`                // 0...N
	Role                    []Descriptor       `xml:"Role,omitempty"`                         // 0...N
	Rating                  []Descriptor       `xml:"Rating,omitempty"`                       // 0...N
	Viewpoint               []Descriptor       `xml:"Viewpoint,omitempty"`                    // 0...N
	ContentComponent        []ContentComponent `xml:"ContentComponent,omitempty"`             // 0...N
	BaseURL                 []BaseURL          `xml:"BaseURL,omitempty"`                      // 0...N
	SegmentBase             []SegmentBase      `xml:"SegmentBase,omitempty"`                  // 0...1
	SegmentList             []SegmentList      `xml:"SegmentList,omitempty"`                  // 0...1
	SegmentTemplate         []SegmentTemplate  `xml:"SegmentTemplate,omitempty"`              // 0...1
	Representation          []Representation   `xml:"Representation,omitempty"`               // 0...N
}

// ContentComponent element of AdaptationSet
type ContentComponent struct {
	Id            uint         `xml:"id,attr,omitempty"`          // O
	Lang          string       `xml:"lang,attr,omitempty"`        // O
	ContentType   string       `xml:"contentType,attr,omitempty"` // O
	Par           Ratio        `xml:"par,attr,omitempty"`         // O
	Accessibility []Descriptor `xml:"Accessibility,omitempty"`    // 0...N
	Role          []Descriptor `xml:"Role,omitempty"`             // 0...N
	Rating        []Descriptor `xml:"Rating,omitempty"`           // 0...N
	Viewpoint     []Descriptor `xml:"Viewpoint,omitempty"`        // 0...N
}

// Representation element of AdaptationSet
type Representation struct {
	RepresentationBase
	Id                     string              `xml:"id,attr"`                               // M
	Bandwidth              uint                `xml:"bandwidth,attr"`                        // M
	QualityRanking         uint                `xml:"qualityRanking,attr,omitempty"`         // O
	DependencyId           []string            `xml:"dependencyId,attr,omitempty"`           // O
	MediaStreamStructureId []string            `xml:"mediaStreamStructureId,attr,omitempty"` // O
	BaseURL                []BaseURL           `xml:"BaseURL,omitempty"`                     // 0...N
	SubRepresentation      []SubRepresentation `xml:"SubRepresentation,omitempty"`           // 0...N
	SegmentBase            []SegmentBase       `xml:"SegmentBase,omitempty"`                 // 0...1
	SegmentList            []SegmentList       `xml:"SegmentList,omitempty"`                 // 0...1
	SegmentTemplate        []SegmentTemplate   `xml:"SegmentTemplate,omitempty"`             // 0...1
}

// SubRepresentation element of Representation
type SubRepresentation struct {
	RepresentationBase
	Level            uint     `xml:"level,attr,omitempty"`           // O
	DependencyLevel  []uint   `xml:"dependencyLevel,attr,omitempty"` // O
	Bandwidth        uint     `xml:"bandwidth,attr,omitempty"`       // CM - @level is present
	ContentComponent []string `xml:"contentComponent,attr"`          // O
}

// RepresentationBase element in Representation, SubRepresentation
type RepresentationBase struct {
	Profiles                  string       `xml:"profiles,attr,omitempty"`             // O
	Width                     uint         `xml:"width,attr,omitempty"`                // O
	Height                    uint         `xml:"height,attr,omitempty"`               // O
	Sar                       Ratio        `xml:"sar,attr,omitempty"`                  // O
	FrameRate                 string       `xml:"frameRate,attr,omitempty"`            // O
	AudioSamplingRate         string       `xml:"audioSamplingRate,attr,omitempty"`    // O
	MimeType                  string       `xml:"mimeType,attr,omitempty"`             // M
	SegmentProfiles           string       `xml:"segmentProfiles,attr,omitempty"`      // O
	Codecs                    string       `xml:"codecs,attr,omitempty"`               // O
	MaximumSAPPeriod          float64      `xml:"maximumSAPPeriod,attr,omitempty"`     // O
	StartWithSAP              uint         `xml:"startWithSAP,attr,omitempty"`         // O - 0(default), 1
	MaxPlayoutRate            float64      `xml:"maxPlayoutRate,attr,omitempty"`       // O
	CodingDependency          bool         `xml:"codingDependency,attr,omitempty"`     // O
	ScanType                  string       `xml:"scanType,attr,omitempty"`             // O - 'progressive'(default), 'interlaced', 'unknown'
	FramePacking              []Descriptor `xml:"FramePacking,omitempty"`              // 0...N
	AudioChannelConfiguration []Descriptor `xml:"AudioChannelConfiguration,omitempty"` // 0...N
	ContentProtection         []Descriptor `xml:"ContentProtection,omitempty"`         // 0...N
	EssentialProperty         []Descriptor `xml:"EssentialProperty,omitempty"`         // 0...N
	SupplementalProperty      []Descriptor `xml:"SupplementalProperty,omitempty"`      // 0...N
	InbandEventStream         []Descriptor `xml:"InbandEventStream,omitempty"`         // 0...N
}

// Subset element of Period
type Subset struct {
	Contains []uint `xml:"contains,attr,omitempty"` // M
	Id       string `xml:"id,attr,omitempty"`       // O
}

// SegmentBase element (inheritable)
type SegmentBase struct {
	Timescale                uint     `xml:"timescale,attr,omitempty"`                // O - default: 1
	PresentationTimeOffset   uint64   `xml:"presentationTimeOffset,attr,omitempty"`   // O
	TimeShiftBufferDepth     Duration `xml:"timeShiftBufferDepth,attr,omitempty"`     // O
	IndexRange               string   `xml:"indexRange,attr,omitempty"`               // O
	IndexRangeExact          bool     `xml:"indexRangeExact,attr,omitempty"`          // OD - default: false
	AvailabilityTimeOffset   float64  `xml:"availabilityTimeOffset,attr,omitempty"`   // O
	AvailabilityTimeComplete bool     `xml:"availabilityTimeComplete,attr,omitempty"` // O
	Initialization           []URL    `xml:"Initialization,omitempty"`                // 0...1
	RepresentationIndex      []URL    `xml:"RepresentationIndex,omitempty"`           // 0...1
}

// MultipleSegmentBase element in SegmentList, SegmentTemplate
type MultipleSegmentBase struct {
	SegmentBase
	Duration           uint              `xml:"duration,attr,omitempty"`      // O
	StartNumber        string            `xml:"startNumber,attr,omitempty"`   // O
	SegmentTimeline    []SegmentTimeline `xml:"SegmentTimeline,omitempty"`    // 0...1
	BitstreamSwitching []URL             `xml:"BitstreamSwitching,omitempty"` // 0...1
}

// URL element
type URL struct {
	SourceURL string `xml:"sourceURL,attr,omitempty"` // O
	Range     string `xml:"range,attr,omitempty"`     // O
}

// SegmentList element (inheritable)
type SegmentList struct {
	MultipleSegmentBase
	XlinkHref    string       `xml:"xlink:href,attr,omitempty"`    // O
	XlinkActuate string       `xml:"xlink:actuate,attr,omitempty"` // OD - 'onRequest'(default), 'onLoad'
	SegmentURL   []SegmentURL `xml:"SegmentURL,omitempty"`         // 0...N
}

// SegmentURL element of SegmentList
type SegmentURL struct {
	Media      string `xml:"media,attr,omitempty"`      // O
	MediaRange string `xml:"mediaRange,attr,omitempty"` // O
	Index      string `xml:"index,attr,omitempty"`      // O
	IndexRange string `xml:"indexRange,attr,omitempty"` // O
}

// SegmentTemplate element (inheritable)
type SegmentTemplate struct {
	MultipleSegmentBase
	Media              string `xml:"media,attr,omitempty"`              // O
	Index              string `xml:"index,attr,omitempty"`              // O
	Initialization     string `xml:"initialization,attr,omitempty"`     // O
	BitstreamSwitching string `xml:"bitstreamSwitching,attr,omitempty"` // O
}

// SegmentTimeline element of MultipleSegmentBase
type SegmentTimeline struct {
	S []S `xml:"S,omitempty"` // 1...N
}

// S element
type S struct {
	T string `xml:"t,attr,omitempty"` // O
	D string `xml:"d,attr"`           // M
	R int    `xml:"r,attr,omitempty"` // OD - default: 0
}

// BaseURL element (inheritable)
type BaseURL struct {
	ServiceLocation          string  `xml:"serviceLocation,attr,omitempty"`          // O
	ByteRange                string  `xml:"byteRange,attr,omitempty"`                // O
	AvailabilityTimeOffset   float64 `xml:"availabilityTimeOffset,attr,omitempty"`   // O
	AvailabilityTimeComplete bool    `xml:"availabilityTimeComplete,attr,omitempty"` // O
	Content                  string  `xml:",innerxml"`
}

// ProgramInformation element of MPD
type ProgramInformation struct {
	Lang               string   `xml:"lang,attr,omitempty"`               // O
	MoreInformationURL string   `xml:"moreInformationURL,attr,omitempty"` // O
	Title              []string `xml:"Title,omitempty"`                   // 0...1
	Source             []string `xml:"Source,omitempty"`                  // 0...1
	Copyright          []string `xml:"Copyright,omitempty"`               // 0...1
}

// Descriptor is a common type
type Descriptor struct {
	SchemeIdUri string `xml:"schemeIdUri,attr,omitempty"` // M
	Value       string `xml:"value,attr,omitempty"`       // O
	Id          string `xml:"id,attr,omitempty"`          // O
}

// Metrics element of MPD
type Metrics struct {
	Metrics   string       `xml:"metrics,attr,omitempty"` // M
	Reporting []Descriptor `xml:"Reporting,omitempty"`    // 1...N
	Range     []Range      `xml:"Range,omitempty"`        // 0...N
}

// Range element
type Range struct {
	Starttime Duration `xml:"starttime,attr,omitempty"` // O
	Duration  Duration `xml:"duration,attr,omitempty"`  // O
}

// EventStream element of Period
type EventStream struct {
	XlinkHref    string  `xml:"xlink:href,attr,omitempty"`    // O
	XlinkActuate string  `xml:"xlink:actuate,attr,omitempty"` // OD - 'onRequest'(default), 'onLoad'
	SchemeIdUri  string  `xml:"schemeIdUri,attr,omitempty"`   // M
	Value        string  `xml:"value,attr,omitempty"`         // O
	Timescale    uint    `xml:"timescale,attr,omitempty"`     // O
	Event        []Event `xml:"Event,omitempty"`              // 0...N
}

// Event element of EventStream
type Event struct {
	PresentationTime uint64 `xml:"presentationTime,attr,omitempty"` // OD - default: 0
	Duration         uint64 `xml:"duration,attr,omitempty"`         // O
	Id               uint   `xml:"id,attr,omitempty"`               // O
}

// DateTime is a formated string, like "2006-01-02T15:04:05.000Z"
type DateTime string

// FormatDateTime returns a formated string by the given time
func FormatDateTime(t time.Time) DateTime {
	return DateTime(t.In(av.UTC).Format("2006-01-02T15:04:05.000Z"))
}

// Duration is a formated string, like "P6Y1M2DT15H4S5.000S"
type Duration string

// FormatDuration returns a formated string by the given duration
func FormatDuration(d time.Duration) Duration {
	var (
		n   time.Duration
		tmp = "P"
	)

	if n = d / (365 * 24 * time.Hour); n > 0 {
		tmp += fmt.Sprintf("%dY", n)
		d %= 365 * 24 * time.Hour
	}

	if n = d / (30 * 24 * time.Hour); n > 0 {
		tmp += fmt.Sprintf("%dM", n)
		d %= 30 * 24 * time.Hour
	}

	if n = d / (24 * time.Hour); n > 0 {
		tmp += fmt.Sprintf("%dD", n)
		d %= 24 * time.Hour
	}

	tmp += "T"

	if n = d / time.Hour; n > 0 {
		tmp += fmt.Sprintf("%dH", n)
		d %= time.Hour
	}

	if n = d / time.Minute; n > 0 {
		tmp += fmt.Sprintf("%dM", n)
		d %= time.Minute
	}

	if n = d / time.Millisecond; n > 0 || tmp == "PT" {
		tmp += fmt.Sprintf("%.3gS", float64(n)/1000)
	}

	return Duration(tmp)
}

// Ratio is a formated string, like "16:9"
type Ratio string

// FormatRatio returns a formated string by the given width and height
func FormatRatio(w uint32, h uint32) Ratio {
	for _, arr := range ratios {
		if w*arr[1] == h*arr[0] {
			w = arr[0]
			h = arr[1]
			break
		}
	}

	return Ratio(fmt.Sprintf("%d:%d", w, h))
}
