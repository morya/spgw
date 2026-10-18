package middle

import (
	"bytes"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// BodyDump 缓存请求体与响应体，请求结束后以 access-log 形式打印。
// 字段与原 echo BodyDump 一致：remote_ip / req / rsp。
func BodyDump() gin.HandlerFunc {
	return func(c *gin.Context) {
		var reqBytes []byte
		if c.Request.Body != nil {
			reqBytes, _ = io.ReadAll(c.Request.Body)
			c.Request.Body = io.NopCloser(bytes.NewBuffer(reqBytes))
		}

		bw := &bodyDumpWriter{
			ResponseWriter: c.Writer,
			buf:            &bytes.Buffer{},
		}
		c.Writer = bw

		c.Next()

		zap.S().With("remote_ip", c.ClientIP()).
			With(zap.Any("req", string(reqBytes))).
			With(zap.Any("rsp", bw.buf.String())).
			Info("access-log")
	}
}

// BodyLimit 限制请求体最大字节数。等价于 echo middleware.BodyLimit。
func BodyLimit(max int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.Body != nil {
			c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, max)
		}
		c.Next()
	}
}

type bodyDumpWriter struct {
	gin.ResponseWriter
	buf *bytes.Buffer
}

func (w *bodyDumpWriter) Write(b []byte) (int, error) {
	w.buf.Write(b)
	return w.ResponseWriter.Write(b)
}

func (w *bodyDumpWriter) WriteString(s string) (int, error) {
	w.buf.WriteString(s)
	return w.ResponseWriter.WriteString(s)
}
