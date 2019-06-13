package fdump

import (
	"time"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcap"
	"github.com/google/gopacket/tcpassembly"
)

type updateFunc func(record *Record)

type controller struct {
	iface       string
	fname       string
	snaplen     int
	filter      string
	handle      *pcap.Handle
	factory     *streamFactory
	msgChan     chan *Record
	updateFuncs []updateFunc
}

func newController(iface string, fname string, snaplen int, filter string, decodeFunc DecodeFunc) *controller {
	log.Infof("iface: %s, snaplen: %d, filter: %s\n", iface, snaplen, filter)
	msgChan := make(chan *Record, 1000)
	c := &controller{
		iface:       iface,
		fname:       fname,
		snaplen:     snaplen,
		filter:      filter,
		msgChan:     msgChan,
		factory:     newStreamFactory(msgChan, decodeFunc),
		updateFuncs: make([]updateFunc, 0, 1),
	}
	return c
}

func (c *controller) Init() error {
	var handle *pcap.Handle
	var err error

	if fname != "" {
		handle, err = pcap.OpenOffline(c.fname)
	} else {
		handle, err = pcap.OpenLive(
			c.iface,
			int32(c.snaplen),
			true,
			pcap.BlockForever)
	}

	if err != nil {
		log.Errorf("OpenLive failed, err: %+v", err)
		return err
	}

	if err := handle.SetBPFFilter(c.filter); err != nil {
		log.Errorf("set bpf filter failed, err: %+v", err)
		return err
	}

	c.handle = handle
	go c.consumeMsg()

	return nil
}

func (c *controller) AddUpdateFunc(f updateFunc) {
	c.updateFuncs = append(c.updateFuncs, f)
}

func (c *controller) consumeMsg() {
	for msg := range c.msgChan {
		for _, f := range c.updateFuncs {
			f(msg)
		}
	}
}

func (c *controller) Run() {
	streamPool := tcpassembly.NewStreamPool(c.factory)
	assembler := tcpassembly.NewAssembler(streamPool)
	log.Infof("reading in packets")

	packetSource := gopacket.NewPacketSource(c.handle, c.handle.LinkType())
	packets := packetSource.Packets()
	ticker := time.Tick(60 * time.Second)
	for {
		select {
		case packet := <-packets:
			if packet == nil {
				log.Errorf("get a nil packet")
				return
			}

			if packet.NetworkLayer() == nil ||
				packet.TransportLayer() == nil {
				continue
			}

			switch packet.TransportLayer().LayerType() {
			case layers.LayerTypeTCP:
				c.assembleTCP(assembler, packet)
			case layers.LayerTypeUDP:
				c.assembleUDP(packet)
			}
		case <-ticker:
			assembler.FlushWithOptions(tcpassembly.FlushOptions{
				T:        time.Now().Add(time.Minute * -2),
				CloseAll: false,
			})
		}
	}
}

func (c *controller) assembleTCP(assembler *tcpassembly.Assembler, packet gopacket.Packet) {
	tcp := packet.TransportLayer().(*layers.TCP)
	assembler.AssembleWithTimestamp(
		packet.NetworkLayer().NetworkFlow(),
		tcp,
		packet.Metadata().Timestamp)
	assembler.FlushWithOptions(tcpassembly.FlushOptions{
		T:        time.Now(),
		CloseAll: false,
	})
}

func (c *controller) assembleUDP(packet gopacket.Packet) {
	udp := packet.TransportLayer().(*layers.UDP)
	netFlow := packet.NetworkLayer().NetworkFlow()
	transportFlow := udp.TransportFlow()
	payload := udp.Payload

	bodies, _, err := c.factory.decodeFunc(netFlow, transportFlow, payload)
	if err != nil {
		log.Debugf("unpack err: %+v", err)
		return
	}

	if len(bodies) == 0 {
		return
	}

	r := &Record{
		Type:      RecordTypeUDP,
		Bodies:    bodies,
		Net:       netFlow,
		Transport: transportFlow,
		Seen:      time.Now(),
		Buffer:    payload,
	}

	c.msgChan <- r
}
