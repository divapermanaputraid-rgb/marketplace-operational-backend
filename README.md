# Marketplace Operations API — Backend

> **Note to Agents and Developers:**
> All project-wide documentation, system rules, architecture, and API specifications have been consolidated into the root workspace `docs/` folder. Please refer to `../../docs/` (or the root `docs/` folder of the workspace) for shared context before modifying this backend.

Go backend for the Marketplace Operations System.

**Stack:** Go + Gin + GORM + Supabase PostgreSQL

## Implemented Modules (Internal MVP)
- Admin Auth (JWT)
- Stores / Marketplace Accounts
- Product Master
- Product Mapping
- Inventory Management & Movement History
- Orders Foundation (Internal processing)
- Sync Center Foundation (Job tracking and Logs)
- Dashboard & Operational Reports

## Prerequisites

- Go 1.21+ installed
- Supabase PostgreSQL database (free tier works)
- Access to the Supabase dashboard for DATABASE_URL

## Local Setup

### 1. Navigate to backend directory

```bash
cd backend
```

### 2. Copy environment file

```bash
cp .env.example .env
```

### 3. Configure .env

Fill in the required values:

```env
APP_ENV=development
PORT=8080
DATABASE_URL=postgresql://postgres:YOUR_PASSWORD@db.YOUR_PROJECT.supabase.co:5432/postgres
JWT_SECRET=your-secure-random-secret
CORS_ALLOWED_ORIGINS=http://localhost:5173
ADMIN_SEED_EMAIL=admin@marketops.local
ADMIN_SEED_PASSWORD=Admin123!
ADMIN_SEED_NAME=Admin
```

### 4. Install dependencies

```bash
go mod download
```

### 5. Run the server

```bash
go run ./cmd/api
```

Expected output:

```
✅ Database connected successfully
🔄 Running auto-migration...
✅ Auto-migration completed
✅ Admin seeded: admin@marketops.local (Admin)
🚀 Marketplace Operations API starting on :8080 (env: development)
📍 Health check: http://localhost:8080/api/health
```

## API Endpoints

### Health Check

```
GET /api/health
```

Response:

```json
{
  "status": "ok",
  "service": "marketplace-ops-api"
}
```

## Database Migration

### Development (Auto-migration)

In development mode (`APP_ENV=development`), GORM auto-migration runs automatically on startup.

### Production (SQL Migration)

For production, apply the SQL migration files manually using the Supabase SQL Editor. Run files `001` through `007` sequentially.

## Admin Seed

The server automatically seeds a default admin account on startup if:

1. `ADMIN_SEED_EMAIL` and `ADMIN_SEED_PASSWORD` are set
2. No admin with that email exists yet

Password is hashed with bcrypt before storage.

## Build

```bash
go build -o server ./cmd/api
```

## Docker Build

```bash
docker build -t marketplace-ops-api .
docker run -p 8080:8080 --env-file .env marketplace-ops-api
```

## Render Deployment

This project includes a `Dockerfile` and `render.yaml` for easy deployment to Render Free.

1. Connect your GitHub repository to Render.
2. Select **Web Service**.
3. Use the `Docker` environment.
4. Set the root directory to `backend`.
5. Set environment variables in the Render dashboard:
   - `APP_ENV=production`
   - `DATABASE_URL` (Use Supabase connection pooler URL, e.g., port 6543)
   - `JWT_SECRET` (Must be a secure random string)
   - `CORS_ALLOWED_ORIGINS` (e.g., `https://your-frontend.vercel.app`)

*Note: In production, the backend will refuse to start if weak default secrets are used.*

## Project Structure

```
backend/
├── cmd/api/
│   └── main.go                    # Application entry point
├── internal/
│   ├── config/
│   │   └── config.go              # Environment configuration
│   ├── database/
│   │   ├── database.go            # GORM connection + auto-migrate
│   │   └── seed.go                # Admin seed logic
│   ├── handlers/
│   │   └── health.go              # Health check handler
│   ├── middleware/
│   │   ├── auth.go                # JWT auth middleware (Sprint 2)
│   │   └── cors.go                # CORS configuration
│   ├── models/
│   │   ├── admin.go               # Admin GORM model
│   │   └── response.go            # Standard API response format
│   ├── repositories/
│   │   └── admin_repository.go    # Admin database operations
│   └── services/
│       └── jwt.go                 # JWT token service
├── migrations/
│   └── 001_create_admins_table.sql
├── .env.example
├── Dockerfile
├── go.mod
├── go.sum
└── README.md
```

## Formatting

```bash
gofmt -w .
```

## Testing

```bash
go test ./...
```
*

