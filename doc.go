/*
Package fdump is a framework to create an application to capture network packet
and decode the packet. It use a tui to show the packets.

Here's the way to create your packet capture application:
1. make a function to decode the binary packet and return an Record.
2. make a function to show the brief of a record.
3. make a function to show the detail message of a record.
4. (optional)Make some hook to modify the replay worker.

Example:

  func decode(net, transport gopacket.Flow, buf []byte) (bodies []interface{}, n int, err error) {
  	if len(buf) < 4 {
  		err = fdump.ErrPkgNoEnough
  		return
  	}
  	pkgLen := binary.BigEndian.Uint32(buf)
  	if uint32(len(buf)) < pkgLen {
  		err = fdump.ErrPkgNoEnough
  		return
  	}
  	str := string(buf[4:pkgLen])
  	bodies = append(bodies, str)
  	n = pkgLen
  	return
  }

  func brief(record *fdump.Record) []string {
  	if record == nil || len(record.Bodies) == 0 {
  		return nil
  	}
  	str, ok := record.Bodies[0].(string)
  	if !ok {
  		return nil
  	}
  	return []string{str[:10]}
  }

  func detail(record *fdump.Record) string {
  	str, ok := record.Bodies[0].(string)
  	if !ok {
  		return ""
  	}
  	return str
  }

  func postSend(conn net.Conn, record *fdump.Record) error {
  	lenBuf := make([]byte, 4)
  	lenLen := 0
  	for lenLen < 4 {
  		err := conn.SetReadDeadline(time.Now().Add(1*time.Second))
  		if err != nil {
  			return err
  		}
  		n, err := conn.Read(headBuf[lenLen:])
  		if err != nil {
  			return err
  		}
  		lenLen += n
  	}

  	bodyLen := binary.BigEndian.Uint32(lenBuf) - 4
  	body := make([]byte, bodyLen)
  	curLen := 0
  	for curLen < int(bodyLen) {
  		err := conn.SetReadDeadline(time.Now().Add(t*time.Second))
  		if err != nil {
  			return err
  		}
  		n, err := conn.Read(body[curlen:])
  		if err != nil {
  			return err
  		}
  		curlen += n
  	}
  	return nil
  }

  func main() {
  	logging.SetLevel(logging.INFO, "")
  	fdump.Init()
  	replayHook := &fdump.ReplayHook{
  		PostSend: postSend,
  	}
  	sidebarAttributes := []*fdump.SidebarColumnAttribute{&fdump.SidebarColumnAttribute{
  			Title: "Head10",
  			MaxWidth: 10,
  		},
  	}

  	a := fdump.NewApp(decode, brief, detail, replayHook, sidebarAttributes)
  	a.Run()
  }

If you want to add your owner command flag, please use fdump.AppFlagSet.

The framework use github.com/op/go-logging to write log. You can get the some log
: `logging.MustGetLogger(fdump.LoggerName)`.

Modify the logger level: `logging.SetLevel(logging.INFO, "")`
*/
package fdump
