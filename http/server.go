package http

import (
	"fmt"
	"net/http"

	httpcfg "github.com/studease/common/http/config"
	"github.com/studease/common/log"
)

// Static constants
const (
	DEFAULT_PORT             = 80
	DEFAULT_TIMEOUT          = 10
	DEFAULT_MAX_IDLE_TIME    = 65
	DEFAULT_SEND_BUFFER_SIZE = 65536
	DEFAULT_READ_BUFFER_SIZE = 65536
	DEFAULT_ROOT             = "webroot"
	DEFAULT_CORS             = "webroot/crossdomain.xml"
)

// Server defines parameters for running an HTTP server
type Server struct {
	CFG     *httpcfg.Server
	logger  log.ILogger
	factory log.ILoggerFactory
}

// Init this class
func (me *Server) Init(cfg *httpcfg.Server, logger log.ILogger, factory log.ILoggerFactory) *Server {
	me.CFG = cfg
	me.logger = logger
	me.factory = factory

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

	return me
}

// ListenAndServe listens on the TCP network address and then calls Serve to handle incoming connections.
// Accepted connections are configured to enable TCP keep-alives.
func (me *Server) ListenAndServe() error {
	defer func() {
		if err := recover(); err != nil {
			me.logger.Errorf("Unexpected error occurred: %v", err)
		}
	}()

	for i, loc := range me.CFG.Locations {
		if loc.Pattern == "" {
			loc.Pattern = "/"
		}
		if loc.Handler == "" {
			loc.Handler = "http-file"
		}

		h := NewHandler(me, &me.CFG.Locations[i], me.factory)
		if h == nil {
			me.logger.Warnf("Handler \"%s\" not registered", loc.Handler)
			continue
		}

		http.HandleFunc(loc.Pattern, h.ServeHTTP)
	}

	if me.CFG.SSL.Enable {
		me.logger.Infof("Listening on port %d", 443)

		go func() {
			err := http.ListenAndServeTLS(":443", me.CFG.SSL.Cert, me.CFG.SSL.Key, nil)
			if err != nil {
				me.logger.Errorf("Failed to listen on port %d", 443)
			}
		}()
	}

	me.logger.Infof("Listening on port %d", me.CFG.Port)

	err := http.ListenAndServe(fmt.Sprintf(":%d", me.CFG.Port), nil)
	if err != nil {
		me.logger.Errorf("Failed to listen on port %d", me.CFG.Port)
		return err
	}

	return nil
}

// CheckOrigin returns true if the request Origin header is acceptable
func (me *Server) CheckOrigin(r *http.Request) bool {
	origin := r.Header.Get("Origin")

	if origin == "" {
		return true
	}

	// TODO: Check CORS

	return true
}
