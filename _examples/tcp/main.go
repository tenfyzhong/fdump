package main

import (
	"encoding/json"
	"fmt"
	"net"

	"github.com/google/gopacket"
	logging "github.com/op/go-logging"
	"github.com/tenfyzhong/fdump"
	"github.com/tenfyzhong/fdump/_examples/tcp/proto"
)

var log = logging.MustGetLogger(fdump.LoggerName)

func main() {
	logging.SetLevel(logging.DEBUG, "")
	fdump.Init()
	replayHook := &fdump.ReplayHook{
		PostSend: postSend,
	}

	briefAttributes := []*fdump.BriefColumnAttribute{
		&fdump.BriefColumnAttribute{
			Title:    "Cmd",
			MaxWidth: 4,
		},
		&fdump.BriefColumnAttribute{
			Title:    "Src",
			MaxWidth: 21,
		},
		&fdump.BriefColumnAttribute{
			Title:    "Dst",
			MaxWidth: 21,
		},
		&fdump.BriefColumnAttribute{
			Title:    "Time",
			MaxWidth: 12,
		},
	}
	a := fdump.NewApp(decode, brief, detail, replayHook, briefAttributes)
	a.Run()
}

func decode(net, transport gopacket.Flow, buf []byte) (bodies []interface{}, n int, err error) {
	if len(buf) < proto.HeaderLength {
		err = fdump.ErrPkgNoEnough
		return
	}

	header, e := proto.DecodeHeader(buf)
	if e != nil {
		err = e
		return
	}

	if uint32(len(buf)) < header.Len {
		err = fdump.ErrPkgNoEnough
		return
	}

	n = int(header.Len)
	bodies = append(bodies, header)

	var body interface{}
	switch header.Command {
	case 1, 3: // command 3 is an error request which use the type Command1Req
		if header.IsRequest == byte(1) {
			body = &proto.Command1Req{}
		} else {
			body = &proto.Command1Rsp{}
		}
	case 2:
		if header.IsRequest == byte(1) {
			body = &proto.Command2Req{}
		} else {
			body = &proto.Command2Rsp{}
		}
	}

	if body != nil {
		e = proto.DecodeBody(buf[proto.HeaderLength:], body)
		if e != nil {
			return
		}
		bodies = append(bodies, body)
	}

	return
}

func brief(record *fdump.Record) []string {
	if record == nil || len(record.Bodies) == 0 {
		return nil
	}

	header, ok := record.Bodies[0].(*proto.Header)
	if !ok {
		return nil
	}

	results := make([]string, 4)
	results[0] = fmt.Sprintf("%d", header.Command)
	results[1] = fmt.Sprintf("%s:%s", getIP(record.Net.Src()), record.Transport.Src().String())
	results[2] = fmt.Sprintf("%s:%s", getIP(record.Net.Dst()), record.Transport.Dst().String())
	results[3] = record.Seen.Format("15:04:05.000")
	return results
}

func detail(record *fdump.Record) string {
	if record == nil || len(record.Bodies) == 0 {
		return ""
	}

	result := ""
	result += fmt.Sprintf("Src: %s:%s\n", getIP(record.Net.Src()), record.Transport.Src().String())
	result += fmt.Sprintf("Dst: %s:%s\n", getIP(record.Net.Dst()), record.Transport.Dst().String())
	result += record.Seen.String() + "\n"
	result += "\n"

	for _, body := range record.Bodies {
		buf, err := json.MarshalIndent(body, "", "  ")
		if err != nil {
			continue
		}
		result += string(buf) + "\n\n"
	}

	return result
}

func postSend(conn net.Conn, record *fdump.Record) error {
	_, _, err := proto.Read(conn)
	return err
}

func getIP(endpoint gopacket.Endpoint) string {
	str := endpoint.String()
	if str == "::1" {
		return "127.0.0.1"
	}
	return str
}
