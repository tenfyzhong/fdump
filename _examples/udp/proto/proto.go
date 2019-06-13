package proto

type Proto struct {
	Whoami string `json:"whoami"`
	Now    int64  `json:"now"`
	N      int    `json:"n"`
}
