package auth

import (
	"github.com/gin-gonic/gin"
	"net/http"
)

// Cors 跨域服务，请求所有
func Cors() gin.HandlerFunc {
	return func(c *gin.Context) {
		method := c.Request.Method
		//请求任何域名都可以（不安全）
		origin := c.Request.Header.Get("Origin")
		if origin != "" {
			c.Header("Access-Control-Allow-Origin", origin)
			c.Header("Access-Control-Allow-Methods", "POST, GET, PUT, DELETE, OPTIONS")
			c.Header("Access-Control-Allow-Headers", "Content-Type, Content-Length, Token")
			c.Header("Access-Control-Expose-Headers", "Access-Control-Allow-Headers, Token")
			c.Header("Access-Control-Max-Age", "172800")
			c.Header("Access-Control-Allow-Credentials", "true")
			c.Set("content-type", "application/json")
		}
		if method == "OPTIONS" {
			//c.JSON(200, Controller.R(200, nil, "Options Request"))
			c.AbortWithStatus(http.StatusNoContent)
		}
		c.Next()
	}
}
