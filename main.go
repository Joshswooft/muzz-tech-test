package main

import (
	"crypto/rand"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-faker/faker/v4"
	_ "github.com/mattn/go-sqlite3"
	"golang.org/x/crypto/bcrypt"
)

// Schema for creating SQLite table
const schema = `
CREATE TABLE IF NOT EXISTS users (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	email TEXT,
	password TEXT,
	name TEXT,
	gender TEXT,
	dob TEXT
);

`

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

type CreateUserResponse struct {
	Result UserResponse `json:"result"`
}

type handler struct {
	db *sql.DB
}

func (h *handler) createUserHandler(w http.ResponseWriter, r *http.Request) {
	randomUser := generateRandomUser()

	err := storeUser(h.db, randomUser)
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

func storeUser(db *sql.DB, user User) error {

	salt := make([]byte, 16)
	_, err := rand.Read(salt)
	if err != nil {
		return fmt.Errorf("failed to salt password: %w", err)
	}

	pwd := append([]byte(user.Password), salt...)
	if len(pwd) > 72 {
		return fmt.Errorf("password cant be longer than 72 bytes")
	}

	hashedPassword, err := bcrypt.GenerateFromPassword(pwd, bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to save user, password couldnt be salted %w", err)
	}

	_, err = db.Exec("INSERT INTO users (email, password, name, gender, dob) VALUES (?, ?, ?, ?, ?)", user.Email, hashedPassword, user.Name, user.Gender, user.DOB)

	if err != nil {
		return err
	}

	return nil
}

func main() {

	db, err := sql.Open("sqlite3", "./muzz.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	if _, err := db.Exec(schema); err != nil {
		log.Fatal(err)
	}

	userHandler := handler{db: db}

	http.HandleFunc("/user/create", userHandler.createUserHandler)
	fmt.Println("Server is listening on port 8080...")
	http.ListenAndServe(":8080", nil)
}
