package middleware

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

var ErrUnauthorized = errors.New("unauthorized")

type AuthenticateFunc func(context.Context, string) (context.Context, error)

func Auth(authenticate AuthenticateFunc, required bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		if header == "" && !required {
			c.Next()
			return
		}
		parts := strings.Fields(header)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}
		ctx, err := authenticate(c.Request.Context(), parts[1])
		if err != nil {
			status := http.StatusInternalServerError
			if errors.Is(err, ErrUnauthorized) {
				status = http.StatusUnauthorized
			}
			c.AbortWithStatus(status)
			return
		}
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	}
}
