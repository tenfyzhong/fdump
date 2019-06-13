package proto

import (
	"bytes"
	"encoding/binary"
	"encoding/gob"
	"errors"
)

// HeaderLength header packet length
const HeaderLength = 25

// errors
var (
	ErrHeaderIsNil = errors.New("heaser is nil")
	ErrPkgNoEnough = errors.New("pkg no enough")
)

// EncodeHeader encode the header use network byte order
func EncodeHeader(header *Header) ([]byte, error) {
	if header == nil {
		return nil, ErrHeaderIsNil
	}

	currentIndex := 0
	pkg := make([]byte, HeaderLength)
	binary.BigEndian.PutUint32(pkg[currentIndex:], header.Len)
	currentIndex += 4
	binary.BigEndian.PutUint32(pkg[currentIndex:], header.Command)
	currentIndex += 4
	binary.BigEndian.PutUint64(pkg[currentIndex:], header.Maybe0)
	currentIndex += 8
	binary.BigEndian.PutUint32(pkg[currentIndex:], header.Maybe1)
	currentIndex += 4
	binary.BigEndian.PutUint32(pkg[currentIndex:], header.Result)
	currentIndex += 4
	pkg[currentIndex] = header.IsRequest
	return pkg, nil
}

// DecodeHeader decode the network byte order packet to Header
func DecodeHeader(pkg []byte) (*Header, error) {
	if len(pkg) < HeaderLength {
		return nil, ErrPkgNoEnough
	}

	header := &Header{}
	currentIndex := 0
	header.Len = binary.BigEndian.Uint32(pkg[currentIndex:])
	currentIndex += 4
	header.Command = binary.BigEndian.Uint32(pkg[currentIndex:])
	currentIndex += 4
	header.Maybe0 = binary.BigEndian.Uint64(pkg[currentIndex:])
	currentIndex += 8
	header.Maybe1 = binary.BigEndian.Uint32(pkg[currentIndex:])
	currentIndex += 4
	header.Result = binary.BigEndian.Uint32(pkg[currentIndex:])
	currentIndex += 4
	header.IsRequest = pkg[currentIndex]
	return header, nil
}

// EncodeBody encode body
func EncodeBody(i interface{}) ([]byte, error) {
	if i == nil {
		return make([]byte, 0, 0), nil
	}
	var buffer bytes.Buffer
	enc := gob.NewEncoder(&buffer)
	err := enc.Encode(i)
	if err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}

// DecodeBody decode body
func DecodeBody(body []byte, i interface{}) error {
	if len(body) == 0 {
		return nil
	}
	if i == nil {
		return errors.New("i is nil")
	}

	buffer := bytes.NewBuffer(body)
	dec := gob.NewDecoder(buffer)
	err := dec.Decode(i)
	if err != nil {
		return err
	}

	return nil
}

// Encode encode header and object to buffer
func Encode(header *Header, i interface{}) ([]byte, error) {
	rspBody, err := EncodeBody(i)
	if err != nil {
		return nil, err
	}

	header.Len = uint32(HeaderLength + len(rspBody))
	headerBuf, err := EncodeHeader(header)
	if err != nil {
		return nil, err
	}

	buf := make([]byte, int(header.Len))
	copy(buf, headerBuf)
	copy(buf[HeaderLength:], rspBody)

	return buf, nil
}
