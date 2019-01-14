package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net"
	"time"

	"github.com/tenfyzhong/fdump/_examples/udp/proto"
)

var (
	addr = flag.String("addr", ":10000", "listen addr")
)

func main() {
	flag.Parse()
	fmt.Println("listen addr, ", *addr)

	udpAddr, err := net.ResolveUDPAddr("udp", *addr)
	if err != nil {
		log.Fatalln("Error: ", err)
	}

	// Build listining connections
	conn, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		log.Fatalln("Error: ", err)
	}
	defer conn.Close()

	recvBuff := make([]byte, 65535)
	for {
		n, remoteAddr, err := conn.ReadFromUDP(recvBuff)
		if err != nil {
			log.Println("read err: ", err)
			return
		}

		b := make([]byte, n)
		copy(b, recvBuff)
		go worker(conn, remoteAddr, b)
	}
}

func worker(conn *net.UDPConn, addr *net.UDPAddr, buff []byte) error {
	obj := &proto.Proto{}

	err := json.Unmarshal(buff, obj)
	if err != nil {
		return err
	}
	obj.Now = time.Now().UnixNano()

	sendBuf, err := json.Marshal(obj)
	if err != nil {
		return err
	}

	_, err = conn.WriteToUDP(sendBuf, addr)
	if err != nil {
		return err
	}
	return nil
}
