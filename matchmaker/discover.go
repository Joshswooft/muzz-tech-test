package matchmaker

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"math"
	"muzz/httpresponse"
	"muzz/middleware"
	"muzz/user"
	"net/http"
	"sort"
	"strconv"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

type profile struct {
	ID             int     `json:"id"`
	Name           string  `json:"name"`
	Gender         string  `json:"gender"`
	Age            int     `json:"age"`
	DistanceFromMe float64 `json:"distanceFromMe"`
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

		userLocation, err := getUserLocation(deps.DB, userID)
		if err != nil || userLocation == nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(httpresponse.ErrorResponse{Error: "failed to get matches"})
			return
		}

		userProfiles, err := getPotentialMatches(deps.DB, userID, deps.now(), filters{age: ageFilter, gender: genderFilter}, *userLocation)
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

func getUserLocation(db *sql.DB, userID int) (*user.GeoLocation, error) {
	var lat, lng sql.NullFloat64
	err := db.QueryRow("SELECT lat, lng FROM users WHERE id = ?", userID).Scan(&lat, &lng)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("user with ID %d not found", userID)
		}
		return nil, err
	}
	return &user.GeoLocation{Lat: lat, Long: lng}, nil
}

// Retrieve userProfiles from the database excluding the current user and the profiles the user has already swiped on
// Assumes all the profiles will fit in memory!
func getPotentialMatches(db *sql.DB, userID int, now time.Time, filters filters, userLocation user.GeoLocation) ([]profile, error) {

	// assumes we only care about the year of the date of birth for simplicity
	query := `
	SELECT 
	id, name, gender, dob, strftime('%Y', date('now')) - strftime('%Y', date(dob)) AS age, lat, lng 
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
		// TODO: fix why it wont scan in u.Location.Lat and long
		if err := rows.Scan(&u.ID, &u.Name, &u.Gender, &u.DOB, &age, &u.Location.Lat, &u.Location.Long); err != nil {
			return nil, err
		}

		age, err := user.CalculateAge(u.DOB, now)
		if err != nil {
			return userProfiles, err
		}
		distanceFromMe := 0.0
		if u.Location.Lat.Valid && u.Location.Long.Valid && userLocation.Lat.Valid && userLocation.Long.Valid {
			distanceFromMe = haversineDistance(u.Location.Lat.Float64, u.Location.Long.Float64, userLocation.Lat.Float64, userLocation.Long.Float64)
		}
		userProfile := profile{ID: u.ID, Name: u.Name, Gender: u.Gender, Age: age, DistanceFromMe: distanceFromMe}
		userProfiles = append(userProfiles, userProfile)
	}

	sort.Slice(userProfiles, func(i, j int) bool {
		return userProfiles[i].DistanceFromMe < userProfiles[j].DistanceFromMe
	})

	return userProfiles, nil
}

const (
	earthRadius = 6371 // Earth radius in kilometers
)

// haversineDistance calculates the distance between two points using the Haversine formula
func haversineDistance(lat1, lon1, lat2, lon2 float64) float64 {
	// Convert latitude and longitude from degrees to radians
	lat1Rad := degreesToRadians(lat1)
	lon1Rad := degreesToRadians(lon1)
	lat2Rad := degreesToRadians(lat2)
	lon2Rad := degreesToRadians(lon2)

	// Calculate differences
	deltaLat := lat2Rad - lat1Rad
	deltaLon := lon2Rad - lon1Rad

	// Calculate distance using Haversine formula
	a := math.Pow(math.Sin(deltaLat/2), 2) + math.Cos(lat1Rad)*math.Cos(lat2Rad)*math.Pow(math.Sin(deltaLon/2), 2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
	distance := earthRadius * c

	return distance
}

// degreesToRadians converts degrees to radians
func degreesToRadians(degrees float64) float64 {
	return degrees * (math.Pi / 180)
}
