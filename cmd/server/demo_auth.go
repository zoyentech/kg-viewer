package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// demoAuthMiddleware requires a server-signed, time-limited token on every
// route when DEMO_SECRET is configured. Invalid requests get a plain 404 so
// crawlers cannot tell protected resources from missing ones.
func demoAuthMiddleware(secret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if secret == "" {
			c.Next()
			return
		}
		token := c.Query("token")
		if token == "" {
			token = c.GetHeader("X-Demo-Token")
		}
		if validDemoToken(token, secret) {
			c.Next()
			return
		}
		c.AbortWithStatus(http.StatusNotFound)
	}
}

// validDemoToken checks a token of the form "<expiryUnix>.<hex hmac-sha256>".
func validDemoToken(token, secret string) bool {
	exp, sig, ok := strings.Cut(token, ".")
	if !ok {
		return false
	}
	expUnix, err := strconv.ParseInt(exp, 10, 64)
	if err != nil || expUnix < time.Now().Unix() {
		return false
	}
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(exp))
	expected := hex.EncodeToString(mac.Sum(nil))
	return subtle.ConstantTimeCompare([]byte(sig), []byte(expected)) == 1
}
