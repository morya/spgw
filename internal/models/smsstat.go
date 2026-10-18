package models

// ReqListIStat 北向账号统计列表查询参数
type ReqListIStat struct {
	BasePaging

	IAccountID uint32 `json:"iaccountId" form:"iaccountId"` // SP 接入账号 ID（可选过滤）
	NodeID     uint32 `json:"nodeId" form:"nodeId"`         // 节点 ID（可选过滤）
}

// ReqListOStat 南向账号统计列表查询参数
type ReqListOStat struct {
	BasePaging

	OAccountID uint32 `json:"oaccountId" form:"oaccountId"` // 下游通道账号 ID（可选过滤）
	NodeID     uint32 `json:"nodeId" form:"nodeId"`         // 节点 ID（可选过滤）
}
