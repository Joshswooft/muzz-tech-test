package main

import (
	"database/sql"
	"fmt"
	"log"
	"log/slog"
	"muzz/auth"
	"muzz/matchmaker"
	"muzz/middleware"
	"muzz/store"
	"muzz/user"
	"net/http"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

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

	tokenAuth := auth.NewTokenAuthenticator()
	authGuardMiddleware := middleware.NewAuthGuardMiddleware(tokenAuth.ExtractClaimsFromToken)

	router := http.NewServeMux()

	// Define auth endpoints
	authRouter := http.NewServeMux()
	authRouter.HandleFunc("POST /user/create", user.CreateUserHandler(user.CreateUserHandlerDeps{DB: db}))
	authRouter.HandleFunc("GET /discover", matchmaker.DiscoverHandler(matchmaker.DiscoverHandlerDeps{DB: db}))
	authRouter.HandleFunc("POST /swipe", matchmaker.SwipeHandler(matchmaker.SwipeHandlerDeps{DB: db}))
	router.Handle("/", authGuardMiddleware(authRouter))

	// Define un-authenticated endpoints
	router.HandleFunc("POST /login", auth.LoginHandler(auth.LoginHandlerDeps{DB: db, JwtTokenGenerator: tokenAuth.GenerateJWTToken}))

	server := http.Server{
		Addr:         ":8080",
		Handler:      middleware.Logger(router),
		WriteTimeout: time.Second * 10,
		ReadTimeout:  time.Second * 10,
	}

	fmt.Println("Server is listening on port 8080...")
	server.ListenAndServe()
}
