package amf

// AMF types
const (
	DOUBLE        byte = 0x00
	BOOLEAN       byte = 0x01
	STRING        byte = 0x02
	OBJECT        byte = 0x03
	MOVIE_CLIP    byte = 0x04 // Not available in Remoting
	NULL          byte = 0x05
	UNDEFINED     byte = 0x06
	REFERENCE     byte = 0x07
	ECMA_ARRAY    byte = 0x08
	END_OF_OBJECT byte = 0x09
	STRICT_ARRAY  byte = 0x0A
	DATE          byte = 0x0B
	LONG_STRING   byte = 0x0C
	UNSUPPORTED   byte = 0x0D
	RECORD_SET    byte = 0x0E // Remoting server-to-client only
	XML           byte = 0x0F
	TYPED_OBJECT  byte = 0x10 // Class instance
	AMF3_DATA     byte = 0x11 // Sent by Flash player 9+
)
