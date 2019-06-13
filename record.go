package fdump

import (
	"time"

	"github.com/google/gopacket"
)

// RecordType type of the record
type RecordType int

//
const (
	RecordTypeTCP = iota
	RecordTypeUDP
)

// Record decoded object
type Record struct {
	Type      RecordType
	Net       gopacket.Flow
	Transport gopacket.Flow
	Seen      time.Time
	Bodies    []interface{}
	Buffer    []byte
}
