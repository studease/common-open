package eventtype

// User control event types
const (
	STREAM_BEGIN       uint16 = 0
	STREAM_EOF         uint16 = 1
	STREAM_DRY         uint16 = 2
	SET_BUFFER_LENGTH  uint16 = 3
	STREAM_IS_RECORDED uint16 = 4
	PING_REQUEST       uint16 = 6
	PING_RESPONSE      uint16 = 7
	BUFFER_EMPTY       uint16 = 31
	BUFFER_READY       uint16 = 32
)
