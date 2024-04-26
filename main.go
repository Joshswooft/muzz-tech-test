package main

import (
	"database/sql"
	"fmt"
	"log"
	"log/slog"
	"muzz/store"
	"muzz/user"
	"net/http"

	_ "github.com/mattn/go-sqlite3"
)

func main() {

	db, err := sql.Open("sqlite3", "./muzz.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	if _, err := db.Exec(store.Schema); err != nil {
		log.Fatal(err)
	}

	testUser := user.User{Name: "TestUser", Email: "testuser@gmail.com", Password: "password"}
	if err := user.StoreUser(db, testUser); err != nil {
		slog.Error("failed to create test user for application", slog.Any("error", err))
	}

	http.HandleFunc("/user/create", user.CreateUserHandler(db))
	fmt.Println("Server is listening on port 8080...")
	http.ListenAndServe(":8080", nil)
}
