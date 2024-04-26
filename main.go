package main

import (
	"database/sql"
	"fmt"
	"log"
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

	http.HandleFunc("/user/create", user.CreateUserHandler(db))
	fmt.Println("Server is listening on port 8080...")
	http.ListenAndServe(":8080", nil)
}
