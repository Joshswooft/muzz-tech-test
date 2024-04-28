package middleware

import (
	"context"
	"errors"
	"muzz/auth"
	"net/http"
	"strings"
)

var (
	errInvalidBearerTokenFormat = errors.New("auth header must be in format 'Bearer <jwt>'")
)

type Middleware func(http.Handler) http.Handler

type contextKey string

func (c contextKey) String() string {
	return string(c)
}

const claimsContextKey = "claims"

// extracts the claims from a given token
type ExtractClaimsFromToken = func(tokenString string) (*auth.JWTClaims, error)

type authGuardMiddleware struct {
	ExtractClaimsFromToken
}

// Creates a new middleware which protects from un-authenticated users
func NewAuthGuardMiddleware(extractor ExtractClaimsFromToken) Middleware {
	m := authGuardMiddleware{ExtractClaimsFromToken: extractor}
	return m.authGuard
}

// Middlware Guards against unauthenticated requests, puts claims onto context to be used in request
func (m *authGuardMiddleware) authGuard(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		if !strings.HasPrefix(authHeader, "Bearer ") {
			http.Error(w, errInvalidBearerTokenFormat.Error(), http.StatusUnauthorized)
			return
		}

		tokenString := authHeader[len("Bearer "):]
		claims, err := m.ExtractClaimsFromToken(tokenString)

		if claims == nil || err != nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		r = r.WithContext(SetClaimsOnContext(r.Context(), *claims))

		next.ServeHTTP(w, r)
	})
}

// stores the claims onto the context
func SetClaimsOnContext(ctx context.Context, claims auth.JWTClaims) context.Context {
	return context.WithValue(ctx, contextKey(claimsContextKey), claims)
}

// returns the claims on the context and a boolean to indicate whether the claims were found
func GetClaimsFromContext(ctx context.Context) (auth.JWTClaims, bool) {
	claims, ok := ctx.Value(contextKey(claimsContextKey)).(auth.JWTClaims)
	return claims, ok
}
