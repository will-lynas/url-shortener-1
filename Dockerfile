FROM golang:1.20-alpine

WORKDIR /app

COPY . .

RUN go mod tidy

EXPOSE 8080

CMD ["go", "run", "cmd/main.go"]