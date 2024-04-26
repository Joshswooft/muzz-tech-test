package main

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

func TestCreateUserHandler(t *testing.T) {
	db, err := sql.Open("sqlite3", "./test.db")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	defer os.Remove("./test.db")

	if _, err := db.Exec(schema); err != nil {
		t.Fatal(err)
	}

	req, err := http.NewRequest("GET", "/user/create", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	userHandler := handler{db: db}
	handler := http.HandlerFunc(userHandler.createUserHandler)

	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}

	expectedContentType := "application/json"
	if contentType := rr.Header().Get("Content-Type"); contentType != expectedContentType {
		t.Errorf("handler returned unexpected content type: got %v want %v",
			contentType, expectedContentType)
	}

	var response CreateUserResponse

	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Errorf("error parsing response body: %v", err)
	}

	if response.Result.Email == "" ||
		response.Result.Password == "" ||
		response.Result.Name == "" ||
		response.Result.Gender == "" ||
		response.Result.Age == 0 {
		t.Errorf("handler returned incomplete user data: %v", response)
	}
}

func TestMain(m *testing.M) {
	os.Exit(m.Run())
}
