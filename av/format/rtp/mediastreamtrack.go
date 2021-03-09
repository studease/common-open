package rtp

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/studease/common/av"
	"github.com/studease/common/av/format"
	"github.com/studease/common/log"
)

// MediaStreamTrack inherits from MediaStreamTrack, holds properties which describes a RTP transport.
type MediaStreamTrack struct {
	format.MediaStreamTrack

	logger     log.ILogger
	parameters map[string]string
	handlers   map[string]func(string) error
	Transport  string
	Channel    byte // for interleaved rtp packet
	Control    byte // for interleaved rtcp packet
	PSent      uint32
	OSent      uint32
}

// Init this class.
func (me *MediaStreamTrack) Init(kind string, source av.IMediaStreamTrackSource, logger log.ILogger) *MediaStreamTrack {
	me.MediaStreamTrack.Init(kind, source, logger)
	me.logger = logger
	me.parameters = make(map[string]string)
	me.handlers = map[string]func(string) error{
		"unicast":     me.processNothing,     //
		"multicast":   me.processMulticast,   //
		"layers":      me.processNothing,     //
		"destination": me.processDestination, // RFC 2326 section 12.39, replaced by dest_addr in RFC 7826 section I.2
		"client_port": me.processClientPort,  // RFC 2326 section 12.39, replaced by dest_addr in RFC 7826 section I.2
		"dest_addr":   me.processDestAddr,    // RFC 7826 section 18.54
		"mode":        me.processNothing,     //
		"interleaved": me.processInterleaved, //
		"MIKEY":       me.processNothing,     // RFC 7826 section 18.54
		"ttl":         me.processNothing,     //
		"ssrc":        me.processNothing,     //
		"RTCP-mux":    me.processNothing,     // RFC 7826 section 18.54
		"setup":       me.processNothing,     // RFC 7826 section 18.54
		"connection":  me.processNothing,     // RFC 7826 section 18.54
		"port":        me.processNothing,     // RFC 2326 section 12.39, replaced by src_addr, dest_addr in RFC 2326 section I.2
		"append":      me.processNothing,     // RFC 2326 section 12.39
	}
	me.Transport = ""
	me.Channel = 0x00
	me.Control = 0x00
	me.PSent = 0
	me.OSent = 0
	return me
}

// SetTransport sets the transport given as the argument
func (me *MediaStreamTrack) SetTransport(transport string) error {
	arr := strings.Split(transport, ";")
	if len(arr) < 3 {
		return fmt.Errorf("parameters not enough: %s", transport)
	}

	me.Transport = arr[0]

	for i := 1; i < len(arr); i++ {
		item := arr[i]

		j := strings.IndexByte(item, '=')
		k := item
		v := ""
		if j != -1 {
			k = string([]byte(item)[:j])
			v = string([]byte(item)[j+1:])
		}

		if h := me.handlers[k]; h != nil {
			if err := h(v); err != nil {
				me.logger.Errorf("Failed to handle transport parameter: %v", err)
				return err
			}
		}
		me.parameters[k] = v
	}

	return nil
}

func (me *MediaStreamTrack) processMulticast(param string) error {
	return fmt.Errorf("multicast not supported: %s", param)
}

func (me *MediaStreamTrack) processDestination(param string) error {
	// TODO(tonylau): support udp in rtsp 1.x
	return fmt.Errorf("destination not supported: %s", param)
}

func (me *MediaStreamTrack) processClientPort(param string) error {
	// TODO(tonylau): support udp in rtsp 1.x
	return fmt.Errorf("client_port not supported: %s", param)
}

func (me *MediaStreamTrack) processDestAddr(param string) error {
	// TODO(tonylau): support udp in rtsp 2.x
	return fmt.Errorf("dest_addr not supported: %s", param)
}

func (me *MediaStreamTrack) processInterleaved(param string) error {
	arr := strings.Split(param, "-")

	n, err := strconv.Atoi(arr[0])
	if err != nil {
		return err
	}

	me.Channel = byte(n)

	n, err = strconv.Atoi(arr[1])
	if err != nil {
		return err
	}

	me.Control = byte(n)
	return nil
}

func (me *MediaStreamTrack) processNothing(param string) error {
	return nil
}

// SetParameter sets a parameter with the given key-value pair.
func (me *MediaStreamTrack) SetParameter(key string, value string) {
	me.parameters[key] = value
}

// GetParameter returns the value of parameter by the key.
func (me *MediaStreamTrack) GetParameter(key string) string {
	return me.parameters[key]
}
