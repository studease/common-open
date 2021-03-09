package rtmp

import (
	"fmt"
	"net"
	"sync"
	"sync/atomic"
	"time"

	Code "github.com/studease/common/events/netstatusevent/code"
	Level "github.com/studease/common/events/netstatusevent/level"
	"github.com/studease/common/log"
	rtmpcfg "github.com/studease/common/rtmp/config"
	"github.com/studease/common/rtmp/message/command"
	"github.com/studease/common/utils"
)

// Static constants.
const (
	DEFAULT_PORT             = 1935
	DEFAULT_TIMEOUT          = 10
	DEFAULT_MAX_IDLE_TIME    = 3600
	DEFAULT_SEND_BUFFER_SIZE = 65536
	DEFAULT_READ_BUFFER_SIZE = 65536
	DEFAULT_ROOT             = "applications"
	DEFAULT_CORS             = "webroot/crossdomain.xml"
	DEFAULT_TARGET           = "conf/target.xml"
	DEFAULT_CHUNK_SIZE       = 4096
	DEFAULT_ACK_WINDOW_SIZE  = 2500000
	DEFAULT_PEER_BANDWIDTH   = 2500000
)

var (
	servers = make(map[int]*Server)
)

// Server defines parameters for running an RTMP server.
type Server struct {
	config       *rtmpcfg.Server
	logger       log.ILogger
	factory      log.ILoggerFactory
	mux          utils.Mux
	mtx          sync.RWMutex
	applications map[string]*Application
}

// Init this class.
func (me *Server) Init(cfg *rtmpcfg.Server, logger log.ILogger, factory log.ILoggerFactory) *Server {
	me.mux.Init()
	me.config = cfg
	me.logger = logger
	me.factory = factory
	me.applications = make(map[string]*Application)

	if cfg.Port == 0 {
		cfg.Port = DEFAULT_PORT
	}
	if cfg.Timeout == 0 {
		cfg.Timeout = DEFAULT_TIMEOUT
	}
	if cfg.MaxIdleTime == 0 {
		cfg.MaxIdleTime = DEFAULT_MAX_IDLE_TIME
	}
	if cfg.SendBufferSize == 0 {
		cfg.SendBufferSize = DEFAULT_SEND_BUFFER_SIZE
	}
	if cfg.ReadBufferSize == 0 {
		cfg.ReadBufferSize = DEFAULT_READ_BUFFER_SIZE
	}
	if cfg.Root == "" {
		cfg.Root = DEFAULT_ROOT
	}
	if cfg.Cors == "" {
		cfg.Cors = DEFAULT_CORS
	}
	if cfg.Target == "" {
		cfg.Target = DEFAULT_TARGET
	}
	if cfg.ChunkSize < 128 || cfg.ChunkSize > 65536 {
		cfg.ChunkSize = DEFAULT_CHUNK_SIZE
	}

	servers[me.config.Port] = me
	return me
}

// ListenAndServe listens on the TCP network address and then calls Serve to handle incoming connections.
// Accepted connections are configured to enable TCP keep-alives.
func (me *Server) ListenAndServe() error {
	for i, loc := range me.config.Locations {
		if loc.Pattern == "" {
			loc.Pattern = "/"
		}
		if loc.Handler == "" {
			loc.Handler = "rtmp-live"
		}

		h := NewHandler(me, &me.config.Locations[i], me.factory)
		if h == nil {
			me.logger.Warnf("Handler \"%s\" not registered", loc.Handler)
			continue
		}

		me.mux.Handle(loc.Pattern, h)
	}

	me.logger.Infof("Listening on port %d", me.config.Port)

	l, err := net.Listen("tcp", fmt.Sprintf(":%d", me.config.Port))
	if err != nil {
		me.logger.Errorf("Failed to listen on port %d", me.config.Port)
		return err
	}

	return me.Serve(new(utils.TCPKeepAliveListener).Init(l, time.Duration(me.config.MaxIdleTime)*time.Second))
}

// Serve accepts incoming connections on the Listener l, creating a new service goroutine for each.
func (me *Server) Serve(l net.Listener) error {
	defer l.Close()

	d := 5 * time.Millisecond // How long to sleep on accept failure
	m := 1 * time.Second

	for {
		c, err := l.Accept()
		if err != nil {
			if ne, ok := err.(net.Error); ok && ne.Temporary() {
				me.logger.Warnf("Accept error: %v; retrying in %dms", err, d)

				time.Sleep(d)
				if d *= 2; d > m {
					d = m
				}
				continue
			}
			return err
		}

		// TODO(tonylau): whitelist & blacklist

		nc := new(NetConnection).Init(c, me, me.logger, me.factory)
		go nc.serve()
	}
}

// Accept an NetConnection.
func (me *Server) Accept(nc *NetConnection) {
	me.logger.Debugf(4, "Accepting connection: app=%s, inst=%s, id=%s", nc.AppName, nc.InstName, nc.FarID)

	me.mtx.Lock()
	defer me.mtx.Unlock()

	app, ok := me.applications[nc.AppName]
	if !ok {
		app = new(Application).Init(nc.AppName, me.logger, me.factory)
		me.applications[nc.AppName] = app
	}

	app.Add(nc)
	atomic.StoreUint32(&nc.readyState, STATE_CONNECTED)
}

// Reject an NetConnection.
func (me *Server) Reject(nc *NetConnection, description string) {
	nc.reply(command.ERROR, 1, Level.ERROR, Code.NETCONNECTION_CONNECT_REJECTED, description)
	nc.Close()
}

// GetStream returns a stream, creates if not exists.
func (me *Server) GetStream(appName string, instName string, name string) *Stream {
	me.mtx.Lock()
	defer me.mtx.Unlock()

	app, ok := me.applications[appName]
	if !ok {
		app = new(Application).Init(appName, me.logger, me.factory)
		me.applications[appName] = app
	}

	return app.GetStream(instName, name)
}

// FindStream returns an existing stream
func (me *Server) FindStream(appName string, instName string, name string) *Stream {
	me.mtx.RLock()
	defer me.mtx.RUnlock()

	app, ok := me.applications[appName]
	if ok {
		return app.FindStream(instName, name)
	}

	return nil
}

// GetServer returns the server listening on the port.
func GetServer(port int) *Server {
	return servers[port]
}
