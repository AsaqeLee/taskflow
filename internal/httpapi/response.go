package httpapi

import (
	"net/http"

	"github.com/AsaqeLee/taskflow/internal/requestmeta"
	"github.com/gin-gonic/gin"
)

type ErrorBody struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type ErrorResponse struct {
	Error     ErrorBody `json:"error"`
	RequestID string    `json:"request_id,omitempty"`
}

func AbortError(c *gin.Context, status int, code, message string) {
	c.AbortWithStatusJSON(status, ErrorResponse{
		Error: ErrorBody{
			Code:    code,
			Message: message,
		},
		RequestID: requestmeta.RequestID(c.Request.Context()),
	})
}

func WriteError(c *gin.Context, status int, code, message string) {
	c.JSON(status, ErrorResponse{
		Error: ErrorBody{
			Code:    code,
			Message: message,
		},
		RequestID: requestmeta.RequestID(c.Request.Context()),
	})
}

func Unauthorized(c *gin.Context, code, message string) {
	c.Header("WWW-Authenticate", `Bearer realm="taskflow"`)
	AbortError(c, http.StatusUnauthorized, code, message)
}
