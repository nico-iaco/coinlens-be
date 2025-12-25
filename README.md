# CoinLens Backend

CoinLens is a powerful backend application designed to catalog and identify coins. Leveraging Google's Gemini 3 Flash AI, it analyzes front and back images of coins to provide detailed identification data (name, country, year, description) and stores this information in a PostgreSQL database while managing image assets on the local filesystem.

## 🚀 Features

- **AI-Powered Identification**: Integrates with Google Gemini SDK to analyze coin images and extract metadata.
- **Image Management**: Handles multipart file uploads, storing images securely on the local filesystem.
- **Data Persistence**: robust PostgreSQL integration for storing coin metadata and file paths.
- **Containerization**: Fully Dockerized with multi-stage builds and Docker Compose support for easy deployment.
- **RESTful API**: Clean API design for frontend integration.

## 🛠️ Tech Stack

- **Language**: Go (Golang) 1.25+
- **Database**: PostgreSQL 15
- **AI Engine**: Google Gemini 3 Flash (`google.golang.org/genai`)
- **Router/HTTP**: Standard library `net/http`
- **Database Driver**: `pgx` (PostgreSQL Driver and Toolkit)
- **Configuration**: `godotenv` for environment variable management
- **Containerization**: Docker & Docker Compose

## 📂 Project Structure

```text
coinlens-be/
├── cmd/
│   └── api/
│       └── main.go           # Application entry point
├── internal/
│   ├── config/               # Configuration loader
│   ├── database/             # Database connection logic
│   ├── handler/              # HTTP Request handlers
│   ├── models/               # Domain data models
│   └── service/              # Core business logic (Gemini, Storage)
├── migrations/               # SQL migration files
├── uploads/                  # Directory for storing uploaded images
├── .env.example              # Example configuration file
├── Dockerfile                # Multi-stage Docker build definition
├── docker-compose.yml        # Docker composition for App + DB
└── go.mod                    # Go module definitions
```

## ⚙️ Setup & Installation

### Prerequisites

- **Go**: Version 1.25 or higher
- **Docker**: For containerized execution
- **Gemini API Key**: Obtainable from Google AI Studio

### Environment Configuration

1. Copy the example environment file:

    ```bash
    cp .env.example .env
    ```

2. Edit `.env` and fill in your details:

    ```env
    DATABASE_URL=postgres://user:password@localhost:5432/coinlens?sslmode=disable
    PORT=8080
    GEMINI_API_KEY=your_actual_api_key_here
    ```

### 📦 Running with Docker (Recommended)

The easiest way to run the application is using Docker Compose. It sets up both the backend and the PostgreSQL database.

1. **Build and Start**:

    ```bash
    docker-compose up --build
    ```

    The API will be available at `http://localhost:8080`.
    Review the `docker-compose.yml` to ensure volumes and ports match your needs.

### 🏃 Running Locally

1. **Start PostgreSQL**: Ensure you have a Postgres instance running and accessible.
2. **Run Migrations**: Execute the SQL scripts in `migrations/` against your database.
3. **Start the Server**:

    ```bash
    go run ./cmd/api
    ```

## 🔌 API Documentation

### Identify Coin

Analyzes uploaded images and returns identification details.

- **Endpoint**: `POST /api/coins/identify`
- **Content-Type**: `multipart/form-data`

#### Parameters

| Key | Type | Description |
| :--- | :--- | :--- |
| `front_image` | File | Image of the coin's front (obverse) |
| `back_image` | File | Image of the coin's back (reverse) |

#### Response Example

```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "name": "Lincoln Cent",
  "description": "The Lincoln cent/penny is a one-cent coin that has been struck by the United States Mint since 1909.",
  "year": "1944",
  "country": "United States"
  "country": "United States"
}
```

### List Coins

Retrieves the catalog of identified coins with image URLs.

- **Endpoint**: `GET /api/coins`
- **Response Example**

```json
[
  {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "name": "Lincoln Cent",
    "description": "...",
    "year": "1944",
    "country": "United States",
    "image_front_url": "/uploads/front.jpg",
    "image_back_url": "/uploads/back.jpg",
    "created_at": "2025-12-25T19:30:00Z"
  }
]
```

## 📝 License

This project is open-source and available for personal and educational use.
