package controller

import (
	"fmt"
	"net/http"
	"push-base-service/conf"
	"push-base-service/controller/auth"

	_ "push-base-service/docs" // 导入生成的 swagger 文档

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

func Run() {
	//err := middleware.InitClient()
	//if err != nil {
	//	panic(err)
	//}
	//fmt.Println("Redis connect success for ip rate.")

	router := gin.Default()
	router.Use(Cors())
	router.Use(Logger())
	//router.Use(middleware.ResponseTime())

	//limiter := middleware.NewIPRateLimiter(100*time.Millisecond, 1000*1000*1000*1000)
	//router.Use(middleware.IPRateLimitMiddleware(limiter))

	// Swagger 文档路由
	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	v1 := router.Group("/v1")
	{
		// 应用 API Key 鉴权中间件到所有 Push API 路由
		pushGroup := v1.Group("/push")
		{
			pushGroup.POST("/set_user_tokens", auth.AuthSignMiddleware(), SetUserTokens)
			// pushGroup.POST("/set_user_tokens", SetUserTokens)
			pushGroup.GET("/get_user_token", GetUserTokenByMetaID)
			pushGroup.GET("/get_user_tokens_list", GetUserTokensList)
			pushGroup.POST("/remove_user_token", RemoveUserToken)
			pushGroup.POST("/remove_user_all_tokens", RemoveUserAllTokens)

			pushGroup.GET("/get_user_blocked_chats", GetUserBlockedChats)
			pushGroup.POST("/add_blocked_chat", AddBlockedChat)
			pushGroup.POST("/remove_blocked_chat", RemoveBlockedChat)
		}
	}

	_ = router.Run(fmt.Sprintf("0.0.0.0:%s", conf.Port))
}

func Cors() gin.HandlerFunc {
	return func(c *gin.Context) {
		method := c.Request.Method
		//origin := c.Request.Header.Get("Origin")
		//if origin != "" {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
		c.Header("Access-Control-Allow-Headers", "Content-Type,AccessToken,X-CSRF-Token, Authorization,X-API-KEY,X-Signature,X-Public-Key")
		c.Header("Access-Control-Allow-Credentials", "true")
		c.Set("content-type", "application/json")
		//}
		if method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
		}
		c.Next()
	}
}

func Logger() gin.HandlerFunc {
	return func(context *gin.Context) {
		//context.JSON(http.StatusUnauthorized, gin.H{"status": "unauthorized"})
		//context.Abort()
		context.Next()
	}
}

func Handle(r *gin.Engine, httpMethods []string, relativePath string, handlers ...gin.HandlerFunc) gin.IRoutes {
	var routes gin.IRoutes
	for _, httpMethod := range httpMethods {
		routes = r.Handle(httpMethod, relativePath, handlers...)
	}
	return routes
}
