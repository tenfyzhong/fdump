package main

import (
	"encoding/json"
	"fmt"
	"net"

	"github.com/google/gopacket"
	logging "github.com/op/go-logging"
	"github.com/tenfyzhong/fdump"
	"github.com/tenfyzhong/fdump/_examples/udp/proto"
)

var log = logging.MustGetLogger(fdump.LoggerName)

func main() {
	fmt.Println("vim-go")
	logging.SetLevel(logging.DEBUG, "")

	fdump.Init()
	replayHook := &fdump.ReplayHook{
		PostSend: postSend,
	}

	briefAttributes := []*fdump.BriefColumnAttribute{
		&fdump.BriefColumnAttribute{
			Title:    "Src",
			MaxWidth: 21,
		},
		&fdump.BriefColumnAttribute{
			Title:    "Dst",
			MaxWidth: 21,
		},
		&fdump.BriefColumnAttribute{
			Title:    "Who",
			MaxWidth: 10,
		},
		&fdump.BriefColumnAttribute{
			Title:    "N",
			MaxWidth: 3,
		},
	}

	a := fdump.NewApp(decode, brief, detail, replayHook, briefAttributes)
	a.Run()
}

func decode(net, transport gopacket.Flow, buf []byte) (bodies []interface{}, n int, err error) {
	obj := &proto.Proto{}
	e := json.Unmarshal(buf, obj)
	if err != nil {
		err = e
		return
	}

	bodies = append(bodies, obj)

	n = len(buf)
	return
}

func brief(record *fdump.Record) []string {
	if record == nil || len(record.Bodies) == 0 {
		return nil
	}

	obj, ok := record.Bodies[0].(*proto.Proto)
	if !ok {
		return nil
	}

	result := make([]string, 4)
	src := fmt.Sprintf("%s:%s", getIP(record.Net.Src()), record.Transport.Src().String())
	result[0] = src
	dst := fmt.Sprintf("%s:%s", getIP(record.Net.Dst()), record.Transport.Dst().String())
	result[1] = dst
	result[2] = obj.Whoami
	result[3] = fmt.Sprintf("%d", obj.N)

	return result
}

func getIP(endpoint gopacket.Endpoint) string {
	str := endpoint.String()
	if str == "::1" {
		return "127.0.0.1"
	}
	return str
}

func detail(record *fdump.Record) string {
	if record == nil || len(record.Bodies) == 0 {
		log.Debugf("record is nil or bodies is empty, record: %v", record)
		return ""
	}

	obj, ok := record.Bodies[0].(*proto.Proto)
	if !ok {
		log.Debugf("bodies[0] is not a Proto")
		return ""
	}

	buf, err := json.MarshalIndent(obj, "", "  ")
	if err != nil {
		log.Debugf("marshal failed")
		return ""
	}

	return string(buf)
}

func postSend(conn net.Conn, record *fdump.Record) error {
	recvBuf := make([]byte, 65535)
	conn.Read(recvBuf)
	return nil
}
