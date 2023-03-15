package mqtt

const (
	// TopicBase RPC主题基路径
	TopicBase = "v1/devices/rdo/"
	// TopicTelemetry 主动上报主题
	TopicTelemetry = TopicBase + "telemetry"

	// TopicRpc RPC主题基路径
	TopicRpc = TopicBase + "rpc/"
	// TopicRpcRequest RPC请求主题
	TopicRpcRequest = TopicRpc + "request/+"
	// TopicRpcResponse RPC处理结果回复主题
	TopicRpcResponse = TopicRpc + "response/"

	// ResponseRpcFmt RPC调用返回消息的模板
	ResponseRpcFmt = "{\"success\":%s,\"message\":\"%s\",\"data\":%s}"

	ResultTrue  = "true"
	ResultFalse = "false"
)

const (
	QoS0 = iota
	QoS1
	QoS2
)
