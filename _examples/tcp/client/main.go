package main

import (
	"flag"
	"log"
	"net"

	"github.com/tenfyzhong/fdump/_examples/tcp/proto"
)

var (
	addr = flag.String("addr", "127.0.0.1:10001", "server addr")
)

func main() {
	conn, err := net.Dial("tcp", *addr)
	if err != nil {
		panic(err)
	}
	err = command1(conn, "tenfy")
	if err != nil {
		panic(err)
	}
	err = command2(conn, 1, 2, proto.OpTypeAdd)
	if err != nil {
		panic(err)
	}
	err = command2(conn, 3, 2, proto.OpTypeMinus)
	err = command2(conn, 3, 2, proto.OpTypeMul)
	if err != nil {
		panic(err)
	}
	err = command2(conn, 4, 2, proto.OpTypeDiv)
	if err != nil {
		panic(err)
	}
	err = command2(conn, 5, 2, proto.OpTypeMod)
	if err != nil {
		panic(err)
	}
	err = command1Error(conn)
	if err != nil {
		panic(err)
	}
}

func command1(conn net.Conn, name string) error {
	header := &proto.Header{
		Command:   1,
		IsRequest: byte(1),
	}
	req := &proto.Command1Req{
		Name: name,
	}
	log.Printf("command1 header: %+v, req: %+v", header, req)

	err := proto.Write(conn, header, req)
	if err != nil {
		return err
	}

	header, body, err := proto.Read(conn)
	if err != nil {
		return err
	}

	rsp := &proto.Command1Rsp{}
	err = proto.DecodeBody(body, rsp)
	if err != nil {
		return err
	}
	log.Printf("command1 header: %+v, rsp: %+v", header, rsp)

	return nil
}

func command2(conn net.Conn, numLeft, numRight int, opType proto.OpType) error {
	header := &proto.Header{
		Command:   2,
		IsRequest: byte(1),
	}
	req := &proto.Command2Req{
		NumLeft:  numLeft,
		NumRight: numRight,
		OpType:   opType,
	}
	log.Printf("command2 header: %+v, req: %+v", header, req)

	err := proto.Write(conn, header, req)
	if err != nil {
		return err
	}

	header, body, err := proto.Read(conn)
	if err != nil {
		return err
	}

	rsp := &proto.Command2Rsp{}
	err = proto.DecodeBody(body, rsp)
	if err != nil {
		return err
	}
	log.Printf("command2 header: %+v, rsp: %+v", header, rsp)
	return nil
}

func command1Error(conn net.Conn) error {
	header := &proto.Header{
		Command:   3,
		IsRequest: byte(1),
	}
	req := &proto.Command1Req{
		Name: "tenfy",
	}
	log.Printf("command1 header: %+v, req: %+v", header, req)

	err := proto.Write(conn, header, req)
	if err != nil {
		return err
	}

	header, body, err := proto.Read(conn)
	if err != nil {
		return err
	}

	rsp := &proto.Command1Rsp{}
	err = proto.DecodeBody(body, rsp)
	if err != nil {
		return err
	}
	log.Printf("command1 header: %+v, rsp: %+v", header, rsp)

	return nil
}
