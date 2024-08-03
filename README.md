# URL Shortener

A URL shortener application built with Go.

## Prerequisites

- Go 1.21 or later
- Docker (optional, for running with Docker)

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
   Edit the `.env` file if you want to customize any settings. The `SAFE_BROWSING_API_KEY` and `SESSION_SECRET_KEY` are optional. If not provided, the safe browsing feature will be disabled and the session secret key will be a random string.

3. Install dependencies:
   ```
   go mod tidy
   ```

4. Run the application:
   ```
   go run main.go
   ```

5. Open a web browser and navigate to `http://localhost:8080` (or the port you specified in the `.env` file).

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
   Edit the `.env` file if you want to customize any settings. The `SAFE_BROWSING_API_KEY` and `SESSION_SECRET_KEY` are optional. If not provided, the safe browsing feature will be disabled and the session secret key will be a random string.

3. Build the Docker image:
   ```
   docker build -t url-shortener .
   ```

4. Run the Docker container:
   ```
   docker run -p 8080:8080 url-shortener
   ```

5. Open a web browser and navigate to `http://localhost:8080`.

## Configuration

The application uses environment variables for configuration. You can set these in the `.env` file:

- `PORT`: The port on which the server will run (default: 8080)
- `DB_PATH`: Path to the SQLite database file (default: database/database.sqlite3)
- `SAFE_BROWSING_API_KEY`: Google Safe Browsing API key (optional, default: disabled)
- `SESSION_SECRET_KEY`: Secret key for encrypting session data (optional, default: a random string)
