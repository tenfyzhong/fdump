package fdump

import (
	"time"

	"github.com/google/gopacket"
)

type serialization struct {
	Type                               RecordType
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

func message2Serialization(record *Record) *serialization {
	netSrc := record.Net.Src()
	netDst := record.Net.Dst()
	transportSrc := record.Transport.Src()
	transportDst := record.Transport.Dst()
	return &serialization{
		Type:             record.Type,
		NetSrcRaw:        netSrc.Raw(),
		NetDstRaw:        netDst.Raw(),
		NetSrcType:       netSrc.EndpointType(),
		NetDstType:       netDst.EndpointType(),
		TransportSrcRaw:  transportSrc.Raw(),
		TransportDstRaw:  transportDst.Raw(),
		TransportSrcType: transportSrc.EndpointType(),
		TransportDstType: transportDst.EndpointType(),
		Seen:             record.Seen,
		Buffer:           record.Buffer,
	}
}
