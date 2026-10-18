package consts

type NodeType int

const (
	NodeTypeUnknown NodeType = iota
	NodeTypeIAgent
	NodeTypeOAgent
	NodeTypeServer
	NodeTypeApi
)
