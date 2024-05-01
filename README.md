# Muzz tech test

Some notes about the technical test.

- I've tried to keep things as simple as possible (KISS)
- I've assumed some common security practices for storing passwords
- I've assumed token meant a JWT token and I just chose a signing alg that I wanted to use
- No extra abstractions
- The database chosen here is sqlite - Purely for fun! I know it's not the most advanced nor practical database for running in production particularly for this type of problem
- No ORMs and general lack of 3rd party library usage. I've done this mainly so you can see my experience at using golang but also my knowledge with SQL. In the real world there is alot of choice between ORM's and type safe query builders.
- Testing has been [done without mocks](https://aran.dev/posts/you-probably-dont-need-to-mock/)! Personally I'm not a fan of mocks, it's extra code to maintain and forces you to make assumptions on the usage and how the underlying code works. With databases this can be particularly tricky. Luckily sqlite is just a file/inmemory
so its very easy to test against. Similarily I've used the `httptest` library which creates an in memory server to test against.
- Lastly using the newest net/http router, this is my first time playing with it. I'm not sure if it covers all the usecases to fully replace something like gin, fiber, echo etc.

Thanks for reading, and looking forward to your feedback!

## Pre-requisites

- Latest golang and docker installed.
- `Makefile` for executing commands.
- [`atlas`](https://atlasgo.io/getting-started/) for migrations (dev usage only)
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