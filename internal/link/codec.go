package link

import (
	"fmt"
	"strings"

	"github.com/hujm2023/go-sms-protocol/codec"

	"github.com/morya/spgw/spgwpb"
)

// MyCodec 负责在 spgwpb.Msg 和底层协议字节流之间转换。
// Link 持有一个 MyCodec 实例完成所有编解码，不再直接碰 protocol.PDU / cmpp 等协议细节。
type MyCodec interface {
	Proto() string

	// Decode 从连接读取一帧并转换为 Msg。
	//   msg - 业务消息，可能为 nil（如纯 ActiveTestResp）
	//   ack - 需立即回写的响应（如 ActiveTestReq -> ActiveTestAck），可能为 nil
	//   err - codec.ErrPacketNotComplete 表示数据不完整，需等待更多数据
	Decode(r codec.ConnReader) (msg, ack *spgwpb.Msg, err error)

	Encode(msg *spgwpb.Msg) ([]byte, error)

	// BuildBindReq 构造绑定请求 Msg，含协议特定的鉴权计算。
	BuildBindReq(spid, pswd string, seqID uint32) *spgwpb.Msg
}

// NewMyCodec 根据协议标识创建对应的编解码器。
// proto 大小写不敏感，兼容 "cmpp30" / "CMPP30" 等写法。
func NewMyCodec(proto string) (MyCodec, error) {
	switch strings.ToLower(proto) {
	case "cmpp30":
		return NewCodecCMPP30(), nil
	case "cmpp20":
		return NewCodecCMPP20(), nil
	default:
		return nil, fmt.Errorf("unsupported protocol: %s", proto)
	}
}
