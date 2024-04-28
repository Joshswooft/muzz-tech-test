package main

import (
	"database/sql"
	"fmt"
	"log"
	"log/slog"
	"muzz/login"
	"muzz/store"
	"muzz/user"
	"net/http"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

type WrappedWriter struct {
	http.ResponseWriter
	statusCode int
}

// Implement the http.ResponseWriter interface
func (w *WrappedWriter) WriteHeader(statusCode int) {
	w.ResponseWriter.WriteHeader(statusCode)
	w.statusCode = statusCode
}

func logger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		wrapped := &WrappedWriter{statusCode: http.StatusOK, ResponseWriter: w}

		next.ServeHTTP(wrapped, r)
		slog.Info("request", slog.Int("code", wrapped.statusCode), slog.String("method", r.Method), slog.String("path", r.URL.Path), slog.Int64("durationMS", time.Since(start).Milliseconds()))
	})
}

func main() {

	db, err := sql.Open("sqlite3", "./muzz.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	if _, err := db.Exec(store.SchemaSQL); err != nil {
		log.Fatal(err)
	}

	testUser := user.User{Name: "TestUser", Email: "testuser@gmail.com", Password: "password"}
	if err := user.StoreUser(db, testUser); err != nil {
		slog.Error("failed to create test user for application", slog.Any("error", err))
	}

	router := http.NewServeMux()

	// Define endpoints
	router.HandleFunc("POST /user/create", user.CreateUserHandler(db))
	router.HandleFunc("POST /login", login.LoginHandler(db))

	server := http.Server{
		Addr:         ":8080",
		Handler:      logger(router),
		WriteTimeout: time.Second * 10,
		ReadTimeout:  time.Second * 10,
	}

	fmt.Println("Server is listening on port 8080...")
	server.ListenAndServe()
}
