mod: go mod download

test:
	go test -tags=embed -race ./...

lint:
	go vet ./...

build:
	go build -o ./bin/muzz

# 'atlas schema apply' plans and executes a database migration to bring a given
# database to the state described in the provided Atlas schema. Before running the
# migration, Atlas will print the migration plan and prompt the user for approval.
migrate:
	atlas schema apply --url="sqlite://muzz.db" --dev-url="sqlite://file?mode=memory" --to="file://store/schema.sql"