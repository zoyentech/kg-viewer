// Package response provides the JSON envelope helpers shared by all
// kg-viewer HTTP handlers. The shape { code, data, msg } matches the
// healthDinner API contract so the viewer page works unchanged.
package response

import "github.com/gin-gonic/gin"

// Body is the standard JSON envelope.
type Body struct {
	Code int    `json:"code"`
	Data any    `json:"data,omitempty"`
	Msg  string `json:"msg,omitempty"`
}

// OK writes a 200 with code 0 and the given data.
func OK(c *gin.Context, data any) {
	c.JSON(200, Body{Code: 0, Data: data})
}

// Error writes the given HTTP status with a descriptive message.
func Error(c *gin.Context, status int, msg string) {
	c.JSON(status, Body{Code: status, Msg: msg})
}
