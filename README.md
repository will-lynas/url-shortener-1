# URL Shortener

A URL shortener built with Go.

## Running from source

1. Clone the repository:
   ```
   git clone https://github.com/artem-streltsov/url-shortener.git
   cd url-shortener
   ```

2. Create a `.env` file in the root of the project:
   ```
   cp example.env .env
   ```
   Edit the `.env` file - all fields are required.

3. Install dependencies:
   ```
   go mod tidy
   ```

4. Run the application:
   ```
   go run main.go
   ```

## Running with Docker

1. Clone the repository:
   ```
   git clone https://github.com/artem-streltsov/url-shortener.git
   cd url-shortener
   ```

2. Create a `.env` file in the root of the project:
   ```
   cp example.env .env
   ```
   Edit the `.env` file - all fields are required.

3. Build the Docker image:
   ```
   docker build -t url-shortener .
   ```

4. Run the Docker container:
   ```
   docker run -p 8080:8080 url-shortener
   ```
