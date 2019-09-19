package config

import (
	"encoding/xml"

	basecfg "github.com/studease/common/utils/config"
)

// Server config
type Server struct {
	XMLName xml.Name `xml:"Server"`
	basecfg.Listener
	Locations []Location `xml:"Location"`
}

// Location config
type Location struct {
	XMLName    xml.Name    `xml:"Location"`
	OnPlay     basecfg.URL `xml:""`
	OnPlayDone basecfg.URL `xml:""`
	Proxy      basecfg.URL `xml:""`
}
