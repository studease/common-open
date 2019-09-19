package dvr

import (
	"github.com/studease/common/av"
	"github.com/studease/common/log"
	"github.com/studease/common/utils"
	basecfg "github.com/studease/common/utils/config"
)

// DVR types
const (
	TYPE_FLV  = "FLV"
	TYPE_FMP4 = "FMP4"
)

var (
	r = utils.NewRegister()
)

// IDVR defines methods to record an IReadableStream
type IDVR interface {
	Init(cfg *basecfg.DVR, logger log.ILogger, factory log.ILoggerFactory) IDVR
	Attach(stream av.IReadableStream)
	CloseNotify() <-chan bool
	Close()
}

// Register an IDVR with the given name
func Register(name string, dvr interface{}) {
	r.Add(name, dvr)
}

// New creates a registered IDVR by the name
func New(name string, cfg *basecfg.DVR, factory log.ILoggerFactory) IDVR {
	if m := r.New(name); m != nil {
		return m.(IDVR).Init(cfg, factory.NewLogger(name), factory)
	}

	return nil
}
