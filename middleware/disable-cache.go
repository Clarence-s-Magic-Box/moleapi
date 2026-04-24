package middleware

import "github.com/gin-gonic/gin"

func SetNoStoreHeaders(c *gin.Context) {
	c.Header("Cache-Control", "no-store, no-cache, must-revalidate, private, max-age=0")
	c.Header("Pragma", "no-cache")
	c.Header("Expires", "0")
}

func DisableCache() gin.HandlerFunc {
	return func(c *gin.Context) {
		SetNoStoreHeaders(c)
		c.Next()
	}
}
