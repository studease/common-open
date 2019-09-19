package rtmp

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"io"
	"math/rand"
	"net"
	"syscall"

	"github.com/studease/common/log"
)

// Static constants
const (
	_PACKET_SIZE = 1536
	_DIGEST_SIZE = 32
	_VERSION     = 0x5033029
	_MAX_BYTES   = 3073
)

// Handshaker states
const (
	HANDSHAKER_INITIALIZED uint32 = 0x00
	HANDSHAKER_CLOSING     uint32 = 0x01
	HANDSHAKER_CLOSED      uint32 = 0x02
)

var (
	_FP_KEY = []byte{
		0x47, 0x65, 0x6E, 0x75, 0x69, 0x6E, 0x65, 0x20,
		0x41, 0x64, 0x6F, 0x62, 0x65, 0x20, 0x46, 0x6C,
		0x61, 0x73, 0x68, 0x20, 0x50, 0x6C, 0x61, 0x79,
		0x65, 0x72, 0x20, 0x30, 0x30, 0x31, /* Genuine Adobe Flash Player 001 */
		0xF0, 0xEE, 0xC2, 0x4A, 0x80, 0x68, 0xBE, 0xE8,
		0x2E, 0x00, 0xD0, 0xD1, 0x02, 0x9E, 0x7E, 0x57,
		0x6E, 0xEC, 0x5D, 0x2D, 0x29, 0x80, 0x6F, 0xAB,
		0x93, 0xB8, 0xE6, 0x36, 0xCF, 0xEB, 0x31, 0xAE,
	}

	_FMS_KEY = []byte{
		0x47, 0x65, 0x6E, 0x75, 0x69, 0x6E, 0x65, 0x20,
		0x41, 0x64, 0x6F, 0x62, 0x65, 0x20, 0x46, 0x6C,
		0x61, 0x73, 0x68, 0x20, 0x4D, 0x65, 0x64, 0x69,
		0x61, 0x20, 0x53, 0x65, 0x72, 0x76, 0x65, 0x72,
		0x20, 0x30, 0x30, 0x31, // Genuine Adobe Flash Media Server 001
		0xF0, 0xEE, 0xC2, 0x4A, 0x80, 0x68, 0xBE, 0xE8,
		0x2E, 0x00, 0xD0, 0xD1, 0x02, 0x9E, 0x7E, 0x57,
		0x6E, 0xEC, 0x5D, 0x2D, 0x29, 0x80, 0x6F, 0xAB,
		0x93, 0xB8, 0xE6, 0x36, 0xCF, 0xEB, 0x31, 0xAE,
	}
)

// Handshaker provides methods to process handshake
type Handshaker struct {
	conn    net.Conn
	logger  log.ILogger
	state   byte
	buffer  bytes.Buffer
	temp    []byte
	bytesIn int
}

// Init this class
func (me *Handshaker) Init(conn net.Conn, logger log.ILogger) *Handshaker {
	me.conn = conn
	me.logger = logger
	me.state = 0
	me.temp = nil
	me.buffer.Reset()
	return me
}

func (me *Handshaker) serve() error {
	defer func() {
		if err := recover(); err != nil {
			me.logger.Errorf("Unexpected error occurred: %v", err)
		}
	}()

	m := 1 + _PACKET_SIZE
	b := make([]byte, m)

	for {
		if me.bytesIn+m > _MAX_BYTES {
			m = _MAX_BYTES - me.bytesIn
		}

		n, err := me.conn.Read(b[:m])
		if err != nil {
			me.logger.Debugf(4, "Failed to read: %v", err)
			return err
		}

		me.bytesIn += n

		err = me.processIncoming(b[:n])
		if err != nil {
			if err == io.EOF {
				break
			}

			me.logger.Errorf("Failed to process: %v", err)
			return err
		}
	}

	return nil
}

func (me *Handshaker) processIncoming(data []byte) error {
	const (
		sw_c0 byte = iota
		sw_c1
		sw_c2
	)

	var (
		size = len(data)
	)

	for i := 0; i < size; i++ {
		switch me.state {
		case sw_c0:
			if data[i] != 0x03 {
				return fmt.Errorf("invalid version: %d", data[0])
			}

			me.state = sw_c1

		case sw_c1:
			n := _PACKET_SIZE - me.buffer.Len()
			if n > size-i {
				n = size - i
			}

			_, err := me.buffer.Write(data[i : i+n])
			if err != nil {
				return err
			}

			i += n - 1

			if me.buffer.Len() == _PACKET_SIZE {
				err := me.processC1(me.buffer.Bytes())
				if err != nil {
					return err
				}

				me.state = sw_c2
				me.buffer.Reset()
			}

		case sw_c2:
			n := _PACKET_SIZE - me.buffer.Len()
			if n > size-i {
				n = size - i
			}

			_, err := me.buffer.Write(data[i : i+n])
			if err != nil {
				return err
			}

			i += n - 1

			if me.buffer.Len() == _PACKET_SIZE {
				err := me.processC2(me.buffer.Bytes())
				if err != nil {
					return err
				}

				me.buffer.Reset()

				// RTMP message slice
				if i+1 < size {
					me.buffer.Write(data[i+1:])
				}

				return io.EOF
			}

		default:
			fmt.Errorf("unrecognized state 0x%02X", me.state)
		}
	}

	return nil
}

func (me *Handshaker) processC1(c1 []byte) error {
	var (
		middle   bool
		c1Digest []byte
		err      error
	)

	i := 0
	n := 1 + _PACKET_SIZE
	b := make([]byte, n)

	complex := binary.BigEndian.Uint32(c1[4:8]) != 0

	// S0
	b[i] = 0x03
	i++

	// S1
	s1 := b[i:]

	binary.BigEndian.PutUint32(b[i:i+4], 0)
	i += 4

	binary.BigEndian.PutUint32(b[i:i+4], 0)
	i += 4

	for ; i < n; i++ {
		b[i] = byte(rand.Int() % 256)
	}

	if complex {
		c1Digest, _, err = me.validateClient(c1, &middle)
		if err != nil {
			me.logger.Errorf("Failed to validate c1: %v", err)
			return err
		}

		digestOffset, err := me.getDigestOffset(s1, middle)
		if err != nil {
			me.logger.Errorf("Failed to get digest offset of s1: %v", err)
			return err
		}

		s1Random := make([]byte, _PACKET_SIZE-_DIGEST_SIZE)
		copy(s1Random, s1[:digestOffset])
		copy(s1Random[digestOffset:], s1[digestOffset+_DIGEST_SIZE:])

		s1Hash := hmac.New(sha256.New, _FMS_KEY[:36])
		s1Hash.Write(s1Random)

		s1Digest := s1Hash.Sum(nil)
		copy(s1[digestOffset:digestOffset+_DIGEST_SIZE], s1Digest)
	}

	_, err = me.conn.Write(b)
	if err != nil {
		me.logger.Errorf("Failed to write s0, s1: %v", err)
		return err
	}

	me.temp = s1

	// S2
	s2 := c1

	if complex {
		n = _PACKET_SIZE - _DIGEST_SIZE
		tmp := make([]byte, n)

		for i := 0; i < n; i++ {
			tmp[i] = byte(rand.Int() % 256)
		}

		s2Hash := hmac.New(sha256.New, _FMS_KEY[:68])
		s2Hash.Write(c1Digest)

		s2Digest := s2Hash.Sum(nil)

		s2Hash = hmac.New(sha256.New, s2Digest)
		s2Hash.Write(tmp)

		s2 = s2Hash.Sum(tmp)
	}

	_, err = me.conn.Write(s2)
	if err != nil {
		me.logger.Errorf("Failed to write s2: %v", err)
		return err
	}

	return nil
}

func (me *Handshaker) processC2(c2 []byte) error {
	s1 := me.temp

	if len(s1) != _PACKET_SIZE || len(c2) != _PACKET_SIZE {
		return fmt.Errorf("packet length not match")
	}

	for i := 0; i < _PACKET_SIZE; i++ {
		if c2[i] != s1[i] {
			return fmt.Errorf("c2 & s1 not match")
		}
	}

	return nil
}

func (me *Handshaker) validateClient(c1 []byte, middle *bool) ([]byte, []byte, error) {
	digest, challenge, err := me.validateClientScheme(c1, true)
	if err == nil {
		*middle = true
		return digest, challenge, nil
	}

	digest, challenge, err = me.validateClientScheme(c1, false)
	if err == nil {
		*middle = false
		return digest, challenge, nil
	}

	return nil, nil, fmt.Errorf("unknown scheme")
}

func (me *Handshaker) validateClientScheme(c1 []byte, middle bool) ([]byte, []byte, error) {
	digestOffset, err := me.getDigestOffset(c1, middle)
	if err != nil {
		return nil, nil, err
	}

	digest := make([]byte, _DIGEST_SIZE)
	copy(digest, c1[digestOffset:digestOffset+_DIGEST_SIZE])

	random := make([]byte, _PACKET_SIZE-_DIGEST_SIZE)
	copy(random, c1[:digestOffset])
	copy(random[digestOffset:], c1[digestOffset+_DIGEST_SIZE:])

	hash := hmac.New(sha256.New, _FP_KEY[:30])
	hash.Write(random)

	tmp := hash.Sum(nil)
	if bytes.Compare(tmp, digest) != 0 {
		return nil, nil, syscall.EINVAL
	}

	challengeOffset, err := me.getDHOffset(c1, middle)
	if err != nil {
		return nil, nil, err
	}

	challenge := make([]byte, 128)
	copy(challenge, c1[challengeOffset:challengeOffset+128])

	return digest, challenge, nil
}

func (me *Handshaker) getDigestOffset(c1 []byte, middle bool) (int, error) {
	offset := 8 + 4
	if middle {
		offset += 764
	}

	offset += (int(c1[offset-4]) + int(c1[offset-3]) + int(c1[offset-2]) + int(c1[offset-1])) % 728
	if offset+_DIGEST_SIZE > _PACKET_SIZE {
		return 0, fmt.Errorf("%d out of range", offset)
	}

	return offset, nil
}

func (me *Handshaker) getDHOffset(c1 []byte, middle bool) (int, error) {
	offset := 8 + 764
	if middle == false {
		offset += 764
	}

	offset = ((int(c1[offset-4]) + int(c1[offset-3]) + int(c1[offset-2]) + int(c1[offset-1])) % 632) + 8
	if middle == false {
		offset += 764
	}

	if offset+128 > _PACKET_SIZE {
		return 0, fmt.Errorf("DH offset %d out of range", offset)
	}

	return offset, nil
}

// Establish a RTMP handshake
func (me *Handshaker) Establish() error {
	defer func() {
		if err := recover(); err != nil {
			me.logger.Errorf("Unexpected error occurred: %v", err)
		}
	}()

	i := 0
	n := 1 + _PACKET_SIZE
	b := make([]byte, n)

	// C0
	b[i] = 0x03
	i++

	// C1
	c1 := b[i:]

	binary.BigEndian.PutUint32(b[i:i+4], 0)
	i += 4

	binary.BigEndian.PutUint32(b[i:i+4], 0)
	i += 4

	for ; i < n; i++ {
		b[i] = byte(rand.Int() % 256)
	}

	_, err := me.conn.Write(b)
	if err != nil {
		me.logger.Errorf("Failed to write c0, c1: %v", err)
		return err
	}

	me.temp = c1

	return me.read()
}

func (me *Handshaker) read() error {
	b := make([]byte, 1+_PACKET_SIZE)

	for {
		n, err := me.conn.Read(b)
		if err != nil {
			me.logger.Debugf(4, "Failed to read: %v", err)
			return err
		}

		err = me.processOutcoming(b[:n])
		if err != nil {
			if err == io.EOF {
				break
			}

			me.logger.Errorf("Failed to process: %v", err)
			return err
		}
	}

	return nil
}

func (me *Handshaker) processOutcoming(data []byte) error {
	const (
		sw_s0 byte = iota
		sw_s1
		sw_s2
	)

	var (
		size = len(data)
	)

	for i := 0; i < size; i++ {
		switch me.state {
		case sw_s0:
			if data[i] != 0x03 {
				return fmt.Errorf("invalid version: %d", data[0])
			}

			me.state = sw_s1

		case sw_s1:
			n := _PACKET_SIZE - me.buffer.Len()
			if n > size-i {
				n = size - i
			}

			_, err := me.buffer.Write(data[i : i+n])
			if err != nil {
				return err
			}

			i += n - 1

			if me.buffer.Len() == _PACKET_SIZE {
				err := me.processS1(me.buffer.Bytes())
				if err != nil {
					return err
				}

				me.state = sw_s2
				me.buffer.Reset()
			}

		case sw_s2:
			n := _PACKET_SIZE - me.buffer.Len()
			if n > size-i {
				n = size - i
			}

			_, err := me.buffer.Write(data[i : i+n])
			if err != nil {
				return err
			}

			i += n - 1

			if me.buffer.Len() == _PACKET_SIZE {
				err := me.processS2(me.buffer.Bytes())
				if err != nil {
					return err
				}

				me.buffer.Reset()

				// RTMP message slice
				if i+1 < size {
					me.buffer.Write(data[i+1:])
				}

				return io.EOF
			}

		default:
			fmt.Errorf("unrecognized state 0x%02X", me.state)
		}
	}

	return nil
}

func (me *Handshaker) processS1(s1 []byte) error {
	// C2
	_, err := me.conn.Write(s1)
	if err != nil {
		me.logger.Errorf("Failed to write c2: %v", err)
		return err
	}

	return nil
}

func (me *Handshaker) processS2(s2 []byte) error {
	c1 := me.temp

	if len(c1) != _PACKET_SIZE || len(s2) != _PACKET_SIZE {
		return fmt.Errorf("packet length not match")
	}

	for i := 0; i < _PACKET_SIZE; i++ {
		if s2[i] != c1[i] {
			return fmt.Errorf("s2 & c1 not match")
		}
	}

	return nil
}
