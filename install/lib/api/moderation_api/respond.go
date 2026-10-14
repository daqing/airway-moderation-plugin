package moderation_api

import (
	"github.com/daqing/airway/lib/render"
	"github.com/gin-gonic/gin"
)

// Business codes this API reports in the response envelope. Transport stays
// HTTP 200 throughout, as the framework's render helpers do.
const (
	codeInvalidRequest = 30001
	codeUnauthorized   = 30002
	codeWordList       = 30003
)

func fail(c *gin.Context, code int, message string) {
	render.ErrorCodeMsg(c, code, message)
}
