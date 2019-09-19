package codec

import (
	"github.com/studease/common/av"
	"github.com/studease/common/log"
	"github.com/studease/common/utils"
)

// Codec IDs
const (
	NONE av.Codec = iota
	AAC
	AVC
)

var (
	r = utils.NewRegister()

	labels = map[av.Codec]string{
		AAC: "AAC",
		AVC: "AVC",
	}
)

// Register an IMediaContext with the given codec
func Register(codec av.Codec, ctx interface{}) {
	if label := labels[codec]; label != "" {
		r.Add(label, ctx)
	}
}

// New creates a registered IMediaContext by the codec
func New(codec av.Codec, info *av.Information, factory log.ILoggerFactory) av.IMediaContext {
	label := labels[codec]

	if c := r.New(label); c != nil {
		return c.(av.IMediaContext).Init(info, factory.NewLogger(label))
	}

	return nil
}
