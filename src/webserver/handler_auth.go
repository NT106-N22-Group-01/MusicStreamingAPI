package webserver

import (
	"encoding/base64"
	"net/http"
	"strings"
	"time"

	"github.com/gbrlsnchs/jwt/v3"
	"gorm.io/gorm"
)

const (
	authRequiredJSON = `{"error": "authentication required"}`
)

// AuthHandler is a handler wrapper used for authentication. Its only job is
// to do the authentication and then pass the work to the Handler it wraps around.
// Possible methods for authentication:
//
//   - Basic Auth with the username and password
//   - Authorization Bearer JWT token
//   - JWT token in a session cookie
//   - JWT token as a query string
//
// Basic auth is preserved for backward compatibility. Needless to say, it so not
// a preferred method for authentication.
type AuthHandler struct {
	wrapped    http.Handler // The actual handler that does the APP Logic job
	db         *gorm.DB
	secret     string   // Secret used to craft and decode tokens
	exceptions []string // Paths which will be exempt from authentication
}

// NewAuthHandler returns a new AuthHandler.
func NewAuthHandler(
	wrapped http.Handler,
	secret string,
	db *gorm.DB,
	exceptions []string,
) *AuthHandler {
	return &AuthHandler{
		wrapped:    wrapped,
		secret:     secret,
		db:         db,
		exceptions: exceptions,
	}
}

// ServeHTTP implements the http.Handler interface and does the actual basic authenticate
// check for every request
func (hl *AuthHandler) ServeHTTP(writer http.ResponseWriter, req *http.Request) {
	if !hl.authenticated(req) {
		writer.Header().Set("Content-Type", "application/json; charset=utf-8")
		writer.WriteHeader(http.StatusUnauthorized)
		_, _ = writer.Write([]byte(authRequiredJSON))
		return
	}

	hl.wrapped.ServeHTTP(writer, req)
}

// Compares the authentication header with the stored user and passwords
// and returns true if they pass.
func (hl *AuthHandler) authenticated(r *http.Request) bool {
	for _, path := range hl.exceptions {
		if strings.HasPrefix(r.URL.Path, path) {
			return true
		}
	}

	authHeader := r.Header.Get("Authorization")

	if strings.HasPrefix(authHeader, "Bearer ") {
		return hl.withJWT(strings.TrimPrefix(authHeader, "Bearer "))
	}

	if strings.HasPrefix(authHeader, "Basic ") {
		return hl.withBasicAuth(strings.TrimPrefix(authHeader, "Basic "))
	}

	return false
}

func (hl *AuthHandler) withBasicAuth(encoded string) bool {
	b, err := base64.StdEncoding.DecodeString(strings.TrimSpace(encoded))
	if err != nil {
		return false
	}

	pair := strings.SplitN(string(b), ":", 2)

	if len(pair) != 2 {
		return false
	}

	return checkLoginCreds(pair[0], pair[1], hl.db)
}

func (hl *AuthHandler) withJWT(token string) bool {
	var jot jwt.Payload

	alg := jwt.NewHS256([]byte(hl.secret))
	exp := jwt.ExpirationTimeValidator(time.Now())
	validatePayload := jwt.ValidatePayload(&jot, exp)

	_, err := jwt.Verify([]byte(token), alg, &jot, validatePayload)
	return err == nil
}
