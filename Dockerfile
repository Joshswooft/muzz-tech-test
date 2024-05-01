FROM golang:1.22-alpine AS build

# required for the go-sqlite3 library
ENV CGO_ENABLED=1


RUN apk add --no-cache gcc g++ pkgconfig

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download
COPY . .

RUN go build -o app

FROM alpine:latest  

WORKDIR /root/

COPY --from=build /app/app .

EXPOSE 8080

CMD ["./app"]
