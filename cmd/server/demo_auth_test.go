package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strconv"
	"testing"
	"time"
)

func TestValidDemoToken(t *testing.T) {
	secret := "demo-secret"
	exp := time.Now().Add(time.Hour).Unix()
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(strconv.FormatInt(exp, 10)))
	token := fmt.Sprintf("%d.%s", exp, hex.EncodeToString(mac.Sum(nil)))

	if !validDemoToken(token, secret) {
		t.Fatal("expected freshly signed token to be valid")
	}
	if validDemoToken(token, "other-secret") {
		t.Fatal("token signed with another secret must be rejected")
	}
	if validDemoToken(fmt.Sprintf("%d.bad", exp), secret) {
		t.Fatal("tampered signature must be rejected")
	}

	expired := time.Now().Add(-time.Hour).Unix()
	mac = hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(strconv.FormatInt(expired, 10)))
	oldToken := fmt.Sprintf("%d.%s", expired, hex.EncodeToString(mac.Sum(nil)))
	if validDemoToken(oldToken, secret) {
		t.Fatal("expired token must be rejected")
	}
}
