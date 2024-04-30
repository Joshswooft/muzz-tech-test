package matchmaker

import (
	"database/sql"
	"encoding/json"
	"muzz/httpresponse"
	"muzz/middleware"
	"muzz/user"
	"net/http"
	"strconv"
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

// handler for getting all the potential matches for a given user excluding profiles who the user has already swiped for
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

		ageFilterStr := r.URL.Query().Get("age")
		genderFilter := r.URL.Query().Get("gender")

		var ageFilter int
		var err error

		if ageFilterStr != "" {
			ageFilter, err = strconv.Atoi(ageFilterStr)

			if err != nil {
				w.WriteHeader(http.StatusBadRequest)
				json.NewEncoder(w).Encode(httpresponse.ErrorResponse{Error: "Age filter must be a number"})
				return
			}

			if ageFilter < 0 {
				w.WriteHeader(http.StatusBadRequest)
				json.NewEncoder(w).Encode(httpresponse.ErrorResponse{Error: "Age filter must be more than 0"})
			}
		}

		userProfiles, err := getPotentialMatches(deps.DB, userID, deps.now(), filters{age: ageFilter, gender: genderFilter})
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(httpresponse.ErrorResponse{Error: "failed to get matches"})
			return
		}

		response := DiscoverResponse{Results: userProfiles}
		json.NewEncoder(w).Encode(response)
	}
}

type filters struct {
	age    int
	gender string
}

// Retrieve userProfiles from the database excluding the current user and the profiles the user has already swiped on
// Assumes all the profiles will fit in memory!
func getPotentialMatches(db *sql.DB, userID int, now time.Time, filters filters) ([]profile, error) {

	// assumes we only care about the year of the date of birth for simplicity
	query := `
	SELECT 
	id, name, gender, dob, strftime('%Y', date('now')) - strftime('%Y', date(dob)) AS age 
	FROM users 
	WHERE id NOT IN (SELECT swipe_target FROM swipes WHERE swiper = ?) AND id != ?`

	params := []interface{}{userID, userID}

	if filters.age != 0 {
		query += " AND age = ?"
		params = append(params, filters.age)
	}

	if filters.gender != "" {
		query += " AND gender = ?"
		params = append(params, filters.gender)
	}

	rows, err := db.Query(query, params...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var userProfiles []profile

	for rows.Next() {
		var u user.User
		var age int
		if err := rows.Scan(&u.ID, &u.Name, &u.Gender, &u.DOB, &age); err != nil {
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
