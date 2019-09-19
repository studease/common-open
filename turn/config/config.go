package config

import (
	"encoding/xml"

	basecfg "github.com/studease/common/utils/config"
)

// Server config
type Server struct {
	XMLName xml.Name `xml:"Server"`
	basecfg.Listener
	REALM     string     `xml:""`
	Locations []Location `xml:"Location"`
}

// Location config
type Location struct {
	XMLName    xml.Name    `xml:"Server"`
	OnPlay     basecfg.URL `xml:""`
	OnPlayDone basecfg.URL `xml:""`
}
