package consts

/*
错误码	错误描述	解决方法
0	成功	无
1	失败	联系客服处理
2	参数校验失败，参数不能为空	查看接口参数，必填参数不为空
3	余额不足	联系客服充值
4	用户不存在	检查用户id是否正确，联系客服确认接口参数
5	未开通HTTP协议接口权限	API接口未开通，联系客服处理
6	sign校验失败	参照接口文档，检查加密的字符串与加密的算法是否正确，MD5用32位，值必须要小写进行加密
7	IP校验失败	检查提交的IP地址是否与开户信息一致，联系客服确认绑定IP
8	time时间格式错误	参照接口文档，检查时间字段格式是否正确，yyyyMMddHHmmss格式，如：20110115105822
9	扩展号格式错误	参照接口文档，检查扩展号格式与长度是否正确，扩展号最长6位，纯数字
10	发送内容包含敏感词[**]	修改短信内容，删除敏感词后提交
14	ts校验有误，要求5分钟内有效	参照接口文档，检查时间戳格式是否正确，是否精确到毫秒，检查服务器时间是否准确
15	号码个数超过限制	参照接口文档中一次批量最大提交号码个数
16	超出可定时发送时间	定时发送任务限制15天内
17	超出可发送时间范围	不在发送时间范围内，禁止发送
*/

const (
	ErrRestCodeSuccess       = 0
	ErrRestCodeFailed        = 1  // 联系客服处理
	ErrRestCodeErrParam      = 2  // 参数校验失败，参数不能为空
	ErrRestCodeErrBalance    = 3  // 余额不足
	ErrRestCodeErrUser       = 4  // 用户不存在
	ErrRestCodeErrPermission = 5  // 未开通HTTP协议接口权限
	ErrRestCodeErrSign       = 6  // sign校验失败
	ErrRestCodeErrIP         = 7  // IP校验失败
	ErrRestCodeErrTime       = 8  // time时间格式错误
	ErrRestCodeErrExt        = 9  // 扩展号格式错误
	ErrRestCodeErrContent    = 10 // 发送内容包含敏感词[**]
	ErrRestCodeErrTS         = 14 // ts校验有误，要求5分钟内有效
	ErrRestCodeErrNum        = 15 // 号码个数超过限制
	ErrRestCodeErrTimeRange  = 17 // 超出可发送时间范围
)
