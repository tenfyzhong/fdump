package fdump

import (
	"time"

	"github.com/google/gopacket"
)

// Record decoded object
type Record struct {
	Net       gopacket.Flow
	Transport gopacket.Flow
	Seen      time.Time
	Bodies    []interface{}
	Buffer    []byte
}
