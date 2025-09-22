package respond

// Message 通用响应结构
// @Description 统一的 API 响应格式
type Message struct {
	Code           int         `json:"code" example:"0" description:"响应代码，0表示成功"`
	Message        string      `json:"message" example:"success" description:"响应消息"`
	ProcessingTime int64       `json:"processingTime" example:"123" description:"处理时间（毫秒）"`
	Data           interface{} `json:"data" description:"响应数据"`
}

// Response 通用响应结构（用于 Swagger 文档）
// @Description 统一的 API 响应格式
type Response struct {
	Code           int         `json:"code" example:"0" description:"响应代码，0表示成功"`
	Message        string      `json:"message" example:"success" description:"响应消息"`
	ProcessingTime int64       `json:"processingTime" example:"123" description:"处理时间（毫秒）"`
	Data           interface{} `json:"data" description:"响应数据"`
}

// AuthError 认证错误
type AuthError struct {
	message string
}

// NewAuthError 创建认证错误
func NewAuthError(message string) *AuthError {
	return &AuthError{message: message}
}

// Error 实现 error 接口
func (e *AuthError) Error() string {
	return e.message
}

func RespSuccess(data interface{}, time int64) Message {
	return Message{
		Code:           HttpsCodeSuccess,
		Message:        RespMessageSuccess,
		ProcessingTime: time,
		Data:           data,
	}
}

func RespErr(err error, time int64, code int) Message {
	if code == 0 {
		code = HttpsCodeError
	}
	return Message{
		Code:           code,
		Message:        err.Error(),
		ProcessingTime: time,
		Data:           nil,
	}
}
