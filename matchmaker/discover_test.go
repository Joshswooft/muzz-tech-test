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

func assertEqualProfiles(t *testing.T, expectedProfiles []*profile, profiles []*profile) bool {
	ok := true

	if len(expectedProfiles) != len(profiles) {
		t.Errorf("Expected %d users, got %d", len(expectedProfiles), len(profiles))
		return false
	}

	for i, expected := range expectedProfiles {
		profile := profiles[i]
		if profile.ID != expected.ID || profile.Name != expected.Name ||
			profile.Gender != expected.Gender || profile.Age != expected.Age || profile.DistanceFromMe != expected.DistanceFromMe || profile.attractivenessScore != expected.attractivenessScore {
			t.Errorf("Unexpected user profile. Expected: %+v, Got: %+v", expected, profile)
			ok = false
		}
	}
	return ok
}

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

	expectedUserProfiles := []*profile{
		{ID: 3, Name: "Charlie", Gender: "male", Age: 29},
		{ID: 4, Name: "Darren", Gender: "male", Age: 23},
	}

	assertEqualProfiles(t, expectedUserProfiles, response.Results)

}

func TestDiscoverHandlerSortByDistance(t *testing.T) {
	db, err := sql.Open("sqlite3", "./test.db")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	defer os.Remove("./test.db")

	if _, err := db.Exec(store.SchemaSQL); err != nil {
		t.Fatal(err)
	}

	if _, err := db.Exec(`
	INSERT INTO users (name, gender, dob, lat, lng) VALUES
    ('John Doe', 'male', '1990-05-15', 40.7128, -74.0060),
    ('Jane Smith', 'female', '1992-08-20', 41, -75),
    ('Alice Johnson', 'female', '1985-12-10', 51.5074, -0.1278),
    ('Bob Williams', 'male', '1988-03-25', 48.8566, 2.3522);
	`); err != nil {
		t.Fatal(err)
	}

	ctx := middleware.SetClaimsOnContext(context.Background(), auth.JWTClaims{UserID: 1})
	now := time.Date(2024, 04, 01, 0, 0, 0, 0, time.Local)

	req, err := http.NewRequestWithContext(ctx, "GET", "/discover", nil)
	if err != nil {
		t.Fatal(err)
	}

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

	expectedUserProfiles := []*profile{
		{ID: 2, Name: "Jane Smith", Gender: "female", Age: 31, DistanceFromMe: 89.48927940866334},
		{ID: 3, Name: "Alice Johnson", Gender: "female", Age: 38, DistanceFromMe: 5570.222179737958},
		{ID: 4, Name: "Bob Williams", Gender: "male", Age: 36, DistanceFromMe: 5837.240903825839},
	}

	assertEqualProfiles(t, expectedUserProfiles, response.Results)

}

func TestDiscoverHandlerSortByAttractiveness(t *testing.T) {
	db, err := sql.Open("sqlite3", "./test.db")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	defer os.Remove("./test.db")

	if _, err := db.Exec(store.SchemaSQL); err != nil {
		t.Fatal(err)
	}

	if _, err := db.Exec(`
	INSERT INTO users (name, gender, dob, lat, lng) VALUES
    ('John Doe', 'male', '1990-05-15', 0, 0),
    ('Jane Smith', 'female', '1992-08-20', 2, 1),
    ('Alice Johnson', 'female', '1985-12-10', 2, 1),
    ('Bob Williams', 'male', '1988-03-25', 3, 3),
	('Luke', 'male', '1988-03-25', -3, -3);

	INSERT INTO swipes (swiper, swipe_target, liked) VALUES 
	(2, 1, TRUE),
	(2, 3, TRUE),
	(4, 3, TRUE),
	(4, 5, TRUE),
	(2, 5, TRUE),
	(3, 4, TRUE);
	`); err != nil {
		t.Fatal("failed to setup db", err)
	}

	ctx := middleware.SetClaimsOnContext(context.Background(), auth.JWTClaims{UserID: 1})
	now := time.Date(2024, 04, 01, 0, 0, 0, 0, time.Local)

	req, err := http.NewRequestWithContext(ctx, "GET", "/discover", nil)
	if err != nil {
		t.Fatal(err)
	}

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

	expectedUserProfiles := []*profile{
		{ID: 3, Name: "Alice Johnson", Gender: "female", Age: 38, DistanceFromMe: 248.62931484681246},
		{ID: 2, Name: "Jane Smith", Gender: "female", Age: 31, DistanceFromMe: 248.62931484681246},
		{ID: 5, Name: "Luke", Gender: "male", Age: 36, DistanceFromMe: 471.65228849900205},
		{ID: 4, Name: "Bob Williams", Gender: "male", Age: 36, DistanceFromMe: 471.65228849900205},
	}

	assertEqualProfiles(t, expectedUserProfiles, response.Results)

}

func TestDiscoverHandlerFilters(t *testing.T) {
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
	('Alice', 'female', '1990-01-01'),
	('Bob', 'male', '1985-01-01'),
	('Charlie', 'male', '1995-01-01'),
	('Darren', 'male', '2000-05-04'),
	('Erica', 'other', '2000-05-04'),
	('Fran', 'female', '1980-01-01');

	INSERT INTO swipes (swiper, swipe_target, liked) VALUES (1, 2, TRUE);
	`); err != nil {
		t.Fatal(err)
	}

	ctx := middleware.SetClaimsOnContext(context.Background(), auth.JWTClaims{UserID: 1})
	now := time.Date(2024, 04, 01, 0, 0, 0, 0, time.Local)

	tests := []struct {
		name             string
		reqBody          string
		expectedResponse DiscoverResponse
	}{
		{
			name:    "filter by age",
			reqBody: "/discover?age=29",
			expectedResponse: DiscoverResponse{
				Results: []*profile{
					{ID: 3, Name: "Charlie", Gender: "male", Age: 29},
				},
			},
		},
		{
			name:    "filter by gender",
			reqBody: "/discover?gender=female",
			expectedResponse: DiscoverResponse{
				Results: []*profile{{ID: 6, Name: "Fran", Gender: "female", Age: 44}},
			},
		},
		{
			name:    "filter by age and gender",
			reqBody: "/discover?age=24&gender=male",
			expectedResponse: DiscoverResponse{
				Results: []*profile{
					{ID: 4, Name: "Darren", Gender: "male", Age: 23},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequestWithContext(ctx, "GET", tt.reqBody, nil)
			if err != nil {
				t.Fatal(err)
			}

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

			expectedUserProfiles := tt.expectedResponse.Results

			assertEqualProfiles(t, expectedUserProfiles, response.Results)

		})
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
