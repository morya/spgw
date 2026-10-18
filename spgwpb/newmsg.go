package spgwpb

func NewMsg(cmdID CmdID, seq uint32) *Msg {
	return &Msg{
		Head: &Head{
			CommandId: cmdID,
			SeqId:     seq,
		},
	}
}
