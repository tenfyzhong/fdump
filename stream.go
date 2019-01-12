package fdump

import (
	"errors"
	"sync"
	"time"

	"github.com/google/gopacket"
	"github.com/google/gopacket/tcpassembly"
)

var (
	// ErrPkgNoEnough packet no enough error. return this error in decode
	// function if the packet no enough.
	ErrPkgNoEnough = errors.New("pkg no enough")
)

// DecodeFunc Decode the packet when receive a packet.
// Return the decoded bodies and used bytes. It will ignore the bodies if it's
// empty.
type DecodeFunc func(gopacket.Flow, gopacket.Flow, []byte) (bodies []interface{}, n int, err error)

type streamFactory struct {
	msgChan    chan *Record
	mutex      sync.Mutex
	cmds       map[uint16]bool
	serverPort int
	decodeFunc DecodeFunc
}

func newStreamFactory(msgChan chan *Record, decodeFunc DecodeFunc) *streamFactory {
	f := &streamFactory{
		msgChan:    msgChan,
		decodeFunc: decodeFunc,
	}
	return f
}

func (factory *streamFactory) New(net, transport gopacket.Flow) tcpassembly.Stream {
	log.Infof("new stream, new: %+v, transport: %+v", net, transport)
	s := &stream{
		net:       net,
		transport: transport,
		buf:       make([]byte, 0),
		factory:   factory,
	}
	return s
}

type stream struct {
	net       gopacket.Flow
	transport gopacket.Flow
	buf       []byte
	seen      time.Time
	factory   *streamFactory
}

func (s stream) Net() gopacket.Flow {
	return s.net
}

func (s stream) Transport() gopacket.Flow {
	return s.transport
}

// Reassembled is called whenever new packet data is available for reading.
// Reassembly objects contain stream data IN ORDER.
func (s *stream) Reassembled(reassemblies []tcpassembly.Reassembly) {
	log.Debugf("ressembled len: %d", len(reassemblies))
	for _, r := range reassemblies {
		s.buf = append(s.buf, r.Bytes...)
		for {
			bodies, n, err := s.factory.decodeFunc(s.net, s.transport, s.buf)
			if err != nil {
				log.Debugf("unpack err: %+v", err)
				break
			}

			usedBuf := s.buf[:n]
			s.buf = s.buf[n:]

			if len(bodies) == 0 {
				log.Debugf("body is empty")
				continue
			}

			m := &Record{
				Bodies:    bodies,
				Net:       s.net,
				Transport: s.transport,
				Seen:      time.Now(),
				Buffer:    usedBuf,
			}
			s.factory.msgChan <- m
		}
	}
}

// ReassemblyComplete is called when the TCP assembler believes a stream has
// finished.
func (s *stream) ReassemblyComplete() {
	log.Infof("reassembly complete, net: %+v, transport: %+v", s.net, s.transport)
}
