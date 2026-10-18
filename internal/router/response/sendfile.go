package response

import (
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
)

func ServeMemFile(c *gin.Context, filename string, body io.Reader) {
	c.Header("Content-Disposition", "attachment; filename="+filename)
	c.DataFromReader(http.StatusOK, -1, "application/octet-stream", body, nil)
}
