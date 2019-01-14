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
	dial  = flag.String("dial", "localhost:10000", "dial addr")
	times = flag.Int("t", 1, "send times")
)

func main() {
	flag.Parse()
	if *times < 0 {
		*times = 1
	}

	fmt.Println("dial ", *dial)
	conn, err := net.DialTimeout("udp", *dial, 1*time.Second)
	if err != nil {
		log.Fatalln(err)
	}
	defer conn.Close()

	for i := 0; i < *times; i++ {
		worker(conn, i+1)
	}

}

func worker(conn net.Conn, n int) {
	obj := &proto.Proto{
		Whoami: "tenfy",
		Now:    time.Now().UnixNano(),
		N:      n,
	}

	sendBuf, err := json.Marshal(obj)
	if err != nil {
		log.Fatalln(err)
	}

	_, err = conn.Write(sendBuf)
	if err != nil {
		log.Fatalln(err)
	}

	recvBuf := make([]byte, 65535)
	readLen, err := conn.Read(recvBuf)
	if err != nil {
		log.Fatalln(err)
	}

	obj.Whoami = ""
	obj.Now = 0
	obj.N = 0
	err = json.Unmarshal(recvBuf[:readLen], &obj)
	if err != nil {
		log.Fatalln(err)
	}

	fmt.Printf("%+v\n", obj)
}
