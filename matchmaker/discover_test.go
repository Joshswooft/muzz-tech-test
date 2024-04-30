package matchmaker

import (
	"context"
	"database/sql"
	"encoding/json"
	"muzz/auth"
	"muzz/middleware"
	"muzz/store"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

func TestDiscoverHandler(t *testing.T) {
	db, err := sql.Open("sqlite3", "./discover-test.db")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	defer os.Remove("./discover-test.db")

	if _, err := db.Exec(store.SchemaSQL); err != nil {
		t.Fatal(err)
	}

	if _, err := db.Exec(`
	INSERT INTO users (name, gender, dob) VALUES
	('Alice', 'female', '1990-01-01'),
	('Bob', 'male', '1985-01-01'),
	('Charlie', 'male', '1995-01-01'),
	('Darren', 'male', '2000-05-04');

	INSERT INTO swipes (swiper, swipe_target, liked) VALUES (1, 2, TRUE);
	`); err != nil {
		t.Fatal(err)
	}

	req, err := http.NewRequestWithContext(middleware.SetClaimsOnContext(context.Background(), auth.JWTClaims{UserID: 1}), "GET", "/discover", nil)
	if err != nil {
		t.Fatal(err)
	}

	now := time.Date(2024, 04, 01, 0, 0, 0, 0, time.Local)

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(DiscoverHandler(DiscoverHandlerDeps{clock: func() time.Time { return now }, DB: db}))
	handler.ServeHTTP(rr, req)

	wantStatusCode := http.StatusOK
	if rr.Code != wantStatusCode {
		t.Fatalf("Expected status code %d, got %d", wantStatusCode, rr.Code)
	}

	var response DiscoverResponse
	if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
		t.Fatal(err)
	}

	expectedUserProfiles := []profile{
		{ID: 3, Name: "Charlie", Gender: "male", Age: 29},
		{ID: 4, Name: "Darren", Gender: "male", Age: 23},
	}

	if len(response.Results) != len(expectedUserProfiles) {
		t.Errorf("Expected %d users, got %d", len(expectedUserProfiles), len(response.Results))
	}
	for i, expected := range expectedUserProfiles {
		if response.Results[i].ID != expected.ID || response.Results[i].Name != expected.Name ||
			response.Results[i].Gender != expected.Gender || response.Results[i].Age != expected.Age {
			t.Errorf("Unexpected user profile. Expected: %+v, Got: %+v", expected, response.Results[i])
		}
	}

}

func TestDiscoverHandlerNoProfiles(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	if _, err := db.Exec(store.SchemaSQL); err != nil {
		t.Fatal(err)
	}

	if _, err := db.Exec(`
	INSERT INTO users (name, gender, dob) VALUES
	('Alice', 'female', '1990-01-01');
	`); err != nil {
		t.Fatal(err)
	}

	req, err := http.NewRequestWithContext(middleware.SetClaimsOnContext(context.Background(), auth.JWTClaims{UserID: 1}), "GET", "/discover", nil)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2024, 04, 01, 0, 0, 0, 0, time.Local)

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(DiscoverHandler(DiscoverHandlerDeps{clock: func() time.Time { return now }, DB: db}))
	handler.ServeHTTP(rr, req)

	wantStatusCode := http.StatusOK
	if rr.Code != wantStatusCode {
		t.Fatalf("Expected status code %d, got %d", wantStatusCode, rr.Code)
	}

	var response DiscoverResponse
	if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
		t.Fatal(err)
	}

	expectedUserProfiles := []profile{}

	if len(response.Results) != len(expectedUserProfiles) {
		t.Errorf("Expected %d users, got %d", len(expectedUserProfiles), len(response.Results))
	}

}

func TestDiscoverHandlerNoJwtToken(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	if _, err := db.Exec(store.SchemaSQL); err != nil {
		t.Fatal(err)
	}

	if _, err := db.Exec(`
	INSERT INTO users (name, gender, dob) VALUES
	('Alice', 'female', '1990-01-01');
	`); err != nil {
		t.Fatal(err)
	}

	req, err := http.NewRequest("GET", "/discover", nil)
	if err != nil {
		t.Fatal(err)
	}

	now := time.Date(2024, 04, 01, 0, 0, 0, 0, time.Local)

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(DiscoverHandler(DiscoverHandlerDeps{clock: func() time.Time { return now }, DB: db}))
	handler.ServeHTTP(rr, req)

	wantStatusCode := http.StatusUnauthorized
	if rr.Code != wantStatusCode {
		t.Fatalf("Expected status code %d, got %d", wantStatusCode, rr.Code)
	}

}

func TestDiscoverHandlerFailedToGetMatches(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	// purposely creating a schema which doesn't match the expected database state
	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS users (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT,
		gender TEXT,
		foo INTEGER
	);
	`); err != nil {
		t.Fatal(err)
	}

	if _, err := db.Exec(`
	INSERT INTO users (name, gender, foo) VALUES
	('Alice', 'female', 1);
	`); err != nil {
		t.Fatal(err)
	}

	if err != nil {
		t.Fatal(err)
	}

	req, err := http.NewRequestWithContext(middleware.SetClaimsOnContext(context.Background(), auth.JWTClaims{UserID: 1}), "GET", "/discover", nil)
	if err != nil {
		t.Fatal(err)
	}

	now := time.Date(2024, 04, 01, 0, 0, 0, 0, time.Local)

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(DiscoverHandler(DiscoverHandlerDeps{clock: func() time.Time { return now }, DB: db}))
	handler.ServeHTTP(rr, req)

	wantStatusCode := http.StatusInternalServerError
	if rr.Code != wantStatusCode {
		t.Fatalf("Expected status code %d, got %d", wantStatusCode, rr.Code)
	}

}
