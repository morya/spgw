package consts

type State int

const (
	StateConnecting State = iota
	StateConnected
	StateAuthed
	StateClosed
)
