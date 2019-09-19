package config

import (
	basecfg "github.com/studease/common/utils/config"
)

// Channel config
type Channel struct {
	Pattern string `xml:"pattern,attr"`

	OnOpen  basecfg.URL `xml:""`
	OnClose basecfg.URL `xml:""`

	Protocol     string `xml:""`
	Capacity     int    `xml:""`
	Notification string `xml:""`

	Group    Group         `xml:""`
	UserList UserList      `xml:""`
	Visitor  Visitor       `xml:""`
	Query    basecfg.Query `xml:""`
	Proxy    basecfg.URL   `xml:""`
}

// Group config
type Group struct {
	Capacity     int    `xml:""`
	Notification string `xml:""`
}

// UserList config
type UserList struct {
	Enable   bool `xml:"enable,attr"`
	Interval int  `xml:""`
}

// Visitor config
type Visitor struct {
	Enable bool   `xml:"enable,attr"`
	Seed   int    `xml:""`
	Name   string `xml:""`
	Role   int    `xml:""`
	Icon   string `xml:""`
}
