package ecode

// 业务错误码定义
const (
	// 成功
	Success = 0

	// 通用错误 (1000-1999)
	ServerError     = 1000 // 服务器内部错误
	InvalidParams   = 1001 // 参数错误
	Unauthorized    = 1002 // 未授权
	Forbidden       = 1003 // 禁止访问
	NotFound        = 1004 // 资源不存在
	RequestTimeout  = 1005 // 请求超时
	TooManyRequests = 1006 // 请求过于频繁

	// 租户相关错误 (2000-2999)
	TenantNotFound  = 2000 // 租户不存在
	TenantInactive  = 2001 // 租户已停用
	TenantSuspended = 2002 // 租户已冻结
	InvalidAPIKey   = 2003 // 无效的 API Key

	// 敏感词相关错误 (3000-3999)
	WordNotFound           = 3000 // 敏感词不存在
	WordAlreadyExists      = 3001 // 敏感词已存在
	WordCreateFailed       = 3002 // 创建敏感词失败
	WordUpdateFailed       = 3003 // 更新敏感词失败
	WordDeleteFailed       = 3004 // 删除敏感词失败
	EmbeddingExtractFailed = 3005 // 向量提取失败

	// 文本审查相关错误 (4000-4999)
	TextCheckFailed = 4000 // 文本审查失败
	TextTooLong     = 4001 // 文本过长

	// 检测历史相关错误 (5000-5999)
	HistoryNotFound     = 5000 // 检测历史不存在
	HistoryCreateFailed = 5001 // 创建检测历史失败
	HistoryDeleteFailed = 5002 // 删除检测历史失败
)

// Ecode 错误码接口
type Ecode interface {
	Code() int
	Message() string
	Error() string // 实现 error 接口
}

// ecode 错误码实现
type ecode struct {
	code    int
	message string
}

// New 创建新的错误码
func New(code int, message string) Ecode {
	return &ecode{code: code, message: message}
}

// Code 返回错误码
func (e *ecode) Code() int {
	return e.code
}

// Message 返回错误信息
func (e *ecode) Message() string {
	return e.message
}

// Error 返回错误信息（实现 error 接口）
func (e *ecode) Error() string {
	return e.message
}

// 预定义的错误码
var (
	ErrServer          = New(ServerError, "服务器内部错误")
	ErrInvalidParams   = New(InvalidParams, "参数错误")
	ErrUnauthorized    = New(Unauthorized, "未授权")
	ErrForbidden       = New(Forbidden, "禁止访问")
	ErrNotFound        = New(NotFound, "资源不存在")
	ErrRequestTimeout  = New(RequestTimeout, "请求超时")
	ErrTooManyRequests = New(TooManyRequests, "请求过于频繁")

	ErrTenantNotFound  = New(TenantNotFound, "租户不存在")
	ErrTenantInactive  = New(TenantInactive, "租户已停用")
	ErrTenantSuspended = New(TenantSuspended, "租户已冻结")
	ErrInvalidAPIKey   = New(InvalidAPIKey, "无效的 API Key")

	ErrWordNotFound           = New(WordNotFound, "敏感词不存在")
	ErrWordAlreadyExists      = New(WordAlreadyExists, "敏感词已存在")
	ErrWordCreateFailed       = New(WordCreateFailed, "创建敏感词失败")
	ErrWordUpdateFailed       = New(WordUpdateFailed, "更新敏感词失败")
	ErrWordDeleteFailed       = New(WordDeleteFailed, "删除敏感词失败")
	ErrEmbeddingExtractFailed = New(EmbeddingExtractFailed, "向量提取失败")

	ErrTextCheckFailed = New(TextCheckFailed, "文本审查失败")
	ErrTextTooLong     = New(TextTooLong, "文本过长")

	ErrHistoryNotFound     = New(HistoryNotFound, "检测历史不存在")
	ErrHistoryCreateFailed = New(HistoryCreateFailed, "创建检测历史失败")
	ErrHistoryDeleteFailed = New(HistoryDeleteFailed, "删除检测历史失败")
)
