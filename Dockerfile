FROM golang:1.21-alpine

WORKDIR /app

COPY go.mod .

RUN go mod download

COPY .env .

COPY main.go .

RUN go build -o /urlshortener

EXPOSE 8080

CMD [ "/urlshortener" ]
