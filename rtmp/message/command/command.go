package command

// RTMP commands
const (
	CONNECT       = "connect"
	CLOSE         = "close"
	CREATE_STREAM = "createStream"
	RESULT        = "_result"
	ERROR         = "_error"

	PLAY          = "play"
	PLAY2         = "play2"
	DELETE_STREAM = "deleteStream"
	CLOSE_STREAM  = "closeStream"
	RECEIVE_AUDIO = "receiveAudio"
	RECEIVE_VIDEO = "receiveVideo"
	PUBLISH       = "publish"
	FC_UNPUBLISH  = "FCUnpublish"
	SEEK          = "seek"
	PAUSE         = "pause"
	ON_STATUS     = "onStatus"

	CHECK_BANDWIDTH = "checkBandwidth"
	GET_STATS       = "getStats"
)
