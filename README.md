# fdump
A framework to dump and decode network packet. 

# Feature
- [x] tui to show records.
- [x] replay records.
- [x] save records to file.
- [x] load records from file.
- [x] support tcp.
- [x] support udp.

# Screenshots

# How to
Here's the way to create your packet capture application:  
1. Make a function to decode the binary packet and return an Record.  
2. Make a function to show a brief message of the record.  
3. Make a function to show a detail message of the record.  
4. (optional)Make some hook to modify the replay worker.  

example:  
```go
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
	briefAttributes := []*fdump.BriefColumnAttribute{&fdump.BriefColumnAttribute{
			Title: "Head10",
			MaxWidth: 10,
		},
	}

	a := fdump.NewApp(decode, brief, detail, replayHook, briefAttributes)
	a.Run()
}
```

# Key
| View   | Key             | Summary                               |
|:-------|:----------------|:--------------------------------------|
| all    | `f`             | toggle frozen scroll                  |
| all    | `s`             | toggle stop capture                   |
| all    | `l`/`Left`      | left                                  |
| all    | `r`/`Right`     | right                                 |
| all    | `j`/`Down`      | down                                  |
| all    | `k`/`Up`        | up                                    |
| all    | `g`/`Home`      | goto first line                       |
| all    | `G`/`End`       | goto last line                        |
| all    | `ctrl-f`/`PgDn` | page down                             |
| all    | `ctrl-b`/`PgUp` | page up                               |
| all    | `ctrl-c`        | exit                                  |
| all    | `?`             | help                                  |
| brief  | `enter`         | enter detail                          |
| brief  | `Esc`           | clean prompt                          |
| brief  | `C`             | clear                                 |
| brief  | `S`             | save selected/all to file             |
| brief  | `L`             | load from file                        |
| brief  | `M`             | toggle multiple select mode           |
| brief  | `m`             | select/unselect row, select mode only |
| brief  | `r`             | revert selected, select mode only     |
| brief  | `a`             | select/unselect all, select mode only |
| brief  | `c`             | clear selected, select mode only      |
| brief  | `R`             | replay current select row             |
| detail | `q`/`Esc`       | exit detail                           |
| help   | `q`/`Esc`       | exit help                             |

# Warnning
Please set `LC_CTYPE` to `en_US.UTF-8`:
```sh
export LC_CTYPE=en_US.UTF-8
```
