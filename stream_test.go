package fdump

import (
	"net"
	"reflect"
	"testing"
	"time"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/tcpassembly"
	"github.com/stretchr/testify/assert"
)

func testDecodeFunc(net, transport gopacket.Flow, data []byte) (bodies []interface{}, n int, err error) {
	if len(data) < 10 {
		err = ErrPkgNoEnough
		return
	}
	n = 10
	result := data[:10]
	ignored := []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0}
	if reflect.DeepEqual(result, ignored) {
		return
	}

	bodies = append(bodies, string(result))
	return
}

func TestNewStream(t *testing.T) {
	msgChan := make(chan *Record, 1)
	f := newStreamFactory(msgChan, testDecodeFunc)

	netSrc := layers.NewIPEndpoint(net.ParseIP("127.0.0.1"))
	netDst := layers.NewIPEndpoint(net.ParseIP("10.2.2.2"))
	transportSrc := layers.NewTCPPortEndpoint(50123)
	transportDst := layers.NewTCPPortEndpoint(20001)
	net, err := gopacket.FlowFromEndpoints(netSrc, netDst)
	assert.NoError(t, err)
	transport, err := gopacket.FlowFromEndpoints(transportSrc, transportDst)
	assert.NoError(t, err)

	ts := f.New(net, transport)
	s, ok := ts.(*stream)
	assert.True(t, ok)
	assert.Equal(t, net, s.net)
	assert.Equal(t, transport, s.transport)
	assert.Equal(t, f, s.factory)

	assert.Equal(t, net, s.Net())
	assert.Equal(t, transport, s.Transport())
}

func TestReassembledEmpty(t *testing.T) {
	msgChan := make(chan *Record, 100)
	f := newStreamFactory(msgChan, testDecodeFunc)

	netSrc := layers.NewIPEndpoint(net.ParseIP("127.0.0.1"))
	netDst := layers.NewIPEndpoint(net.ParseIP("10.2.2.2"))
	transportSrc := layers.NewTCPPortEndpoint(50123)
	transportDst := layers.NewTCPPortEndpoint(20001)
	net, err := gopacket.FlowFromEndpoints(netSrc, netDst)
	assert.NoError(t, err)
	transport, err := gopacket.FlowFromEndpoints(transportSrc, transportDst)
	assert.NoError(t, err)

	ts := f.New(net, transport)
	s, ok := ts.(*stream)
	assert.True(t, ok)

	reassembly := tcpassembly.Reassembly{
		Bytes: []byte{},
		Seen:  time.Now(),
	}

	s.Reassembled([]tcpassembly.Reassembly{reassembly})

	select {
	case <-msgChan:
		assert.FailNow(t, "error")
	default:
	}

	assert.Empty(t, s.buf)
}

func TestReassembled1Object(t *testing.T) {
	msgChan := make(chan *Record, 100)
	f := newStreamFactory(msgChan, testDecodeFunc)

	netSrc := layers.NewIPEndpoint(net.ParseIP("127.0.0.1"))
	netDst := layers.NewIPEndpoint(net.ParseIP("10.2.2.2"))
	transportSrc := layers.NewTCPPortEndpoint(50123)
	transportDst := layers.NewTCPPortEndpoint(20001)
	net, err := gopacket.FlowFromEndpoints(netSrc, netDst)
	assert.NoError(t, err)
	transport, err := gopacket.FlowFromEndpoints(transportSrc, transportDst)
	assert.NoError(t, err)

	ts := f.New(net, transport)
	s, ok := ts.(*stream)
	assert.True(t, ok)

	reassembly := tcpassembly.Reassembly{
		Bytes: []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10},
		Seen:  time.Now(),
	}

	s.Reassembled([]tcpassembly.Reassembly{reassembly})

	select {
	case r := <-msgChan:
		assert.Equal(t, RecordType(RecordTypeTCP), r.Type)
		assert.Equal(t, net, r.Net)
		assert.Equal(t, transport, r.Transport)
		assert.Equal(t, reassembly.Bytes, r.Buffer)
		assert.Equal(t, 1, len(r.Bodies))
		str, ok := r.Bodies[0].(string)
		assert.True(t, ok)
		assert.Equal(t, reassembly.Bytes, []byte(str))
		assert.True(t, r.Seen.UnixNano() > reassembly.Seen.UnixNano())
	default:
		assert.FailNow(t, "consume chan failed")
	}

	assert.Empty(t, s.buf)
}

func TestReassembledIgnore(t *testing.T) {
	msgChan := make(chan *Record, 100)
	f := newStreamFactory(msgChan, testDecodeFunc)

	netSrc := layers.NewIPEndpoint(net.ParseIP("127.0.0.1"))
	netDst := layers.NewIPEndpoint(net.ParseIP("10.2.2.2"))
	transportSrc := layers.NewTCPPortEndpoint(50123)
	transportDst := layers.NewTCPPortEndpoint(20001)
	net, err := gopacket.FlowFromEndpoints(netSrc, netDst)
	assert.NoError(t, err)
	transport, err := gopacket.FlowFromEndpoints(transportSrc, transportDst)
	assert.NoError(t, err)

	ts := f.New(net, transport)
	s, ok := ts.(*stream)
	assert.True(t, ok)

	reassembly := tcpassembly.Reassembly{
		Bytes: []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
		Seen:  time.Now(),
	}

	s.Reassembled([]tcpassembly.Reassembly{reassembly})

	select {
	case <-msgChan:
		assert.FailNow(t, "error")
	default:
	}

	assert.Empty(t, s.buf)
}

func TestReassembledLess10(t *testing.T) {
	msgChan := make(chan *Record, 100)
	f := newStreamFactory(msgChan, testDecodeFunc)

	netSrc := layers.NewIPEndpoint(net.ParseIP("127.0.0.1"))
	netDst := layers.NewIPEndpoint(net.ParseIP("10.2.2.2"))
	transportSrc := layers.NewTCPPortEndpoint(50123)
	transportDst := layers.NewTCPPortEndpoint(20001)
	net, err := gopacket.FlowFromEndpoints(netSrc, netDst)
	assert.NoError(t, err)
	transport, err := gopacket.FlowFromEndpoints(transportSrc, transportDst)
	assert.NoError(t, err)

	ts := f.New(net, transport)
	s, ok := ts.(*stream)
	assert.True(t, ok)

	reassembly := tcpassembly.Reassembly{
		Bytes: []byte{0, 0, 0, 0, 0, 0, 0, 0, 0},
		Seen:  time.Now(),
	}

	s.Reassembled([]tcpassembly.Reassembly{reassembly})

	select {
	case <-msgChan:
		assert.FailNow(t, "error")
	default:
	}

	assert.Equal(t, reassembly.Bytes, s.buf)
}

func TestReassembled2Packet(t *testing.T) {
	msgChan := make(chan *Record, 100)
	f := newStreamFactory(msgChan, testDecodeFunc)

	netSrc := layers.NewIPEndpoint(net.ParseIP("127.0.0.1"))
	netDst := layers.NewIPEndpoint(net.ParseIP("10.2.2.2"))
	transportSrc := layers.NewTCPPortEndpoint(50123)
	transportDst := layers.NewTCPPortEndpoint(20001)
	net, err := gopacket.FlowFromEndpoints(netSrc, netDst)
	assert.NoError(t, err)
	transport, err := gopacket.FlowFromEndpoints(transportSrc, transportDst)
	assert.NoError(t, err)

	ts := f.New(net, transport)
	s, ok := ts.(*stream)
	assert.True(t, ok)

	reassembly0 := tcpassembly.Reassembly{
		Bytes: []byte{1, 2, 3},
		Seen:  time.Now(),
	}
	reassembly1 := tcpassembly.Reassembly{
		Bytes: []byte{4, 5, 6, 7, 8, 9, 10, 11},
		Seen:  time.Now(),
	}

	s.Reassembled([]tcpassembly.Reassembly{reassembly0, reassembly1})

	select {
	case r := <-msgChan:
		assert.Equal(t, RecordType(RecordTypeTCP), r.Type)
		assert.Equal(t, net, r.Net)
		assert.Equal(t, transport, r.Transport)
		assert.Equal(t, []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}, r.Buffer)
		assert.Equal(t, 1, len(r.Bodies))
		str, ok := r.Bodies[0].(string)
		assert.True(t, ok)
		assert.Equal(t, []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}, []byte(str))
		assert.True(t, r.Seen.UnixNano() > reassembly1.Seen.UnixNano())
	default:
		assert.FailNow(t, "error")
	}

	assert.Equal(t, []byte{11}, s.buf)
}
