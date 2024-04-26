package user

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-faker/faker/v4"
	"golang.org/x/crypto/bcrypt"
)

// User represents the data stored in the user table
type User struct {
	ID       int    `json:"id"`
	Email    string `json:"email"`
	Password string `json:"password"`
	Name     string `json:"name"`
	Gender   string `json:"gender"`
	DOB      string `json:"date_of_birth"`
}

// UserResponse is the data returned to the client
type UserResponse struct {
	ID       int    `json:"id"`
	Email    string `json:"email"`
	Password string `json:"password"`
	Name     string `json:"name"`
	Gender   string `json:"gender"`
	Age      int    `json:"age"`
}

// CreateUserResponse is the data returned to the client when creating a new user
type CreateUserResponse struct {
	Result UserResponse `json:"result"`
}

// Creates a random user
func CreateUserHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		randomUser := generateRandomUser()
		err := StoreUser(db, randomUser)
		if err != nil {
			slog.Error("failed to create user", slog.Any("error", err))
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		userResponse, err := convertToUserResponse(randomUser)

		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}

		response := CreateUserResponse{Result: userResponse}

		jsonResponse, err := json.Marshal(response)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(jsonResponse)
	}
}

func convertToUserResponse(user User) (UserResponse, error) {
	age, err := calculateAge(user.DOB)

	if err != nil {
		return UserResponse{}, err
	}

	return UserResponse{
		ID:       user.ID,
		Email:    user.Email,
		Password: user.Password,
		Name:     user.Name,
		Gender:   user.Gender,
		Age:      age,
	}, nil
}

// calculates age from a dob
func calculateAge(dob string) (int, error) {
	layout := "2006-01-02"
	t, err := time.Parse(layout, dob)
	if err != nil {
		return 0, err
	}
	age := time.Since(t).Hours() / 24 / 365
	return int(age), nil
}

func generateRandomUser() User {
	user := User{
		Email:    faker.Email(),
		Password: faker.Password(),
		Name:     faker.Name(),
		Gender:   faker.Gender(),
		DOB:      faker.Date(),
	}

	return user
}

// Stores a user in the sqlite db
func StoreUser(db *sql.DB, user User) error {

	// bcrypt salts the password for us
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to save user, password couldnt be salted %w", err)
	}

	_, err = db.Exec("INSERT INTO users (email, password, name, gender, dob) VALUES (?, ?, ?, ?, ?)", user.Email, hashedPassword, user.Name, user.Gender, user.DOB)

	if err != nil {
		return err
	}

	return nil
}
