package fdump

import (
	"net"
	"testing"
	"time"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/stretchr/testify/assert"
)

func TestMessage2Sericalization(t *testing.T) {
	netSrc := layers.NewIPEndpoint(net.ParseIP("127.0.0.1"))
	netDst := layers.NewIPEndpoint(net.ParseIP("10.2.2.2"))
	transportSrc := layers.NewTCPPortEndpoint(50123)
	transportDst := layers.NewTCPPortEndpoint(20001)
	net, err := gopacket.FlowFromEndpoints(netSrc, netDst)
	assert.NoError(t, err)
	transport, err := gopacket.FlowFromEndpoints(transportSrc, transportDst)
	assert.NoError(t, err)
	seen := time.Now()
	record := &Record{
		Type:      RecordTypeTCP,
		Net:       net,
		Transport: transport,
		Seen:      seen,
		Buffer:    []byte{1, 2, 3},
	}

	s := message2Serialization(record)
	assert.NotNil(t, s)
	assert.Equal(t, RecordType(RecordTypeTCP), s.Type)
	assert.Equal(t, netSrc.Raw(), s.NetSrcRaw)
	assert.Equal(t, netSrc.EndpointType(), s.NetSrcType)
	assert.Equal(t, netDst.Raw(), s.NetDstRaw)
	assert.Equal(t, netDst.EndpointType(), s.NetDstType)
	assert.Equal(t, transportSrc.Raw(), s.TransportSrcRaw)
	assert.Equal(t, transportSrc.EndpointType(), s.TransportSrcType)
	assert.Equal(t, transportDst.Raw(), s.TransportDstRaw)
	assert.Equal(t, transportDst.EndpointType(), s.TransportDstType)
	assert.Equal(t, seen, s.Seen)
	assert.Equal(t, []byte{1, 2, 3}, s.Buffer)

	actualNet, err := s.Net()
	assert.NoError(t, err)
	assert.Equal(t, net, actualNet)

	actualTransport, err := s.Transport()
	assert.NoError(t, err)
	assert.Equal(t, transport, actualTransport)
}
