package rtmp

import (
	"net"
	"time"
)

// TCPKeepAliveListener auto sets keepalive
type TCPKeepAliveListener struct {
	*net.TCPListener
	duration time.Duration
}

// Init this class
func (me *TCPKeepAliveListener) Init(l net.Listener, d time.Duration) *TCPKeepAliveListener {
	me.TCPListener = l.(*net.TCPListener)
	me.duration = d
	return me
}

// Accept the next incoming call and returns the new connection
func (me *TCPKeepAliveListener) Accept() (net.Conn, error) {
	c, err := me.AcceptTCP()
	if err == nil {
		c.SetKeepAlive(true)
		c.SetKeepAlivePeriod(me.duration)
	}

	return c, nil
}
