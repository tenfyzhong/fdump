package main

import (
	"errors"
	"flag"
	"log"
	"net"

	"github.com/tenfyzhong/fdump/_examples/tcp/proto"
)

var (
	addr = flag.String("addr", ":10001", "server addr")
)

func main() {
	l, err := net.Listen("tcp", *addr)
	if err != nil {
		panic(err)
	}

	for {
		c, err := l.Accept()
		if err != nil {
			panic(err)
		}
		// start a new goroutine to handle the connection.
		go handleConn(c)
	}
}

func handleConn(c net.Conn) {
	defer c.Close()
	for {
		// read
		header, body, err := proto.Read(c)
		if err != nil {
			log.Println("read", c, err)
			break
		}

		// process
		rspHeader, rsp := process(header, body)

		// write
		err = proto.Write(c, rspHeader, rsp)
		if err != nil {
			log.Println("write", c, err)
			break
		}
	}
}

func process(header *proto.Header, body []byte) (*proto.Header, interface{}) {
	header.IsRequest = byte(0)
	switch header.Command {
	case 1:
		return command1(header, body)
	case 2:
		return command2(header, body)
	}
	header.Result = 1
	return header, nil
}

func command1(header *proto.Header, body []byte) (*proto.Header, interface{}) {
	req := &proto.Command1Req{}
	rsp := &proto.Command1Rsp{}
	err := proto.DecodeBody(body, req)
	if err != nil {
		rsp.Error = err
		return header, rsp
	}

	rsp.Greet = "Hello, " + req.Name
	return header, rsp
}

func command2(header *proto.Header, body []byte) (*proto.Header, interface{}) {
	req := &proto.Command2Req{}
	rsp := &proto.Command2Rsp{}

	switch req.OpType {
	case proto.OpTypeAdd:
		rsp.Result = req.NumLeft + req.NumRight
	case proto.OpTypeMinus:
		rsp.Result = req.NumLeft - req.NumRight
	case proto.OpTypeMul:
		rsp.Result = req.NumLeft * req.NumRight
	case proto.OpTypeDiv:
		if req.NumRight == 0 {
			rsp.Error = errors.New("divisor is 0")
		} else {
			rsp.Result = req.NumLeft / req.NumRight
		}
	case proto.OpTypeMod:
		if req.NumRight == 0 {
			rsp.Error = errors.New("divisor is 0")
		} else {
			rsp.Result = req.NumLeft % req.NumRight
		}
	default:
		rsp.Error = errors.New("unsupport OpType")
	}
	return header, rsp
}
