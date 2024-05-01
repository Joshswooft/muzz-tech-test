package middleware

import (
	"context"
	"errors"
	"muzz/auth"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetClaimsFromContext(t *testing.T) {

	sampleClaims := auth.JWTClaims{UserID: 123}

	type args struct {
		ctx context.Context
	}
	tests := []struct {
		name  string
		args  args
		want  auth.JWTClaims
		found bool
	}{
		{
			name:  "context with no claims",
			args:  args{ctx: context.Background()},
			want:  auth.JWTClaims{},
			found: false,
		},
		{
			name:  "context with claims",
			args:  args{ctx: context.WithValue(context.Background(), contextKey(claimsContextKey), sampleClaims)},
			want:  sampleClaims,
			found: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, found := GetClaimsFromContext(tt.args.ctx)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetClaimsFromContext() got = %v, want %v", got, tt.want)
			}
			if found != tt.found {
				t.Errorf("GetClaimsFromContext() found = %v, want %v", found, tt.found)
			}
		})
	}
}

func TestAuthGuard(t *testing.T) {

	testCases := []struct {
		name         string
		header       string
		expectedCode int
		extractor    ExtractClaimsFromToken
	}{
		{
			name:         "Empty Authorization header",
			header:       "",
			expectedCode: http.StatusUnauthorized,
			extractor:    func(tokenString string) (*auth.JWTClaims, error) { return nil, nil },
		},
		{
			name:         "Invalid Authorization header format",
			header:       "invalid_format",
			expectedCode: http.StatusUnauthorized,
			extractor:    func(tokenString string) (*auth.JWTClaims, error) { return nil, nil },
		},
		{
			name:         "Valid Authorization header with invalid token",
			header:       "Bearer invalid_token",
			expectedCode: http.StatusUnauthorized,
			extractor: func(tokenString string) (*auth.JWTClaims, error) {
				return nil, errors.New("token couldnt be extracted")
			},
		},
		{
			name:         "Valid Authorization header with valid token",
			header:       "Bearer my-token",
			expectedCode: http.StatusOK,
			extractor: func(tokenString string) (*auth.JWTClaims, error) {
				assert.Equal(t, "my-token", tokenString)
				return &auth.JWTClaims{}, nil
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create a request with the test header
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.Header.Set("Authorization", tc.header)

			// Create a response recorder to capture the response
			rr := httptest.NewRecorder()

			// Call the AuthGuard middleware with a dummy handler
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})
			middleware := NewAuthGuardMiddleware(tc.extractor)
			authGuard := middleware(handler)
			authGuard.ServeHTTP(rr, req)

			// Check the response status code
			if rr.Code != tc.expectedCode {
				t.Errorf("Test case '%s': Expected status code %d, got %d", tc.name, tc.expectedCode, rr.Code)
			}
		})
	}
}

func TestSetClaimsOnContext(t *testing.T) {
	type args struct {
		ctx    context.Context
		claims *auth.JWTClaims
	}
	tests := []struct {
		name       string
		args       args
		wantClaims auth.JWTClaims
	}{
		{
			name:       "no claims being set",
			args:       args{ctx: context.Background()},
			wantClaims: auth.JWTClaims{},
		},
		{
			name:       "set claims",
			args:       args{ctx: context.Background(), claims: &auth.JWTClaims{UserID: 1}},
			wantClaims: auth.JWTClaims{UserID: 1},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := SetClaimsOnContext(tt.args.ctx, tt.wantClaims)

			claims, _ := GetClaimsFromContext(ctx)

			if !reflect.DeepEqual(claims, tt.wantClaims) {
				t.Errorf("claims not equal, want: %v, got: %v", tt.wantClaims, claims)
			}
		})
	}
}
