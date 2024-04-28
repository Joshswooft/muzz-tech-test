package auth

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"io"
	"muzz/store"
	"muzz/user"
	"net/http"
	"net/http/httptest"
	"testing"

	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
)

func TestLoginHandler(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	if _, err := db.Exec(store.SchemaSQL); err != nil {
		t.Fatal(err)
	}

	email := "test@example.com"
	password := "password123"

	testUser := user.User{
		Email:    email,
		Password: password,
	}
	if err := user.StoreUser(db, testUser); err != nil {
		t.Fatal(err)
	}

	req, err := http.NewRequest("POST", "/login", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()

	user := user.User{Email: email, Password: password}

	reqBody, err := json.Marshal(user)
	if err != nil {
		t.Fatal(err)
	}
	req.Body = io.NopCloser(bytes.NewReader(reqBody))

	handler := http.HandlerFunc(LoginHandler(LoginHandlerDeps{DB: db, JwtTokenGenerator: NewTokenAuthenticator().GenerateJWTToken}))

	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code, "handler returned wrong status code")

	expectedContentType := "application/json"
	assert.Equal(t, expectedContentType, rr.Header().Get("Content-Type"), "handler returned unexpected content type")

	var response map[string]string
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Errorf("error parsing response body: %v", err)
	}

	assert.NotNil(t, response["token"], "handler did not return token")
}
