package link

import "fmt"

type Mode int

const (
	ModeClient Mode = iota
	ModeServer
)

var modeDesc = map[Mode]string{
	ModeClient: "client",
	ModeServer: "server",
}

func (m Mode) String() string {
	s, ok := modeDesc[m]
	if !ok {
		return fmt.Sprintf("Bad Mode(%d)", m)
	}
	return s
}
