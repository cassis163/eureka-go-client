FROM golang:1.24.5-alpine3.22
WORKDIR /app
COPY . .
RUN go build -mod=vendor -o client ./cmd/client.go
EXPOSE 8080
CMD ["./client"]
