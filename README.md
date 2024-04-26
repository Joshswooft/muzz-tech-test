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
curl -X POST "http://localhost:8080/user/create"
```

Note: The main app will create 1 dummy user for you
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