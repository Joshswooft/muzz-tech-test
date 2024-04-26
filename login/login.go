package login

import (
	"database/sql"
	"encoding/json"
	"log/slog"
	"muzz/user"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt"
	"golang.org/x/crypto/bcrypt"
)

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error string `json:"error"`
}

// JWTClaims represents JWT claims
type JWTClaims struct {
	UserID int `json:"user_id"`
	jwt.StandardClaims
}

// logs the user into the application and returns a JWT token to the user
func LoginHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		var user user.User
		if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(ErrorResponse{Error: "Invalid request payload"})
			return
		}

		row := db.QueryRow("SELECT id, password FROM users WHERE email = ?", user.Email)
		var storedID int
		var storedPassword string
		err := row.Scan(&storedID, &storedPassword)
		if err != nil {
			slog.Error("error scanning row", slog.Any("error", err))
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(ErrorResponse{Error: "Invalid email or password"})
			return
		}

		if err := bcrypt.CompareHashAndPassword([]byte(storedPassword), []byte(user.Password)); err != nil {
			slog.Error("Invalid email or password", slog.Any("error", err))
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(ErrorResponse{Error: "Invalid email or password"})
			return
		}

		token, err := generateJWTToken(storedID)
		if err != nil {
			slog.Error("error generating JWT", slog.Any("error", err))
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(ErrorResponse{Error: "Failed to generate token"})
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"token": token})
	}
}

// generateJWTToken generates a JWT token with the given user ID
func generateJWTToken(userID int) (string, error) {
	claims := JWTClaims{
		UserID: userID,
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: time.Now().Add(time.Hour * 24).Unix(), // Token expires in 24 hours
			IssuedAt:  time.Now().Unix(),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte("secret"))
}
