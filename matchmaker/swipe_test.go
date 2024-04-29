package matchmaker

import (
	"bytes"
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

	_ "github.com/mattn/go-sqlite3"
)

// Creates a match where 2 users have 'swiped' each other
func createMatchRecord(tx *sql.Tx, userID1 int, userID2 int) error {

	u1 := userID1
	u2 := userID2

	if userID2 < userID1 {
		u1 = userID2
		u2 = userID1
	}

	_, err := tx.Exec("INSERT INTO matches (user1, user2) VALUES (?, ?)",
		u1,
		u2)

	if err != nil {
		return err
	}

	return nil
}

func TestSwipeHandler(t *testing.T) {

	db, err := sql.Open("sqlite3", "./swipe-test.db")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	defer os.Remove("./swipe-test.db")

	if _, err := db.Exec(store.SchemaSQL); err != nil {
		t.Fatal(err)
	}

	tx, err := db.Begin()
	if err != nil {
		t.Fatal("failed to start transaction: ", err)
	}

	if err = createMatchRecord(tx, 2, 1); err != nil {
		t.Fatal("failed to insert match: ", err)
	}

	if err = createSwipeRecordInTransaction(tx, 3, 4, true); err != nil {
		t.Fatal("failed to create swipe", err)
	}

	if err = createSwipeRecordInTransaction(tx, 5, 4, false); err != nil {
		t.Fatal("failed to create swipe", err)
	}

	if err = tx.Commit(); err != nil {
		t.Fatal("failed to commit setup code: ", err)
	}

	tests := []struct {
		name           string
		ctx            context.Context
		reqBody        string
		expectedStatus int
		expectedMatch  bool
	}{
		{
			name:           "Invalid payload",
			ctx:            context.Background(),
			reqBody:        `invalid_json_payload`,
			expectedStatus: http.StatusBadRequest,
			expectedMatch:  false,
		},
		{
			name:           "no user id given on context",
			ctx:            context.Background(),
			reqBody:        `{"other_user_id": 1, "like": false}`,
			expectedStatus: http.StatusUnauthorized,
			expectedMatch:  false,
		},
		{
			name:           "Returns match when there is an existing match record between user 1 and user 2",
			ctx:            middleware.SetClaimsOnContext(context.Background(), auth.JWTClaims{UserID: 1}),
			reqBody:        `{"other_user_id": 2, "like": true}`,
			expectedStatus: http.StatusOK,
			expectedMatch:  true,
		},
		{
			name:           "Returns a match when there is an existing swipe on user 4 from user 3",
			ctx:            middleware.SetClaimsOnContext(context.Background(), auth.JWTClaims{UserID: 4}),
			reqBody:        `{"other_user_id": 3, "like": true}`,
			expectedStatus: http.StatusOK,
			expectedMatch:  true,
		},
		{
			name:           "Returns without a match when only one user swipes",
			ctx:            middleware.SetClaimsOnContext(context.Background(), auth.JWTClaims{UserID: 1}),
			reqBody:        `{"other_user_id": 4, "like": true}`,
			expectedStatus: http.StatusOK,
			expectedMatch:  false,
		},
		{
			name:           "Returns without a match when one user swipes YES but the other has swiped with NO",
			ctx:            middleware.SetClaimsOnContext(context.Background(), auth.JWTClaims{UserID: 4}),
			reqBody:        `{"other_user_id": 5, "like": true}`,
			expectedStatus: http.StatusOK,
			expectedMatch:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequestWithContext(tt.ctx, "POST", "/swipe", bytes.NewBufferString(tt.reqBody))
			if err != nil {
				t.Fatal(err)
			}

			rr := httptest.NewRecorder()
			http.HandlerFunc(SwipeHandler(SwipeHandlerDeps{DB: db})).ServeHTTP(rr, req)

			if rr.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, rr.Code)
			}

			if rr.Code == http.StatusOK {
				var res SwipeResponse
				if err := json.Unmarshal(rr.Body.Bytes(), &res); err != nil {
					t.Fatal(err)
				}
				if res.Results.Matched != tt.expectedMatch {
					t.Errorf("expected matched %v, got %v", tt.expectedMatch, res.Results.Matched)
				}
			}
		})
	}
}
