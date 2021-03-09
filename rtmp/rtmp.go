package rtmp

import (
	"github.com/studease/common/log"
	rtmpcfg "github.com/studease/common/rtmp/config"
	"github.com/studease/common/utils"
)

// Static constants.
var (
	SERVER = ""
)

var (
	r = utils.NewRegister()
)

// IHandler serves rtmp connection.
type IHandler interface {
	Init(srv *Server, cfg *rtmpcfg.Location, logger log.ILogger, factory log.ILoggerFactory) IHandler
	ServeRTMP(nc *NetConnection) error
}

// Register an IHandler with the given name.
func Register(name string, handler interface{}) {
	r.Add(name, handler)
}

// NewHandler creates a registered IHandler by the name.
func NewHandler(srv *Server, loc *rtmpcfg.Location, factory log.ILoggerFactory) IHandler {
	if h := r.New(loc.Handler); h != nil {
		return h.(IHandler).Init(srv, loc, factory.NewLogger(loc.Handler), factory)
	}
	return nil
}
