package config

import (
	"encoding/xml"

	basecfg "github.com/studease/common/utils/config"
)

// Server config of rtmp.
type Server struct {
	basecfg.Listener
	Target    string     `xml:""`
	ChunkSize int        `xml:""`
	Locations []Location `xml:"Location"`
}

// Location config of rtmp server.
type Location struct {
	XMLName       xml.Name      `xml:"Location"`
	Pattern       string        `xml:"pattern,attr"`
	Handler       string        `xml:""`
	Proxy         basecfg.URL   `xml:""`
	OnOpen        basecfg.URL   `xml:""`
	OnClose       basecfg.URL   `xml:""`
	OnPublish     basecfg.URL   `xml:""`
	OnPublishDone basecfg.URL   `xml:""`
	OnPlay        basecfg.URL   `xml:""`
	OnPlayDone    basecfg.URL   `xml:""`
	DVRs          []basecfg.DVR `xml:"DVR"`
}
