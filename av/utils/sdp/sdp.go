package sdp

import (
	"fmt"
)

const (
	// DEFAULT_VERSION of this lib used.
	DEFAULT_VERSION uint = 0
)

// Static constants.
const (
	CRLF        = "\r\n"
	IN          = "IN"          // <nettype>
	IP4         = "IP4"         // <addrtype>
	IP6         = "IP6"         // <addrtype>
	CT          = "CT"          // <bwtype>
	AS          = "AS"          // <bwtype>
	CLEAR       = "clear"       // <method>
	BASE64      = "base64"      // <method>
	URI         = "uri"         // <method>
	PROMPT      = "prompt"      // <method>
	AUDIO       = "audio"       // <media>
	VIDEO       = "video"       // <media>
	TEXT        = "text"        // <media>
	APPLICATION = "application" // <media>
	MESSAGE     = "message"     // <media>
	UDP         = "upd"         // <proto>
	RTP_AVP     = "RTP/AVP"     // <proto>
	RTP_AVP_UDP = "RTP/AVP/UDP" // <proto>
	RTP_AVP_TCP = "RTP/AVP/TCP" // <proto>
	RTP_SAVP    = "RTP/SAVP"    // <proto>
)

// S=Session Level, M=Media Level, E=Either.
const (
	CAT       = "cat"       // S - a=cat:<category>
	KEYWDS    = "keywds"    // S - a=keywds:<keywords>
	TOOL      = "tool"      // S - a=tool:<name and version of tool>
	PTIME     = "ptime"     // M - a=ptime:<packet time>
	MAXPTIME  = "maxptime"  // M - a=maxptime:<maximum packet time>
	RTPMAP    = "rtpmap"    // M - a=rtpmap:<payload type> <encoding name>/<clock rate> [/<encoding parameters>]
	RECVONLY  = "recvonly"  // E - a=recvonly
	SENDRECV  = "sendrecv"  // E - a=sendrecv
	SENDONLY  = "sendonly"  // E - a=sendonly
	INACTIVE  = "inactive"  // E - a=inactive
	ORIENT    = "orient"    // M - a=orient:<orientation>
	TYPE      = "type"      // S - a=type:<conference type>
	CHARSET   = "charset"   // S - a=charset:<character set>
	SDPLANG   = "sdplang"   // E - a=sdplang:<language tag>
	LANG      = "lang"      // E - a=lang:<language tag>
	FRAMERATE = "framerate" // M - a=framerate:<frame rate>
	QUALITY   = "quality"   // M - a=quality:<quality>
	FMTP      = "fmtp"      // M - a=fmtp:<format> <format specific parameters>
)

// SDP defines an SDP file.
type SDP struct {
	V  uint        //   Protocol Version, v=0
	O  Origin      //   Origin, o=<username> <sess-id> <sess-version> <nettype> <addrtype> <unicast-address>
	S  string      //   Session Name, s=<session name>
	I  string      // * Session Information, i=<session description>
	U  string      // * URI, u=<uri>
	E  string      // * Email Address, e=<email-address>
	P  string      // * Phone Number, p=<phone-number>
	C  Connection  // * Connection Data, c=<nettype> <addrtype> <connection-address>
	B  []Attribute // * Bandwidth, b=<bwtype>:<bandwidth>
	TD []TD        //   One or more time descriptions
	Z  []string    // * Time Zones, z=<adjustment time> <offset> <adjustment time> <offset> ....
	K  Attribute   // * Encryption Keys, k=<method>, k=<method>:<encryption key>
	A  []Attribute //   Zero or more attribute lines
	MD []MD        //   Zero or more media descriptions
}

// Init this class.
func (me *SDP) Init(ver uint) *SDP {
	me.V = ver
	return me
}

// Marshal returns the lined encoding of this SDP.
func (me *SDP) Marshal() ([]byte, error) {
	tmp := fmt.Sprintf("v=%d", me.V) + CRLF
	tmp += fmt.Sprintf("o=%s %s %s %s %s %s", me.O.UserName, me.O.SessID, me.O.SessVersion,
		me.O.NetType, me.O.AddrType, me.O.Address) + CRLF
	tmp += fmt.Sprintf("s=%s", me.S) + CRLF

	if me.I != "" {
		tmp += fmt.Sprintf("i=%s", me.I) + CRLF
	}
	if me.U != "" {
		tmp += fmt.Sprintf("u=%s", me.U) + CRLF
	}
	if me.E != "" {
		tmp += fmt.Sprintf("e=%s", me.E) + CRLF
	}
	if me.P != "" {
		tmp += fmt.Sprintf("p=%s", me.P) + CRLF
	}
	if me.C.NetType != "" {
		tmp += fmt.Sprintf("c=%s %s %s", me.C.NetType, me.C.AddrType, me.C.Address) + CRLF
	}

	for _, b := range me.B {
		tmp += fmt.Sprintf("b=%s", b.Fmt()) + CRLF
	}

	for _, v := range me.TD {
		if v.T != "" {
			tmp += fmt.Sprintf("t=%s", v.T) + CRLF
		}

		if v.R != "" {
			tmp += fmt.Sprintf("r=%s", v.R) + CRLF
		}
	}

	for _, v := range me.Z {
		tmp += fmt.Sprintf("z=%s", v) + CRLF
	}

	if me.K.Key != "" {
		tmp += fmt.Sprintf("k=%s", me.K.Fmt()) + CRLF
	}

	for _, a := range me.A {
		tmp += fmt.Sprintf("a=%s", a.Fmt()) + CRLF
	}

	for _, v := range me.MD {
		tmp += fmt.Sprintf("m=%s %d %s %d", v.M.Media, v.M.Port, v.M.Proto, v.M.Fmt) + CRLF

		if v.I != "" {
			tmp += fmt.Sprintf("i=%s", v.I) + CRLF
		}

		if v.C.NetType != "" {
			tmp += fmt.Sprintf("c=%s %s %s", v.C.NetType, v.C.AddrType, v.C.Address) + CRLF
		}

		for _, b := range v.B {
			tmp += fmt.Sprintf("b=%s", b.Fmt()) + CRLF
		}

		if v.K.Key != "" {
			tmp += fmt.Sprintf("k=%s", v.K.Fmt()) + CRLF
		}

		for _, a := range v.A {
			tmp += fmt.Sprintf("a=%s", a.Fmt()) + CRLF
		}
	}

	return []byte(tmp), nil
}

// Origin => o=<username> <sess-id> <sess-version> <nettype> <addrtype> <unicast-address>
type Origin struct {
	UserName    string
	SessID      string
	SessVersion string
	NetType     string
	AddrType    string
	Address     string
}

// Connection => c=<nettype> <addrtype> <connection-address>
type Connection struct {
	NetType  string
	AddrType string
	Address  string
}

// TD (Time Description)
type TD struct {
	T string //   Timing, t=<start-time> <stop-time>
	R string // * Repeat Times, r=<repeat interval> <active duration> <offsets from start-time>
}

// Attribute defines an attribute line.
type Attribute struct {
	Key   string
	Value string
}

// Fmt returns a formated string, like "k:v".
func (me *Attribute) Fmt() string {
	tmp := me.Key

	if me.Value != "" {
		tmp += ":" + me.Value
	}

	return tmp
}

// MD (Media Description)
type MD struct {
	M Media       //   Media Descriptions, m=<media> <port> <proto> <fmt> ...
	I string      // * Media Title, i=<title>
	C Connection  // * Connection Data, c=<nettype> <addrtype> <connection-address>
	B []Attribute // * Bandwidth, b=<bwtype>:<bandwidth>
	K Attribute   // * Encryption Keys, k=<method>, k=<method>:<encryption key>
	A []Attribute //   Zero or more attribute lines
}

// Media => m=<media> <port> <proto> <fmt> ...
type Media struct {
	Media string
	Port  int
	Proto string
	Fmt   int
}
