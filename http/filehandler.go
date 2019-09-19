package http

import (
	"net/http"

	httpcfg "github.com/studease/common/http/config"
	"github.com/studease/common/log"
	"github.com/studease/common/utils/bw"
)

func init() {
	Register("http-file", FileHandler{})
}

// FileHandler provides static resources service
type FileHandler struct {
	bw.Manager

	srv     *Server
	cfg     *httpcfg.Location
	logger  log.ILogger
	factory log.ILoggerFactory
}

// Init this class
func (me *FileHandler) Init(srv *Server, cfg *httpcfg.Location, logger log.ILogger, factory log.ILoggerFactory) IHandler {
	me.Manager.Init(cfg.Limitation.Upload, 0)
	me.srv = srv
	me.cfg = cfg
	me.logger = logger
	me.factory = factory
	return me
}

// ServeHTTP handles the request
func (me *FileHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Server", SERVER)

	defer func() {
		if err := recover(); err != nil {
			me.logger.Errorf("Unexpected error occurred: %v", err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		}
	}()

	if !me.srv.CheckOrigin(r) {
		http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
		return
	}

	origin := r.Header.Get("Origin")
	if origin != "" {
		w.Header().Set("Access-Control-Allow-Origin", "*")
	}

	if r.Method == http.MethodOptions {
		w.Header().Set("Access-Control-Allow-Methods", r.Header.Get("Access-Control-Request-Method"))
		w.Header().Set("Access-Control-Allow-Headers", r.Header.Get("Access-Control-Request-Headers"))
		w.WriteHeader(http.StatusOK)
		return
	}

	if r.Method != http.MethodGet {
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}

	me.logger.Debugf(0, "%s", r.URL.Path)
	http.ServeFile(w, r, me.srv.CFG.Root+r.URL.Path)
}
