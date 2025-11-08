# Generic SaaS Platform

A modern SaaS application template built with Go backend and Astro.js frontend with Solid.js components.

## Architecture

- **Backend**: Go with standard library HTTP server
- **Frontend**: Astro.js with Solid.js components and Tailwind CSS
- **Database**: PostgreSQL with in-memory fallback for development
- **Authentication**: Custom JWT-based authentication
- **Containerization**: Docker for PostgreSQL database

## Prerequisites

- Go 1.25.2+
- Node.js 18+
- Docker and Docker Compose
- PostgreSQL (via Docker)

## Quick Start

### 1. Start the Database (PostgreSQL via Docker)

```bash
# Start PostgreSQL container
docker run -d \
  --name generic-saas-postgres \
  -e POSTGRES_DB=generic_saas \
  -e POSTGRES_USER=saas_user \
  -e POSTGRES_PASSWORD=saas_password \
  -p 5432:5432 \
  postgres:15

# Verify container is running
docker ps | grep generic-saas-postgres
```

### 2. Start the Backend Server

```bash
# Navigate to backend directory
cd backend

# Install dependencies
go mod tidy

# Option 1: Run with PostgreSQL (recommended)
export DATABASE_URL="postgres://saas_user:saas_password@localhost:5432/generic_saas?sslmode=disable"
go run cmd/server/main.go

# Option 2: Run with in-memory database (for testing)
go run cmd/server/main.go
```

The backend server will:
- Start on `http://localhost:8080`
- Automatically run database migrations on startup
- Use PostgreSQL if `DATABASE_URL` is set, otherwise fall back to in-memory storage

### 3. Start the Frontend

```bash
# Navigate to frontend directory
cd frontend

# Install dependencies
npm install

# Start development server
npm run dev
```

The frontend will be available at `http://localhost:4321`

## Environment Variables

### Backend Configuration

| Variable | Description | Default |
|----------|-------------|---------|
| `PORT` | Backend server port | `8080` |
| `DATABASE_URL` | PostgreSQL connection string | None (uses in-memory) |

### Example Environment Setup

```bash
# For production-like setup with PostgreSQL
export DATABASE_URL="postgres://saas_user:saas_password@localhost:5432/generic_saas?sslmode=disable"
export PORT="8080"

# For development with in-memory database
unset DATABASE_URL
export PORT="8080"
```

## Database Management

### Migrations

Database migrations run automatically when the server starts. The migration system:

- Creates a `schema_migrations` table to track applied migrations
- Runs migrations in transaction-safe blocks
- Supports both up and down migrations
- Currently includes user table creation

### Manual Migration Management

```bash
# Connect to database directly
docker exec -it generic-saas-postgres psql -U saas_user -d generic_saas

# View current schema
\dt

# View migration status
SELECT * FROM schema_migrations;
```

### Reset Database

```bash
# Stop and remove existing container
docker stop generic-saas-postgres
docker rm generic-saas-postgres

# Start fresh container
docker run -d \
  --name generic-saas-postgres \
  -e POSTGRES_DB=generic_saas \
  -e POSTGRES_USER=saas_user \
  -e POSTGRES_PASSWORD=saas_password \
  -p 5432:5432 \
  postgres:15
```

## API Endpoints

### Health Checks

- `GET /` - API information and status
- `GET /health` - Health check endpoint

### Authentication

- `POST /auth/register` - User registration
- `POST /auth/login` - User login

### Request/Response Examples

#### User Registration
```bash
curl -X POST http://localhost:8080/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "name": "John Doe",
    "email": "john@example.com",
    "password": "securepassword123"
  }'
```

#### User Login
```bash
curl -X POST http://localhost:8080/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "john@example.com",
    "password": "securepassword123"
  }'
```

## Development

### Running Tests

```bash
# Backend tests
cd backend
go test ./...

# Test with PostgreSQL specifically
go test ./internal/database -v

# Test with memory database
go test ./internal/database -run TestMemory -v
```

### Building for Production

```bash
# Build backend binary
cd backend
go build -o bin/server cmd/server/main.go

# Build frontend
cd frontend
npm run build
```

### Development Workflow

1. **Start services in order:**
   ```bash
   # Terminal 1: Start PostgreSQL
   docker run -d --name generic-saas-postgres -e POSTGRES_DB=generic_saas -e POSTGRES_USER=saas_user -e POSTGRES_PASSWORD=saas_password -p 5432:5432 postgres:15

   # Terminal 2: Start backend
   cd backend && DATABASE_URL="postgres://saas_user:saas_password@localhost:5432/generic_saas?sslmode=disable" go run cmd/server/main.go

   # Terminal 3: Start frontend
   cd frontend && npm run dev
   ```

2. **Access the application:**
   - Frontend: http://localhost:4321
   - Backend API: http://localhost:8080
   - Database: localhost:5432

### Troubleshooting

#### Docker Issues
```bash
# Check if Docker daemon is running
docker info

# Check container logs
docker logs generic-saas-postgres

# Restart Docker service (macOS)
sudo systemctl restart docker
```

#### Database Connection Issues
```bash
# Test database connection
docker exec -it generic-saas-postgres pg_isready -U saas_user -d generic_saas

# View database logs
docker logs generic-saas-postgres
```

#### Port Conflicts
```bash
# Check what's using port 8080
lsof -i :8080

# Check what's using port 4321
lsof -i :4321

# Kill process on port
kill -9 <PID>
```

## Project Structure

```
generic-saas/
├── backend/
│   ├── cmd/server/          # Main application entry point
│   ├── internal/
│   │   ├── auth/            # Authentication services
│   │   ├── database/        # Database interfaces and implementations
│   │   │   ├── interface.go # Database interfaces
│   │   │   ├── memory.go    # In-memory implementation
│   │   │   ├── postgres.go  # PostgreSQL implementation
│   │   │   ├── factory.go   # Database factory
│   │   │   └── migrations.go # Migration system
│   │   └── middleware/      # HTTP middleware
│   ├── go.mod
│   └── go.sum
├── frontend/
│   ├── src/
│   │   ├── components/      # Astro and Solid.js components
│   │   ├── pages/          # Astro pages
│   │   └── styles/         # Tailwind CSS
│   ├── package.json
│   └── astro.config.mjs
└── README.md
```

## Features

- ✅ **User Authentication** - Registration and login system
- ✅ **Database Abstraction** - Easy switching between storage backends
- ✅ **PostgreSQL Integration** - Full SQL database support with migrations
- ✅ **In-Memory Fallback** - Development mode without external dependencies
- ✅ **Structured Logging** - JSON-formatted logs with request tracking
- ✅ **Graceful Shutdown** - Proper request handling during shutdown
- ✅ **CORS Support** - Cross-origin request handling
- ✅ **Error Recovery** - Panic recovery middleware
- ✅ **Request Logging** - HTTP request/response logging
- ✅ **Docker Support** - Containerized PostgreSQL setup

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests for new functionality
5. Ensure all tests pass
6. Submit a pull request

## License

MIT License - see LICENSE file for details