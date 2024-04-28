package matchmaker

import (
	"database/sql"
	"encoding/json"
	"muzz/httpresponse"
	"muzz/middleware"
	"muzz/user"
	"net/http"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

type profile struct {
	ID     int    `json:"id"`
	Name   string `json:"name"`
	Gender string `json:"gender"`
	Age    int    `json:"age"`
}

type DiscoverResponse struct {
	Results []profile `json:"results"`
}

type DiscoverHandlerDeps struct {
	DB    *sql.DB
	clock func() time.Time
}

// now is a time generator that falls back to std lib if clock is not specified
func (c *DiscoverHandlerDeps) now() time.Time {
	if c.clock == nil {
		return time.Now()
	}
	return c.clock()
}

// handler for getting all the potential matches for a given user
func DiscoverHandler(deps DiscoverHandlerDeps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		claims, found := middleware.GetClaimsFromContext(r.Context())

		if !found {
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(httpresponse.ErrorResponse{Error: "Unauthenticated"})
			return
		}

		userID := claims.UserID

		userProfiles, err := getMatchedUsers(deps.DB, userID, deps.now())
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(httpresponse.ErrorResponse{Error: "failed to get matches"})
			return
		}

		// Construct response
		response := DiscoverResponse{Results: userProfiles}

		// Encode response as JSON
		json.NewEncoder(w).Encode(response)
	}
}

// Retrieve userProfiles from the database excluding the current user
// Assumes all the profiles will fit in memory!
func getMatchedUsers(db *sql.DB, userID int, now time.Time) ([]profile, error) {
	rows, err := db.Query("SELECT id, name, gender, dob FROM users WHERE id != ?", userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var userProfiles []profile
	for rows.Next() {
		var u user.User
		if err := rows.Scan(&u.ID, &u.Name, &u.Gender, &u.DOB); err != nil {
			return nil, err
		}

		age, err := user.CalculateAge(u.DOB, now)
		if err != nil {
			return userProfiles, err
		}
		userProfile := profile{ID: u.ID, Name: u.Name, Gender: u.Gender, Age: age}
		userProfiles = append(userProfiles, userProfile)
	}

	return userProfiles, nil
}
