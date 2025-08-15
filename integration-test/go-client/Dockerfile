FROM golang:1.24.5-alpine3.22
WORKDIR /app
COPY . .
RUN ls -la /app && ls -la /app/client || true
RUN go build -mod=vendor -o client
EXPOSE 8080
CMD ["./client"]
