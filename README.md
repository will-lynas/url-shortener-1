## Running
1. Create a `.env` file in the root of the project, see `example.env`
2. `$ go mod tidy`
3. `$ go run cmd/main.go`
4. Navigate to `localhost:port`, where port is specified in `.env`.

## Running with Docker
1. Create a `.env` file in the root of the project, see `example.env`.
2. `$ docker build -t url-shortener .`
3. `$ docker run -p 8080:8080 url-shortener`