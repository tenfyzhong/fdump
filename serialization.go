package fdump

import (
	"time"

	"github.com/google/gopacket"
)

type serialization struct {
	NetSrcRaw, NetDstRaw               []byte
	NetSrcType, NetDstType             gopacket.EndpointType
	TransportSrcRaw, TransportDstRaw   []byte
	TransportSrcType, TransportDstType gopacket.EndpointType
	Seen                               time.Time
	Buffer                             []byte
}

func (s serialization) Net() (gopacket.Flow, error) {
	netSrc := gopacket.NewEndpoint(s.NetSrcType, s.NetSrcRaw)
	netDst := gopacket.NewEndpoint(s.NetDstType, s.NetDstRaw)
	net, err := gopacket.FlowFromEndpoints(netSrc, netDst)
	return net, err
}

func (s serialization) Transport() (gopacket.Flow, error) {
	transportSrc := gopacket.NewEndpoint(s.TransportSrcType, s.TransportSrcRaw)
	transportDst := gopacket.NewEndpoint(s.TransportDstType, s.TransportDstRaw)
	transport, err := gopacket.FlowFromEndpoints(transportSrc, transportDst)
	return transport, err
}

func message2Serialization(m *Record) *serialization {
	netSrc := m.Net.Src()
	netDst := m.Net.Dst()
	transportSrc := m.Transport.Src()
	transportDst := m.Transport.Dst()
	return &serialization{
		NetSrcRaw:        netSrc.Raw(),
		NetDstRaw:        netDst.Raw(),
		NetSrcType:       netSrc.EndpointType(),
		NetDstType:       netDst.EndpointType(),
		TransportSrcRaw:  transportSrc.Raw(),
		TransportDstRaw:  transportDst.Raw(),
		TransportSrcType: transportSrc.EndpointType(),
		TransportDstType: transportDst.EndpointType(),
		Seen:             m.Seen,
		Buffer:           m.Buffer,
	}
}
