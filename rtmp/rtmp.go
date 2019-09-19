package rtmp

import (
	"strings"

	"github.com/studease/common/log"
	rtmpcfg "github.com/studease/common/rtmp/config"
	"github.com/studease/common/rtmp/message"
	"github.com/studease/common/utils"
)

// Object encodings
const (
	AMF0 byte = 0
	AMF3 byte = 3
)

var (
	r = utils.NewRegister()
)

// INetStream defines methods to handle rtmp massages
type INetStream interface {
	setID(id uint32)
	setBufferLength(n uint32)
	process(ck *message.Message) error
	Close() error
}

// IHandler processes RTMP messages
type IHandler interface {
	Init(srv *Server, cfg *rtmpcfg.Location, logger log.ILogger, factory log.ILoggerFactory) IHandler
	ServeRTMP(nc *NetConnection) error
}

// Register an IHandler with the given name
func Register(name string, handler interface{}) {
	r.Add(name, handler)
}

// NewHandler creates a registered IHandler by the name
func NewHandler(srv *Server, loc *rtmpcfg.Location, factory log.ILoggerFactory) IHandler {
	if h := r.New(loc.Handler); h != nil {
		return h.(IHandler).Init(srv, loc, factory.NewLogger(strings.TrimPrefix(loc.Handler, "rtmp-")), factory)
	}

	return nil
}
