package consts

const (
	ErrCmppCodeSuccess        = 0
	ErrCmppCodeMsgStruct      = 1   // 消息结构错，检查 CMPP_LOGIN_REQ包是否正确
	ErrCmppCodeInvalidSrc     = 2   // 非法源地址，确认账号绑定IP是否正确
	ErrCmppCodeAuthErr        = 3   // 认证错误，核对网关账号和密码是否有误
	ErrCmppCodeVersionErr     = 4   // CMPP版本错误，与技术人员确认CMPP对应版本
	ErrCmppCodeMaxConnErr     = 5   // 超过网关最大连接数，通常是由于客户端发起过多连接造成，一般情况下，可以断开客户端所有连接，等待30秒后重试
	ErrCmppCodeBusinessErr    = 7   // 业务代码错误，联系网关技术人员处理
	ErrCmppCodeMaxFlowErr     = 8   // 超过网关最大流速，降低发送速度，联系网关技术人员处理
	ErrCmppCodeSrcIdErr       = 10  // Src_Id错误，网关下行接入号错误，联系网关技术人员处理
	ErrCmppCodeSpIpErr        = 57  // SP登陆IP错误，绑定的IP地址错误，联系网关技术人员处理
	ErrCmppCodeGatewayErr     = 169 // 网关错误，联系网关技术人员处理
	ErrCmppCodeGatewayConnErr = 255 // 网关连接错误，联系网关技术人员处理

	// deliverAck result 最大值 255
	// 参考 type ErrCode uint32
)
