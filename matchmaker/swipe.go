package matchmaker

import (
	"database/sql"
	"encoding/json"
	"errors"
	"muzz/httpresponse"
	"muzz/middleware"
	"net/http"

	_ "github.com/mattn/go-sqlite3"
)

// Request body for the SwipeHandler
type SwipeRequest struct {
	OtherUserID int `json:"other_user_id"`
	// did the user want to match with other user?
	Like bool `json:"like"`
}

type swipeResult struct {
	Matched bool `json:"matched"`
	MatchID int  `json:"matchID,omitempty"`
}

// The response from the swipe handler
type SwipeResponse struct {
	Results swipeResult `json:"results"`
}

// Represents the Match entity for the sqlite db
type Match struct {
	ID    int
	User1 int
	User2 int
}

type SwipeHandlerDeps struct {
	DB *sql.DB
}

// allows the sender to potentially match with other users on the platform
// The other user needs to have also 'matched' against the sender to consider it a match
// Returns whether the user has matched with the person they are swiping on and the `matchID`
func SwipeHandler(deps SwipeHandlerDeps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		w.Header().Set("Content-Type", "application/json")

		var req SwipeRequest
		err := json.NewDecoder(r.Body).Decode(&req)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(httpresponse.ErrorResponse{Error: "Invalid request payload"})
			return
		}

		claims, found := middleware.GetClaimsFromContext(r.Context())

		if !found {
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(httpresponse.ErrorResponse{Error: "Unauthenticated"})
			return
		}

		tx, err := deps.DB.Begin()
		if err != nil {
			http.Error(w, "Failed to start transaction", http.StatusInternalServerError)
			return
		}
		defer tx.Rollback()

		userLiked := req.Like
		myUserID := claims.UserID

		if createSwipeErr := createSwipeRecordInTransaction(tx, myUserID, req.OtherUserID, userLiked); createSwipeErr != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(httpresponse.ErrorResponse{Error: "Failed to process swipe request"})
			return
		}

		err = tx.Commit()
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(httpresponse.ErrorResponse{Error: "Failed to process swipe request"})
			return
		}

		existingMatch, foundMatchErr := getExistingMatchForUser(deps.DB, myUserID, req.OtherUserID)

		// no existing match found
		if errors.Is(foundMatchErr, errNoExistingMatchFound) || existingMatch == nil {
			resp := SwipeResponse{Results: swipeResult{Matched: false}}
			json.NewEncoder(w).Encode(resp)
			return
		}

		if foundMatchErr != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(httpresponse.ErrorResponse{Error: "Failed to process swipe request"})
			return
		}

		resp := SwipeResponse{Results: swipeResult{Matched: true, MatchID: existingMatch.ID}}
		json.NewEncoder(w).Encode(resp)
	}
}

var errNoExistingMatchFound = errors.New("no existing match found")

func getExistingMatchForUser(db *sql.DB, userID1 int, userID2 int) (*Match, error) {
	var match Match

	u1 := userID1
	u2 := userID2

	if userID2 < userID1 {
		u1 = userID2
		u2 = userID1
	}
	err := db.QueryRow("SELECT id, user1, user2 FROM matches WHERE user1 = ? AND user2 = ?", u1, u2).
		Scan(&match.ID, &match.User1, &match.User2)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, errNoExistingMatchFound
		}
		return nil, err
	}

	return &match, nil
}

// When we create a new swipe record it will call a SQL trigger which might create a match record if both users have liked each other
func createSwipeRecordInTransaction(tx *sql.Tx, swiper int, swipe_target int, liked bool) error {
	_, err := tx.Exec("INSERT INTO swipes (swiper, swipe_target, liked) VALUES (?, ?, ?)",
		swiper, swipe_target, liked)

	if err != nil {
		return err
	}

	return nil
}
