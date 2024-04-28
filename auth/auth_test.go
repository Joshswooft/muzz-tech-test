package auth

import (
	"errors"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func TestGenerateJWTToken(t *testing.T) {
	userID := 123
	expectedExpiration := time.Now().Add(time.Hour * 24).Unix()

	tokenAuth := NewTokenAuthenticator()
	tokenString, err := tokenAuth.GenerateJWTToken(userID)
	if err != nil {
		t.Fatalf("Error generating JWT token: %v", err)
	}

	token, err := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, jwt.ErrSignatureInvalid
		}
		return []byte(jwtSecretKey), nil
	})
	if err != nil || !token.Valid {
		t.Fatalf("Error parsing JWT token: %v", err)
	}

	claims, ok := token.Claims.(*JWTClaims)
	if !ok {
		t.Fatal("Invalid token claims")
	}

	if claims.UserID != userID {
		t.Errorf("Expected user ID %d, got %v", userID, claims.UserID)
	}

	if claims.ExpiresAt.Unix() != expectedExpiration {
		t.Errorf("Expected expiration time %d, got %v", expectedExpiration, claims.ExpiresAt)
	}

	if claims.Issuer != issuer {
		t.Errorf("invalid issuer: %s, got: %s", "muzz", claims.Issuer)
	}

}

// for testing invalid tokens
func generateInValidToken(withClaims bool) string {
	token := jwt.New(jwt.SigningMethodHS256)
	if withClaims {
		token = jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{"foo": "bar"})
	}
	tokenString, _ := token.SignedString([]byte(jwtSecretKey))
	return tokenString
}

func generateJWTWithNoUserID() string {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, JWTClaims{})
	tokenString, _ := token.SignedString([]byte(jwtSecretKey))

	return tokenString

}

func TestExtractClaimsFromToken(t *testing.T) {

	tokenAuth := NewTokenAuthenticator()
	userID := 1
	happyJWT, err := tokenAuth.GenerateJWTToken(userID)
	if err != nil {
		t.Fatal("failed to make jwt", err)
	}

	testCases := []struct {
		name           string
		token          string
		expectedUserID int
		expectedErr    error
	}{
		{
			name:           "No token given",
			expectedUserID: 0,
			expectedErr:    errNoJWTToken,
		},
		{
			name:           "Invalid - different set of claims given",
			token:          generateInValidToken(false),
			expectedUserID: 0,
			expectedErr:    errNoUserIDFoundOnJWT,
		},
		{
			name:           "Invalid no user ID given on claims",
			token:          generateJWTWithNoUserID(),
			expectedUserID: 0,
			expectedErr:    errNoUserIDFoundOnJWT,
		},
		{
			name:           "Valid token in request header",
			token:          happyJWT,
			expectedUserID: userID,
			expectedErr:    nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			claims, err := tokenAuth.ExtractClaimsFromToken(tc.token)

			if !errors.Is(err, tc.expectedErr) {
				t.Fatalf("Test case '%s': Expected err '%+v', got '%+v'", tc.name, tc.expectedErr, err)
			}

			if claims != nil {
				userID := claims.UserID

				if userID != tc.expectedUserID {
					t.Errorf("Test case '%s': Expected user ID %d, got %d", tc.name, tc.expectedUserID, userID)
				}

			}
		})
	}
}
