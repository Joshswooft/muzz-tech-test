# Muzz tech test

## Pre-requisites

- Latest golang and docker installed.
- `Makefile` for executing commands.
- [`atlas`](https://atlasgo.io/getting-started/) for migrations

## Run application

Application will launch a http server available on `localhost:8080`.

```bash
go run main.go
```

## Build
```bash
make build
```

Then to run binary: `./bin/muzz`

## Test application

```bash
make test
```

### Some commands to test the API

Create a random user:
```bash
curl -X POST http://localhost:8080/user/create -H 'Authorization: Bearer <token>'
```

Note: The main app will create 1 dummy user for you which you can use for testing out the app.
Log user into the application:
```bash
curl -X POST \
  http://localhost:8080/login \
  -H 'Content-Type: application/json' \
  -d '{
	"email": "testuser@gmail.com",
	"password": "password"
}'
```


Retrieve potential matches
Requires authentication.
```bash
curl "http://localhost:8080/discover" -H 'Authorization: Bearer <token>' \
-H 'Content-Type: application/json'
```

Swipe on a potential match
Requires authentication.
```bash
curl -X POST \
  http://localhost:8080/swipe \
  -H 'Content-Type: application/json' \
  -H 'Authorization: Bearer your_token_here' \
  -d '{
    "other_user_id": 2,
    "like": true
}'
```