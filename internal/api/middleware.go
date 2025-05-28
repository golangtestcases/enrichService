package api

import (
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

func ErrorMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		if len(c.Errors) > 0 {
			err := c.Errors.Last()
			switch e := err.Err.(type) {
			case ErrorResponse:
				c.JSON(e.Code, e)
			case validator.ValidationErrors:
				HandleValidationError(c, e)
			default:
				HandleError(c, e)
			}
			c.Abort()
			return
		}
	}
}
