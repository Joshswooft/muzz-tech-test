# Muzz tech test

## Pre-requisites

- Latest golang and docker installed.
- `Makefile` for executing commands.
- [`atlas`](https://atlasgo.io/getting-started/) for migrations
- gcc compiler (required for compiling the go-sqlite3 library)

## Running the application from docker

### Build the docker image
```bash
docker build -t muzz-app .
```

### Run the docker application

```bash
docker run -p 8080:8080 muzz-app
```

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

## Some commands to test the API

The tech tests 'Ensure that all other endpoints are appropriately authenticated' so I've made everything but `/login` authenticated.

Note: The main app will create 1 dummy user for you which you can use for testing out the app.

### Create a random user

Create a random user:
```bash
curl -X POST http://localhost:8080/user/create -H 'Authorization: Bearer <token>'
```

### Login user

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
### Discover potential matches

Retrieve potential matches based on distance from the user and the other profile's attractiveness.

The attractiveness is based on both distance and number of likes a profile has received. This is a weighted normalized score
so it doesn't heavily favour one stat over another. For this case I've decided to place a heavier weight onto distance.

This means profiles that are close by will be recommended higher but some amount of likes will influence the final results.

Requires authentication.
```bash
curl "http://localhost:8080/discover" -H 'Authorization: Bearer <token>' \
-H 'Content-Type: application/json'
```

#### Filters

The discover endpoint comes with 2 filters which can be used seperately or together. These are `age` and `gender`.
The spec didn't say how the age had to be filtered so I assumed an exact match for simplicity sake.

```bash
curl "http://localhost:8080/discover?age=30&gender=other" -H 'Authorization: Bearer <token>' \
-H 'Content-Type: application/json'
```


### Swipe on users

Swipe on a profile for a potential match. If the other user has also 'swiped' on you then it's considered a match. 

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