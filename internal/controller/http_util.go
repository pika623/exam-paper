package controller

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func bindJSON(c *gin.Context, target any) bool {
	if err := c.ShouldBindJSON(target); err != nil {
		writeError(c, "请求格式不正确。", http.StatusBadRequest)
		return false
	}
	return true
}

func writeError(c *gin.Context, message string, status int) {
	c.Header("Cache-Control", "no-store")
	c.JSON(status, gin.H{"error": message})
}

func writeJSON(c *gin.Context, payload any) {
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, payload)
}
