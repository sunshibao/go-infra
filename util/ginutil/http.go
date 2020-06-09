package ginutil

import "github.com/gin-gonic/gin"

func QueryOrForm(c *gin.Context, key string) string {
	return c.DefaultQuery(key, c.PostForm(key))
}

func DefaultQueryOrForm(c *gin.Context, key, defaultValue string) string {
	return c.DefaultQuery(key, c.DefaultPostForm(key, defaultValue))
}