# Muzz tech test

## Pre-requisites

Latest golang and docker installed.

Using `Makefile` for executing commands.

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