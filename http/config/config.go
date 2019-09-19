package config

import (
	"encoding/xml"
	"fmt"
	"regexp"
	"strconv"
	"time"

	chatcfg "github.com/studease/common/chat/config"
	basecfg "github.com/studease/common/utils/config"
)

var (
	timedReg, _ = regexp.Compile("^(?:(\\d+):(\\d+))?-(?:(\\d+):(\\d+))?$")
)

// Server config
type Server struct {
	XMLName xml.Name `xml:"Server"`
	basecfg.Listener
	SSL struct {
		Enable bool   `xml:"enable,attr"`
		Cert   string `xml:""`
		Key    string `xml:""`
	} `xml:""`
	Locations []Location `xml:"Location"`
}

// Location config
type Location struct {
	XMLName xml.Name    `xml:"Location"`
	Pattern string      `xml:"pattern,attr"`
	Handler string      `xml:""`
	Proxy   basecfg.URL `xml:""`

	// ws-chat
	Protocol string            `xml:""`
	Channels []chatcfg.Channel `xml:"Channel"`
	Query    basecfg.Query     `xml:""`

	// sync
	Tracker    basecfg.URL `xml:""`
	Storage    basecfg.URL `xml:""`
	OnDone     basecfg.URL `xml:""`
	OnError    basecfg.URL `xml:""`
	Attributes []Item      `xml:"Attributes>Item"`
	Limitation Limitation  `xml:""`
}

// Item of Attributes
type Item struct {
	XMLName xml.Name `xml:"Item"`
	Name    string   `xml:"name,attr"`
	Value   string   `xml:",innerxml"`
}

// Limitation config
type Limitation struct {
	MaxTasks  int32 `xml:""`
	ChunkSize int64 `xml:""`
	Upload    int32 `xml:""`
	Download  int32 `xml:""`
	Timed     Timed `xml:""`
}

// Timed config
type Timed struct {
	Value       string `xml:",innerxml"`
	HourStart   int
	HourEnd     int
	MinuteStart int
	MinuteEnd   int
}

// Parse this config
func (me *Timed) Parse() error {
	if me.Value == "" {
		me.Value = "-"
	}

	arr := timedReg.FindStringSubmatch(me.Value)
	if arr == nil {
		return fmt.Errorf("not match")
	}

	if arr[1] != "" {
		n, err := strconv.ParseInt(arr[1], 10, 32)
		if err != nil {
			return err
		}

		me.HourStart = int(n)
	}

	if arr[2] != "" {
		n, err := strconv.ParseInt(arr[2], 10, 32)
		if err != nil {
			return err
		}

		me.MinuteStart = int(n)
	}

	if arr[3] != "" {
		n, err := strconv.ParseInt(arr[3], 10, 32)
		if err != nil {
			return err
		}

		me.HourEnd = int(n)
	}

	if arr[4] != "" {
		n, err := strconv.ParseInt(arr[4], 10, 32)
		if err != nil {
			return err
		}

		me.MinuteEnd = int(n)
	}

	return nil
}

// Within checks whether the time now is in the timed range
func (me *Timed) Within() bool {
	if me.Value == "-" {
		return true
	}

	now := time.Now()
	year := now.Year()
	month := now.Month()
	day := now.Day()

	start := time.Date(year, month, day, me.HourStart, me.MinuteStart, 0, 0, time.Local)
	if now.Before(start) {
		return false
	}

	end := time.Date(year, month, day, me.HourEnd, me.MinuteEnd, 0, 0, time.Local)
	if !start.Before(end) {
		end = end.Add(24 * time.Hour)
	}

	if now.After(end) {
		return false
	}

	return true
}

// AboutToStart returns a duration of how long to reach the start time
func (me *Timed) AboutToStart() time.Duration {
	if me.Value == "-" {
		return 0
	}

	now := time.Now()
	year := now.Year()
	month := now.Month()
	day := now.Day()

	start := time.Date(year, month, day, me.HourStart, me.MinuteStart, 0, 0, time.Local)
	if now.Before(start) {
		return start.Sub(now)
	}

	return 0
}
