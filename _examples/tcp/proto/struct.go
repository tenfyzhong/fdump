package proto

// Header proto header
type Header struct {
	Len       uint32 // header len plus body len
	Command   uint32
	Maybe0    uint64
	Maybe1    uint32
	Result    uint32
	IsRequest byte
}

type Command1Req struct {
	Name string
}
type Command1Rsp struct {
	Error error
	Greet string
}

type OpType int

const (
	OpTypeAdd OpType = iota
	OpTypeMinus
	OpTypeMul
	OpTypeDiv
	OpTypeMod
)

type Command2Req struct {
	NumLeft  int
	NumRight int
	OpType   OpType
}
type Command2Rsp struct {
	Error  error
	Result int
}
