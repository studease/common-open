package mediarecorder

import (
	"github.com/studease/common/av"
	"github.com/studease/common/log"
	"github.com/studease/common/utils"
)

// States.
const (
	StateInactive  uint32 = 0x00
	StateRecording uint32 = 0x01
	StatePaused    uint32 = 0x02
)

var (
	r = utils.NewRegister()
)

// Register an IMediaRecorder with the given name.
func Register(name string, recorder interface{}) {
	r.Add(name, recorder)
}

// New creates a registered IMediaRecorder by the name.
func New(name string, constraints *av.MediaRecorderConstraints, factory log.ILoggerFactory) av.IMediaRecorder {
	if recorder := r.New(name); recorder != nil {
		return recorder.(av.IMediaRecorder).Init(constraints, factory.NewLogger(name))
	}
	return nil
}
