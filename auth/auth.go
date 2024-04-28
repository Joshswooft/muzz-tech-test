package auth

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const jwtSecretKey = "secret"
const issuer = "muzz"

// JWTClaims represents JWT claims
type JWTClaims struct {
	UserID int `json:"user_id"`
	jwt.RegisteredClaims
}

// Implements the ClaimsValidator interface
// gets called automatically when the token is being parsed
func (j *JWTClaims) Validate() error {
	if j.UserID == 0 {
		return errNoUserIDFoundOnJWT
	}
	return nil
}

// provides functional-style options to modify the behaviour of the token authenticator.
type tokenAuthenticatorOption func(*tokenAuthenticator)

// handles creating and retrieving tokens
type tokenAuthenticator struct {
	// timeFunc is used to supply the current time that is needed for
	// token validation. If unspecified, this defaults to time.Now.
	timeFunc func() time.Time
}

// Mainly used for testing to offset the clock
func WithTimeFunc(timeFunc func() time.Time) tokenAuthenticatorOption {
	return func(t *tokenAuthenticator) {
		t.timeFunc = timeFunc
	}
}

// Handles creating and retrieving tokens
// Pass in some optional function options to modify the behaviour of the token auth
func NewTokenAuthenticator(options ...tokenAuthenticatorOption) *tokenAuthenticator {
	t := &tokenAuthenticator{}

	for _, option := range options {
		option(t)
	}
	return t
}

// generates a JWT token with the given user ID
func (t *tokenAuthenticator) GenerateJWTToken(userID int) (string, error) {

	var now time.Time

	// Check, if we have a time func
	if t.timeFunc != nil {
		now = t.timeFunc()
	} else {
		now = time.Now()
	}

	claims := JWTClaims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			NotBefore: jwt.NewNumericDate(now),
			Issuer:    issuer,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(time.Hour * 24)),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(jwtSecretKey))
}

var (
	errNoJWTToken         = errors.New("no jwt token given")
	errNoUserIDFoundOnJWT = errors.New("no user ID on JWT token")
)

// gets the claims from the JWT token
// claims are checked for their validity otherwise an error is returned
func (t *tokenAuthenticator) ExtractClaimsFromToken(tokenString string) (*JWTClaims, error) {

	if tokenString == "" {
		return nil, errNoJWTToken
	}

	token, err := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, jwt.ErrSignatureInvalid
		}
		return []byte(jwtSecretKey), nil
	})
	if err != nil || !token.Valid {
		return nil, err
	}

	claims, ok := token.Claims.(*JWTClaims)

	if !ok {
		return nil, jwt.ErrTokenInvalidClaims
	}

	return claims, nil
}
