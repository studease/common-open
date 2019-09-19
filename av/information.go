package av

// Information is used as the mediator of Context objects
type Information struct {
	MimeType          string
	Codecs            string
	Timescale         uint32
	TimeBase          uint32
	Timestamp         uint32
	Duration          uint32
	Size              int64
	RefSampleDuration uint32
	Width             uint32
	Height            uint32
	CodecWidth        uint32
	CodecHeight       uint32
	AudioDataRate     uint32
	VideoDataRate     uint32
	BitRate           uint32
	FrameRate         Rational
	SampleRate        uint32
	SampleSize        uint32
	Channels          uint32
}

// Init this class
func (me *Information) Init() *Information {
	me.Timescale = 1000
	me.FrameRate.Init()
	return me
}

// Rational is used to define rational numbers
type Rational struct {
	Num float64 // Numerator
	Den float64 // Denominator
}

// Init this class
func (me *Rational) Init() *Rational {
	me.Num = 30
	me.Den = 1
	return me
}
