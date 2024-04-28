package auth

import (
	"database/sql"
	"encoding/json"
	"log/slog"
	"muzz/httpresponse"
	"muzz/user"
	"net/http"

	_ "github.com/mattn/go-sqlite3"
	"golang.org/x/crypto/bcrypt"
)

type LoginHandlerDeps struct {
	DB *sql.DB
	JwtTokenGenerator
}

// Generates a signed JWT token from a given user id
type JwtTokenGenerator func(userID int) (string, error)

// logs the user into the application and returns a JWT token to the user
func LoginHandler(deps LoginHandlerDeps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		var user user.User
		if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(httpresponse.ErrorResponse{Error: "Invalid request payload"})
			return
		}

		row := deps.DB.QueryRow("SELECT id, password FROM users WHERE email = ?", user.Email)
		var storedID int
		var storedPassword string
		err := row.Scan(&storedID, &storedPassword)
		if err != nil {
			slog.Error("error scanning row", slog.Any("error", err))
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(httpresponse.ErrorResponse{Error: "Invalid email or password"})
			return
		}

		if err := bcrypt.CompareHashAndPassword([]byte(storedPassword), []byte(user.Password)); err != nil {
			slog.Error("Invalid email or password", slog.Any("error", err))
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(httpresponse.ErrorResponse{Error: "Invalid email or password"})
			return
		}

		token, err := deps.JwtTokenGenerator(storedID)
		if err != nil {
			slog.Error("error generating JWT", slog.Any("error", err))
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(httpresponse.ErrorResponse{Error: "Failed to generate token"})
			return
		}

		json.NewEncoder(w).Encode(map[string]string{"token": token})
	}
}
