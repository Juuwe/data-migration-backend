package auth

import (
	"errors"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func TestJWTValidation(t *testing.T) {
	if _, err := NewManager("short"); err == nil {
		t.Fatal("short signing secret accepted")
	}
	secret := strings.Repeat("s", 32)
	manager, err := NewManager(secret)
	if err != nil {
		t.Fatal(err)
	}
	token, expiresAt, err := manager.Issue(42)
	if err != nil || expiresAt.Before(time.Now()) {
		t.Fatalf("issued token expiry = %v, error = %v", expiresAt, err)
	}
	if id, err := manager.UserID(token); err != nil || id != 42 {
		t.Fatalf("token user ID = %d, error = %v", id, err)
	}
	if _, _, err := manager.Issue(0); err == nil {
		t.Fatal("issued token for user ID 0")
	}
	other, _ := NewManager(strings.Repeat("x", 32))
	if _, err := other.UserID(token); !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("wrong secret error = %v", err)
	}

	for _, test := range []struct {
		name   string
		method jwt.SigningMethod
		claims jwt.RegisteredClaims
	}{
		{"expired", jwt.SigningMethodHS256, jwt.RegisteredClaims{Issuer: issuer, Subject: "42", ExpiresAt: jwt.NewNumericDate(time.Now().Add(-time.Minute))}},
		{"missing expiry", jwt.SigningMethodHS256, jwt.RegisteredClaims{Issuer: issuer, Subject: "42"}},
		{"wrong issuer", jwt.SigningMethodHS256, jwt.RegisteredClaims{Issuer: "other", Subject: "42", ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour))}},
		{"invalid subject", jwt.SigningMethodHS256, jwt.RegisteredClaims{Issuer: issuer, Subject: strconv.FormatInt(0, 10), ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour))}},
		{"wrong algorithm", jwt.SigningMethodHS512, jwt.RegisteredClaims{Issuer: issuer, Subject: "42", ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour))}},
	} {
		t.Run(test.name, func(t *testing.T) {
			signed, err := jwt.NewWithClaims(test.method, test.claims).SignedString([]byte(secret))
			if err != nil {
				t.Fatal(err)
			}
			if _, err := manager.UserID(signed); !errors.Is(err, ErrInvalidToken) {
				t.Fatalf("invalid token accepted: %v", err)
			}
		})
	}
}
