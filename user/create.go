package user

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"muzz/httpresponse"
	"net/http"
	"time"

	"github.com/go-faker/faker/v4"
	"golang.org/x/crypto/bcrypt"
)

// User represents the data stored in the user table
type User struct {
	ID       int
	Email    string
	Password string
	Name     string
	Gender   string
	DOB      string
	Location GeoLocation
}

type GeoLocation struct {
	Lat  sql.NullFloat64
	Long sql.NullFloat64
}

// createUserResult is the data returned to the client upon successful creation of the user
type createUserResult struct {
	ID       int    `json:"id"`
	Email    string `json:"email"`
	Password string `json:"password"`
	Name     string `json:"name"`
	Gender   string `json:"gender"`
	Age      int    `json:"age"`
}

// CreateUserResponse is the data returned to the client when creating a new user
type CreateUserResponse struct {
	Result createUserResult `json:"result"`
}

type CreateUserHandlerDeps struct {
	// db to save the user to
	DB *sql.DB

	// for managing the time yourself - in most cases you wont need to use this
	// mainly for testing
	Clock func() time.Time
}

// now is a time generator that falls back to std lib if clock is not specified
func (c *CreateUserHandlerDeps) now() time.Time {
	if c.Clock == nil {
		return time.Now()
	}
	return c.Clock()
}

// Creates a random user
func CreateUserHandler(deps CreateUserHandlerDeps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		randomUser := generateRandomUser()
		err := StoreUser(deps.DB, randomUser)
		if err != nil {
			slog.Error("failed to create user", slog.Any("error", err))
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(httpresponse.ErrorResponse{Error: "failed to create user"})
			return
		}

		createUserResult, err := convertToCreateUserResult(randomUser, deps.now())

		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(httpresponse.ErrorResponse{Error: "failed to create user"})
		}

		response := CreateUserResponse{Result: createUserResult}

		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(response)
	}
}

func convertToCreateUserResult(user User, now time.Time) (createUserResult, error) {
	age, err := CalculateAge(user.DOB, now)

	if err != nil {
		return createUserResult{}, err
	}

	return createUserResult{
		ID:       user.ID,
		Email:    user.Email,
		Password: user.Password,
		Name:     user.Name,
		Gender:   user.Gender,
		Age:      age,
	}, nil
}

// calculates age from a dob
func CalculateAge(dob string, now time.Time) (int, error) {
	layout := "2006-01-02"
	t, err := time.Parse(layout, dob)
	if err != nil {
		return 0, err
	}
	age := now.Sub(t).Hours() / 24 / 365
	return int(age), nil
}

func generateRandomUser() User {
	user := User{
		Email:    faker.Email(),
		Password: faker.Password(),
		Name:     faker.Name(),
		Gender:   faker.Gender(),
		DOB:      faker.Date(),
		Location: GeoLocation{
			Lat:  sql.NullFloat64{Float64: faker.Latitude(), Valid: true},
			Long: sql.NullFloat64{Float64: faker.Longitude(), Valid: true},
		},
	}

	return user
}

// Stores a user in the sqlite db
func StoreUser(db *sql.DB, user User) error {

	if db == nil {
		return fmt.Errorf("no database provided")
	}

	// bcrypt salts the password for us
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to save user, password couldnt be salted %w", err)
	}

	_, err = db.Exec("INSERT INTO users (email, password, name, gender, dob, lat, lng) VALUES (?, ?, ?, ?, ?, ?, ?)", user.Email, hashedPassword, user.Name, user.Gender, user.DOB, user.Location.Lat, user.Location.Long)

	if err != nil {
		return err
	}

	return nil
}
