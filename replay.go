package fdump

import (
	"net"
)

// PreReplayHook Will call before replay packet
type PreReplayHook func(conn net.Conn, records []*Record) error

// PreSendHook will call before send packet
type PreSendHook func(conn net.Conn, record *Record) error

// PostSendHook will call after send packet. You should implement it and
// receive the response packet if you capture the replay response packet.
// Otherwise it will close the `conn` before receive the response packet.
type PostSendHook func(conn net.Conn, record *Record) error

// PostReplayHook will call after replay
type PostReplayHook func(conn net.Conn) error

// ReplayHook will use in replay action.
type ReplayHook struct {
	PreReplay  PreReplayHook
	PreSend    PreSendHook
	PostSend   PostSendHook
	PostReplay PostReplayHook
}
