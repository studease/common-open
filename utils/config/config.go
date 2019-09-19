package config

// Server config of target
type Server struct {
	Name        string `xml:"name,attr"`
	Weight      int    `xml:"weight,attr"`
	Timeout     int    `xml:"timeout,attr"`
	MaxFailures int    `xml:"maxFailures,attr"`
	Mask        string `xml:"mask,attr"`
	HostPort    string `xml:",innerxml"`
	Failures    int32
}

// Listener config
type Listener struct {
	Port           int    `xml:"Listen"`
	Timeout        int    `xml:""`
	MaxIdleTime    int    `xml:""`
	SendBufferSize int    `xml:""`
	ReadBufferSize int    `xml:""`
	Root           string `xml:""`
	Cors           string `xml:""`
}

// URL config
type URL struct {
	Enable bool   `xml:"enable,attr"`
	Method string `xml:"method,attr"`
	Path   string `xml:",innerxml"`
}

// Query config
type Query struct {
	Enable  bool   `xml:"enable,attr"`
	Name    string `xml:""`
	File    string `xml:""`
	History int    `xml:""`
}

// DVR config
type DVR struct {
	ID          string `xml:"id,attr"`
	Name        string `xml:""`
	Mode        string `xml:""`
	Directory   string `xml:""`
	FileName    string `xml:""`
	Seekable    bool   `xml:""`
	Unique      bool   `xml:""`
	Append      bool   `xml:""`
	MaxDuration uint32 `xml:""`
	MaxSize     int64  `xml:""`
	MaxFrames   int64  `xml:""`

	OnRecord     URL `xml:""`
	OnRecordDone URL `xml:""`
}
