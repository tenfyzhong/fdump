package proto

import (
	"errors"
	"io"
	"net"
	"time"
)

// Write write header and i to conn
func Write(conn net.Conn, header *Header, i interface{}) error {
	if conn == nil || header == nil {
		return errors.New("conn or header is nil")
	}

	buffer, err := Encode(header, i)
	if err != nil {
		return err
	}

	writedN := 0
	for writedN < len(buffer) {
		err := conn.SetWriteDeadline(time.Now().Add(1 * time.Millisecond))
		if err != nil {
			return err
		}
		n, err := conn.Write(buffer[writedN:])
		if err != nil && err != io.EOF {
			return err
		}
		writedN += n
	}

	return nil
}

// Read read header and body
func Read(c net.Conn) (header *Header, body []byte, err error) {
	headerBuf := make([]byte, HeaderLength)
	headerReadedN := 0
	for headerReadedN < HeaderLength {
		n, e := c.Read(headerBuf[headerReadedN:])
		if e != nil && e != io.EOF {
			err = e
			return
		}
		headerReadedN += n
	}

	h, e := DecodeHeader(headerBuf)
	if e != nil {
		err = e
		return
	}
	header = h

	if header.Len < uint32(headerReadedN) {
		err = errors.New("pkg error")
		return
	}

	bodyLen := header.Len - uint32(headerReadedN)
	bodyReadedN := uint32(0)
	body = make([]byte, bodyLen)
	for bodyReadedN < bodyLen {
		n, e := c.Read(body[bodyReadedN:])
		if e != nil {
			err = e
		}
		bodyReadedN += uint32(n)
	}

	return
}
