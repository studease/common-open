package http

import (
	"net/http"
	"strings"

	httpcfg "github.com/studease/common/http/config"
	"github.com/studease/common/log"
	"github.com/studease/common/utils"
)

// Static constants
var (
	SERVER = ""
)

var (
	r = utils.NewRegister()
)

// IHandler responds to an HTTP request
type IHandler interface {
	Init(srv *Server, cfg *httpcfg.Location, logger log.ILogger, factory log.ILoggerFactory) IHandler
	ServeHTTP(w http.ResponseWriter, r *http.Request)
}

// Register an IHandler with the given name
func Register(name string, handler interface{}) {
	r.Add(name, handler)
}

// NewHandler creates a registered IHandler by the name
func NewHandler(srv *Server, loc *httpcfg.Location, factory log.ILoggerFactory) IHandler {
	if h := r.New(loc.Handler); h != nil {
		scope := strings.TrimPrefix(loc.Handler, "http-")
		scope = strings.TrimPrefix(scope, "ws-")
		scope = strings.TrimPrefix(scope, "chat-")
		scope = strings.TrimPrefix(scope, "rtmp-")
		return h.(IHandler).Init(srv, loc, factory.NewLogger(scope), factory)
	}

	return nil
}
