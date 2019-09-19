package av

// Context is used as the base class for the creation of MediaContext objects
type Context struct {
	info     *Information
	MimeType string
	Codecs   string

	// Fields for parsing
	FrameType   byte // 0xF0
	Codec       byte // 0x0F
	Format      byte // 1111 0000
	SampleRate  byte // 0000 1100
	SampleSize  byte // 0000 0010
	SampleType  byte // 0000 0001
	DataType    byte
	Keyframe    bool
	CTS         uint32
	DTS         uint32
	PTS         uint32
	Duration    uint32
	ExpectedDts uint32
	Flags       struct {
		IsLeading           byte
		SampleDependsOn     byte
		SampleIsDependedOn  byte
		SampleHasRedundancy byte
		IsNonSync           byte
	}

	// Raw Frame
	Data []byte
}

// Init this class
func (me *Context) Init(info *Information) *Context {
	me.info = info
	return me
}

// Information returns the associated Information
func (me *Context) Information() *Information {
	return me.info
}

// Basic returns the Context
func (me *Context) Basic() *Context {
	return me
}
