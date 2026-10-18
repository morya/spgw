package seq

import (
	"strings"
	"testing"
)

func dumpBinary(msgID uint32) string {
	var s strings.Builder
	for i := 31; i >= 0; i-- {
		if msgID&(1<<uint(i)) != 0 {
			s.WriteString("1")
		} else {
			s.WriteString("0")
		}
	}
	return s.String()
}

func TestNextMsgId(t *testing.T) {
	seq = 65530
	for range 100 {
		v := NextSeqId()

		vv := uint32(uint16(v))
		t.Logf("msgID 16bit: %s, 32bit: %s", dumpBinary(vv), dumpBinary(v))
	}
}
