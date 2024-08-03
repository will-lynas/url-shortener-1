# URL Shortener

A simple URL shortener application built with Go.

## Prerequisites

- Go 1.21 or later
- Docker (optional, for running with Docker)

## Running from source

1. Clone the repository:
   ```
   git clone https://github.com/artemstreltsov/url-shortener.git
   cd url-shortener
   ```

2. Create a `.env` file in the root of the project:
   ```
   cp example.env .env
   ```
   Edit the `.env` file if you want to customize any settings. The `SAFE_BROWSING_API_KEY` is optional. If not provided, the safe browsing feature will be disabled.

3. Install dependencies:
   ```
   go mod tidy
   ```

4. Run the application:
   ```
   go run cmd/main.go
   ```

5. Open a web browser and navigate to `http://localhost:8080` (or the port you specified in the .env file).

## Running with Docker

1. Clone the repository:
   ```
   git clone https://github.com/artemstreltsov/url-shortener.git
   cd url-shortener
   ```

2. Create a `.env` file in the root of the project:
   ```
   cp example.env .env
   ```
   Edit the `.env` file if you want to customize any settings. The `SAFE_BROWSING_API_KEY` is optional. If not provided, the safe browsing feature will be disabled.

3. Build the Docker image:
   ```
   docker build -t url-shortener .
   ```

4. Run the Docker container:
   ```
   docker run -p 8080:8080 --env-file .env -v $(pwd):/app url-shortener
   ```

5. Open a web browser and navigate to `http://localhost:8080`.

## Configuration

The application uses environment variables for configuration. You can set these in the `.env` file or pass them directly when running the application. Here are the available options:

- `PORT`: The port on which the server will run (default: 8080)
- `DB_PATH`: Path to the SQLite database file (default: database/database.sqlite3)
- `SAFE_BROWSING_API_KEY`: Google Safe Browsing API key (optional)
- `SESSION_SECRET_KEY`: Secret key for encrypting session data (default: a random string)

## Features

- Shorten long URLs
- User registration and authentication
- Dashboard to manage shortened URLs
- Optional Safe Browsing check for malicious URLs (requires Google Safe Browsing API key)

## Troubleshooting

If you encounter any issues while running the application, please check the following:

1. Make sure you have Go 1.21 or later installed on your system.
2. Ensure that all required dependencies are installed by running `go mod tidy`.
3. Check if the `.env` file is present in the root directory of the project and contains the necessary configuration.
4. If using Docker, make sure Docker is installed and running on your system.
5. If you're still experiencing issues, try removing the `database` directory (if it exists) and restarting the application to create a fresh database.

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.