package consts

const (
	BusCodeUserStartCode = (iota+1)*1000 + 1
	BusCodeSettingStartCode
	BusCodeUserGroupStartCode
	BusCodeStorageStartCode
	BusCodeCloudTokenStartCode
	BusCodeFileStartCode
	BusCodeTaskStateStartCode
	BusCodeStorageAdvanceStartCode
	BusCodeDavStartCode
	BusCodeAutoIngestStartCode
	BusCodeLoginLogStartCode
	BusCodeMediaStartCode

	BusCodeMiddlewareAuth = 99100 + 1

	// 二维码登录状态码（供前端轮询判断使用）
	BusCodeQrcodeExpired = 40001 // 二维码已过期
	BusCodeQrcodeCancel  = 40002 // 用户取消登录
	BusCodeQrcodeWaiting = 40003 // 等待用户扫码确认
)
