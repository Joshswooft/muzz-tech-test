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
	ID     int    `json:"id"`
	Name   string `json:"name"`
	Gender string `json:"gender"`
	// Age in years
	Age int `json:"age"`
	// distance from me in km
	DistanceFromMe float64 `json:"distanceFromMe"`
	// totalLikes received from other users swiping on them
	totalLikes int

	// users attractiveness is a weighted score based on distance from a user and their total likes
	attractivenessScore float64
}

// score is based on how close the profile is to the user logged in and how many total likes they have
func (p *profile) calculateAttractivenessScore(normalizedDistance float64, normalizedTotalLikes float64) {
	score := ((1 - normalizedDistance) * 0.8) + (normalizedTotalLikes * 0.2)
	p.attractivenessScore = score
}

// The JSON response for the discover handler
type DiscoverResponse struct {
	Results []*profile `json:"results"`
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
	var lat, lng float64
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
func getPotentialMatches(db *sql.DB, userID int, now time.Time, filters filters, userLocation user.GeoLocation) (userProfiles []*profile, err error) {

	userProfiles = []*profile{}

	// assumes we only care about the year of the date of birth for simplicity
	query := `
	SELECT 
	u.id, u.name, u.gender, u.dob, strftime('%Y', date('now')) - strftime('%Y', date(u.dob)) AS age, u.lat, u.lng, COUNT(s.id) AS like_count
	FROM users u
	LEFT JOIN swipes s ON u.id = s.swipe_target AND s.liked = 1
	WHERE u.id NOT IN (SELECT swipe_target FROM swipes WHERE swiper = ?) AND u.id != ?
	`

	params := []interface{}{userID, userID}

	if filters.age != 0 {
		query += " AND age = ?"
		params = append(params, filters.age)
	}

	if filters.gender != "" {
		query += " AND gender = ?"
		params = append(params, filters.gender)
	}

	query += "GROUP BY u.id, u.name, u.gender, u.dob, u.lat, u.lng"

	rows, err := db.Query(query, params...)
	if err != nil {
		return
	}
	defer rows.Close()

	var minDistanceFromMe, maxDistanceFromMe float64
	var minTotalLikes, maxTotalLikes int

	for rows.Next() {
		var u user.User
		var age, totalLikes int
		if err := rows.Scan(&u.ID, &u.Name, &u.Gender, &u.DOB, &age, &u.Location.Lat, &u.Location.Long, &totalLikes); err != nil {
			return userProfiles, err
		}

		age, err := user.CalculateAge(u.DOB, now)
		if err != nil {
			return userProfiles, err
		}

		distanceFromMe := haversineDistance(u.Location.Lat, u.Location.Long, userLocation.Lat, userLocation.Long)
		minDistanceFromMe = math.Min(distanceFromMe, minDistanceFromMe)
		maxDistanceFromMe = math.Max(distanceFromMe, maxDistanceFromMe)

		if totalLikes < minTotalLikes {
			minTotalLikes = totalLikes
		}

		if totalLikes > maxTotalLikes {
			maxTotalLikes = totalLikes
		}

		userProfile := &profile{ID: u.ID, Name: u.Name, Gender: u.Gender, Age: age, DistanceFromMe: distanceFromMe, totalLikes: totalLikes}
		userProfiles = append(userProfiles, userProfile)
	}

	for _, profile := range userProfiles {
		normalizedDistance := normalizeScore(profile.DistanceFromMe, minDistanceFromMe, maxDistanceFromMe)
		normalizedTotalLikes := normalizeScore(float64(profile.totalLikes), float64(minTotalLikes), float64(maxTotalLikes))
		profile.calculateAttractivenessScore(normalizedDistance, normalizedTotalLikes)
	}

	// sorts user profiles by their 'attractiveness' DESC
	sort.Slice(userProfiles, func(i, j int) bool {
		return userProfiles[i].attractivenessScore > userProfiles[j].attractivenessScore
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

// normalizeScore normalizes a raw score to fall within a specified range.
// minRawScore and maxRawScore define the range of the raw scores.
func normalizeScore(rawScore, minRawScore, maxRawScore float64) float64 {
	// Ensure that minRawScore is not greater than maxRawScore to avoid division by zero
	if minRawScore >= maxRawScore {
		return 0.0
	}

	// Normalize the raw score to the range [0, 1]
	normalizedScore := (rawScore - minRawScore) / (maxRawScore - minRawScore)

	// Clamp the normalized score to the range [0, 1]
	if normalizedScore < 0.0 {
		return 0.0
	}
	if normalizedScore > 1.0 {
		return 1.0
	}

	return normalizedScore
}
