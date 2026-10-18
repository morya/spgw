package main

import (
	"time"

	"github.com/gin-gonic/gin"
)

type APIResp struct {
	Code      string `json:"code"`
	Msg       string `json:"msg"`
	Success   bool   `json:"success"`
	Timestamp int64  `json:"timestamp"`
	Data      any    `json:"data,omitempty"`
}

func ok(c *gin.Context, data any) {
	c.JSON(200, APIResp{
		Code:      "0",
		Msg:       "OK",
		Success:   true,
		Timestamp: time.Now().UnixMilli(),
		Data:      data,
	})
}

func fail(c *gin.Context, code, msg string) {
	c.JSON(200, APIResp{
		Code:      code,
		Msg:       msg,
		Success:   false,
		Timestamp: time.Now().UnixMilli(),
	})
}
