package seq

import (
	"sync/atomic"
)

var seq uint32

func NextSeqId() uint32 {
	atomic.AddUint32(&seq, 1)
	return seq
}
