package controller

import (
	"errors"
	"net/http"
	"push-base-service/controller/request"
	"push-base-service/controller/respond"
	"push-base-service/service/pebble_service"
	"push-base-service/tool"
	"strconv"

	"github.com/gin-gonic/gin"
)

// SetUserTokens godoc
// @Summary 设置用户推送令牌
// @Description 为指定用户在指定平台设置推送令牌，支持Token唯一性检查。Token本身就是设备的唯一标识，如果Token已被其他用户使用，会自动从原用户中移除该平台的令牌。
// @Tags Push API
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param request body request.SetUserTokensReq true "请求参数（metaId、platform、token）"
// @Success 200 {object} respond.Response "成功响应"
// @Failure 400 {object} respond.Response "参数错误"
// @Failure 401 {object} respond.Response "认证失败"
// @Failure 500 {object} respond.Response "服务器内部错误"
// @Router /v1/push/set_user_tokens [post]
func SetUserTokens(c *gin.Context) {
	var (
		t            int64 = tool.MakeTimestamp()
		requestModel *request.SetUserTokensReq
	)

	if c.ShouldBindJSON(&requestModel) == nil {
		// 调用 push_service 的方法（token作为设备ID）
		err := pebble_service.SetUserToken(requestModel.MetaID, requestModel.Platform, requestModel.Token)
		if err != nil {
			c.JSONP(http.StatusOK, respond.RespErr(err, tool.MakeTimestamp()-t, respond.HttpsCodeError))
			return
		}

		// 构造成功响应
		responseData := map[string]interface{}{
			"success": true,
			"message": "用户令牌设置成功",
		}

		c.JSONP(http.StatusOK, respond.RespSuccess(responseData, tool.MakeTimestamp()-t))
		return
	}

	c.JSONP(http.StatusInternalServerError, respond.RespErr(errors.New("参数错误"), tool.MakeTimestamp()-t, respond.HttpsCodeError))
}

// GetUserTokenByMetaID godoc
// @Summary 根据 metaId 获取用户推送令牌
// @Description 根据用户 metaId 获取该用户的所有推送令牌
// @Tags Push API
// @Produce json
// @Param metaId query string true "用户唯一标识"
// @Success 200 {object} respond.Response{data=models.UserPushTokens} "成功响应"
// @Failure 400 {object} respond.Response "参数错误"
// @Failure 401 {object} respond.Response "认证失败"
// @Failure 500 {object} respond.Response "服务器内部错误"
// @Router /v1/push/get_user_token [get]
func GetUserTokenByMetaID(c *gin.Context) {
	var t int64 = tool.MakeTimestamp()

	// 从 query 参数获取 metaId
	metaId := c.Query("metaId")
	if metaId == "" {
		c.JSONP(http.StatusOK, respond.RespErr(errors.New("metaId 参数不能为空"), tool.MakeTimestamp()-t, respond.HttpsCodeError))
		return
	}

	// 调用 pebble_service 的方法
	userTokens, err := pebble_service.GetUserTokenByMetaID(metaId)
	if err != nil {
		c.JSONP(http.StatusOK, respond.RespErr(err, tool.MakeTimestamp()-t, respond.HttpsCodeError))
		return
	}

	c.JSONP(http.StatusOK, respond.RespSuccess(userTokens, tool.MakeTimestamp()-t))
}

// GetUserTokensList godoc
// @Summary 获取用户推送令牌列表（分页）
// @Description 分页获取所有用户的推送令牌列表
// @Tags Push API
// @Produce json
// @Param page query int false "页码，默认为1" default(1)
// @Param pageSize query int false "每页大小，默认为10" default(10)
// @Success 200 {object} respond.Response{data=pebble_service.PaginatedUserTokens} "成功响应"
// @Failure 400 {object} respond.Response "参数错误"
// @Failure 401 {object} respond.Response "认证失败"
// @Failure 500 {object} respond.Response "服务器内部错误"
// @Router /v1/push/get_user_tokens_list [get]
func GetUserTokensList(c *gin.Context) {
	var t int64 = tool.MakeTimestamp()

	// 从 query 参数获取分页信息
	page := 1
	pageSize := 10

	if pageStr := c.Query("page"); pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}

	if pageSizeStr := c.Query("pageSize"); pageSizeStr != "" {
		if ps, err := strconv.Atoi(pageSizeStr); err == nil && ps > 0 {
			pageSize = ps
		}
	}

	// 调用 pebble_service 的方法
	result, err := pebble_service.GetUserTokensList(page, pageSize)
	if err != nil {
		c.JSONP(http.StatusOK, respond.RespErr(err, tool.MakeTimestamp()-t, respond.HttpsCodeError))
		return
	}

	c.JSONP(http.StatusOK, respond.RespSuccess(result, tool.MakeTimestamp()-t))
}

// RemoveUserToken godoc
// @Summary 移除用户推送令牌
// @Description 移除指定用户在指定平台的推送令牌
// @Tags Push API
// @Accept json
// @Produce json
// @Param request body request.RemoveUserTokenReq true "请求参数"
// @Success 200 {object} respond.Response "成功响应"
// @Failure 400 {object} respond.Response "参数错误"
// @Failure 401 {object} respond.Response "认证失败"
// @Failure 500 {object} respond.Response "服务器内部错误"
// @Router /v1/push/remove_user_token [post]
func RemoveUserToken(c *gin.Context) {
	var (
		t            int64 = tool.MakeTimestamp()
		requestModel *request.RemoveUserTokenReq
	)

	if c.ShouldBindJSON(&requestModel) == nil {
		// 调用 pebble_service 的方法
		err := pebble_service.RemoveUserToken(requestModel.MetaID, requestModel.Platform)
		if err != nil {
			c.JSONP(http.StatusOK, respond.RespErr(err, tool.MakeTimestamp()-t, respond.HttpsCodeError))
			return
		}

		// 构造成功响应
		responseData := map[string]interface{}{
			"success": true,
			"message": "用户令牌移除成功",
		}

		c.JSONP(http.StatusOK, respond.RespSuccess(responseData, tool.MakeTimestamp()-t))
		return
	}

	c.JSONP(http.StatusInternalServerError, respond.RespErr(errors.New("参数错误"), tool.MakeTimestamp()-t, respond.HttpsCodeError))
}

// RemoveUserAllTokens godoc
// @Summary 移除用户所有推送令牌
// @Description 移除指定用户的所有推送令牌
// @Tags Push API
// @Accept json
// @Produce json
// @Param request body request.RemoveUserAllTokensReq true "请求参数"
// @Success 200 {object} respond.Response "成功响应"
// @Failure 400 {object} respond.Response "参数错误"
// @Failure 401 {object} respond.Response "认证失败"
// @Failure 500 {object} respond.Response "服务器内部错误"
// @Router /v1/push/remove_user_all_tokens [post]
func RemoveUserAllTokens(c *gin.Context) {
	var (
		t            int64 = tool.MakeTimestamp()
		requestModel *request.RemoveUserAllTokensReq
	)

	if c.ShouldBindJSON(&requestModel) == nil {
		// 调用 pebble_service 的方法
		err := pebble_service.RemoveUserAllTokens(requestModel.MetaID)
		if err != nil {
			c.JSONP(http.StatusOK, respond.RespErr(err, tool.MakeTimestamp()-t, respond.HttpsCodeError))
			return
		}

		// 构造成功响应
		responseData := map[string]interface{}{
			"success": true,
			"message": "用户所有令牌移除成功",
		}

		c.JSONP(http.StatusOK, respond.RespSuccess(responseData, tool.MakeTimestamp()-t))
		return
	}

	c.JSONP(http.StatusInternalServerError, respond.RespErr(errors.New("参数错误"), tool.MakeTimestamp()-t, respond.HttpsCodeError))
}

// ===== 屏蔽聊天相关API接口 =====

// GetUserBlockedChats godoc
// @Summary 获取用户屏蔽聊天列表
// @Description 根据用户 metaId 获取该用户屏蔽的聊天列表
// @Tags Push API
// @Produce json
// @Param metaId query string true "用户唯一标识"
// @Success 200 {object} respond.Response{data=models.UserBlockedChats} "成功响应"
// @Failure 400 {object} respond.Response "参数错误"
// @Failure 401 {object} respond.Response "认证失败"
// @Failure 500 {object} respond.Response "服务器内部错误"
// @Router /v1/push/get_user_blocked_chats [get]
func GetUserBlockedChats(c *gin.Context) {
	var t int64 = tool.MakeTimestamp()

	// 从 query 参数获取 metaId
	metaId := c.Query("metaId")
	if metaId == "" {
		c.JSONP(http.StatusOK, respond.RespErr(errors.New("metaId 参数不能为空"), tool.MakeTimestamp()-t, respond.HttpsCodeError))
		return
	}

	// 调用 pebble_service 的方法
	userBlockedChats, err := pebble_service.GetUserBlockedChats(metaId)
	if err != nil {
		c.JSONP(http.StatusOK, respond.RespErr(err, tool.MakeTimestamp()-t, respond.HttpsCodeError))
		return
	}

	c.JSONP(http.StatusOK, respond.RespSuccess(userBlockedChats, tool.MakeTimestamp()-t))
}

// AddBlockedChat godoc
// @Summary 添加屏蔽聊天
// @Description 为用户添加屏蔽某个群聊或私聊
// @Tags Push API
// @Accept json
// @Produce json
// @Param request body request.AddBlockedChatReq true "请求参数"
// @Success 200 {object} respond.Response "成功响应"
// @Failure 400 {object} respond.Response "参数错误"
// @Failure 401 {object} respond.Response "认证失败"
// @Failure 500 {object} respond.Response "服务器内部错误"
// @Router /v1/push/add_blocked_chat [post]
func AddBlockedChat(c *gin.Context) {
	var (
		t            int64 = tool.MakeTimestamp()
		requestModel *request.AddBlockedChatReq
	)

	if c.ShouldBindJSON(&requestModel) == nil {
		// 调用 pebble_service 的方法
		err := pebble_service.AddBlockedChat(requestModel.MetaID, requestModel.ChatID, requestModel.ChatType, requestModel.Reason)
		if err != nil {
			c.JSONP(http.StatusOK, respond.RespErr(err, tool.MakeTimestamp()-t, respond.HttpsCodeError))
			return
		}

		// 构造成功响应
		responseData := map[string]interface{}{
			"success": true,
			"message": "屏蔽聊天添加成功",
			"data": map[string]interface{}{
				"metaId":   requestModel.MetaID,
				"chatId":   requestModel.ChatID,
				"chatType": requestModel.ChatType,
				"reason":   requestModel.Reason,
			},
		}

		c.JSONP(http.StatusOK, respond.RespSuccess(responseData, tool.MakeTimestamp()-t))
		return
	}

	c.JSONP(http.StatusInternalServerError, respond.RespErr(errors.New("参数错误"), tool.MakeTimestamp()-t, respond.HttpsCodeError))
}

// RemoveBlockedChat godoc
// @Summary 移除屏蔽聊天
// @Description 移除用户对某个群聊或私聊的屏蔽
// @Tags Push API
// @Accept json
// @Produce json
// @Param request body request.RemoveBlockedChatReq true "请求参数"
// @Success 200 {object} respond.Response "成功响应"
// @Failure 400 {object} respond.Response "参数错误"
// @Failure 401 {object} respond.Response "认证失败"
// @Failure 500 {object} respond.Response "服务器内部错误"
// @Router /v1/push/remove_blocked_chat [post]
func RemoveBlockedChat(c *gin.Context) {
	var (
		t            int64 = tool.MakeTimestamp()
		requestModel *request.RemoveBlockedChatReq
	)

	if c.ShouldBindJSON(&requestModel) == nil {
		// 调用 pebble_service 的方法
		err := pebble_service.RemoveBlockedChat(requestModel.MetaID, requestModel.ChatID)
		if err != nil {
			c.JSONP(http.StatusOK, respond.RespErr(err, tool.MakeTimestamp()-t, respond.HttpsCodeError))
			return
		}

		// 构造成功响应
		responseData := map[string]interface{}{
			"success": true,
			"message": "屏蔽聊天移除成功",
			"data": map[string]interface{}{
				"metaId": requestModel.MetaID,
				"chatId": requestModel.ChatID,
			},
		}

		c.JSONP(http.StatusOK, respond.RespSuccess(responseData, tool.MakeTimestamp()-t))
		return
	}

	c.JSONP(http.StatusInternalServerError, respond.RespErr(errors.New("参数错误"), tool.MakeTimestamp()-t, respond.HttpsCodeError))
}
