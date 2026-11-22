package handler

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"

	"github.com/gin-gonic/gin"
)

const (
	feedSessionCookie = "data_migration_feed_session"
	feedSessionTTL    = 24 * 60 * 60
)

func feedSessionID(c *gin.Context) string {
	if value, err := c.Cookie(feedSessionCookie); err == nil && validFeedSessionID(value) {
		return value
	}

	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return ""
	}
	sessionID := hex.EncodeToString(raw)

	secure := c.Request.TLS != nil || c.GetHeader("X-Forwarded-Proto") == "https"
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(feedSessionCookie, sessionID, feedSessionTTL, "/", "", secure, true)
	return sessionID
}

func validFeedSessionID(value string) bool {
	if len(value) != 64 {
		return false
	}
	_, err := hex.DecodeString(value)
	return err == nil
}
