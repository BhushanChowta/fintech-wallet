# Fintech Wallet API

Go-based REST API for a wallet system with authentication, balance management, money transfer, and transaction history.

This README explains the project structure and implementation flow in depth so you can quickly understand how requests move through the codebase and where to extend functionality safely.

## 1) Tech Stack

| Layer | Technology |
| --- | --- |
| Language | Go 1.23 |
| HTTP framework | Gin |
| ORM | GORM |
| Database | PostgreSQL |
| Cache | Redis |
| Auth | JWT (HMAC-SHA256) |
| Password security | bcrypt |
| Infra | Docker, docker-compose |

## 2) High-Level Architecture

The code follows a layered architecture:

`HTTP Handlers -> Services -> Repositories -> Postgres/Redis`

- Handlers parse input, call business logic, and map errors to HTTP responses.
- Services hold business rules and transaction boundaries.
- Repositories isolate data access logic.
- Models define schema and serialization contracts.

This separation keeps logic easier to test and change, especially in fintech flows where data correctness matters.

## 3) Folder-by-Folder Code Structure

```text
cmd/
  api/
    main.go                  # Application bootstrap, dependency wiring, route setup

config/
  config.go                  # Environment loading (.env + OS env), default values

internal/
  auth/
    jwt.go                   # JWT claim model, token generation, token validation
  handlers/
    handlers.go              # Gin handlers for auth + wallet endpoints
  middleware/
    auth.go                  # Bearer token middleware and request context injection
  models/
    models.go                # User/Wallet/Transaction entities + enums + UUID hooks
  repository/
    repository.go            # Postgres and Redis data access methods
  service/
    service.go               # Auth and wallet business logic + DB transactions

pkg/
  database/
    database.go              # Postgres and Redis connection initialization

Dockerfile                  # Multi-stage build for small runtime image
docker-compose.yml          # API + Postgres + Redis + pgAdmin local stack
.env.example                # Required environment variables template
```

## 4) Application Startup Flow

Defined in `cmd/api/main.go`:

1. Load configuration using `config.Load()`.
2. Initialize PostgreSQL and Redis clients (`pkg/database`).
3. Auto-migrate models: `User`, `Wallet`, `Transaction`.
4. Wire dependencies manually:
- `JWTService`
- `UserRepository`, `WalletRepository`, `TransactionRepository`
- `AuthService`, `WalletService`
- `AuthHandler`, `WalletHandler`
5. Register routes:
- Public: `/health`, `/api/v1/auth/register`, `/api/v1/auth/login`
- Protected with JWT middleware: wallet routes
6. Start Gin server on `APP_PORT`.

This explicit wiring (instead of reflection-heavy DI frameworks) makes ownership of dependencies very clear.

## 5) Layer Responsibilities in Detail

### `config` layer

`config/config.go` reads `.env` with `godotenv`, then falls back to OS env variables/defaults.

Main values:
- API server port
- Postgres host/port/user/password/db
- Redis host/port
- JWT secret + expiration hours

### `models` layer

`internal/models/models.go` contains:

- `User`
- `Wallet` (1:1 with user via unique `UserID`)
- `Transaction`

Notable design points:
- UUID primary keys for all core entities.
- `BeforeCreate` hooks auto-generate UUIDs if missing.
- `TransactionType` enum: `CREDIT`, `DEBIT`, `TRANSFER`.
- `TransactionStatus` enum: `SUCCESS`, `FAILED`, `PENDING`.
- `PasswordHash` is excluded from API JSON output using `json:"-"`.

### `repository` layer

`internal/repository/repository.go` handles persistence and cache reads.

- `UserRepository`
  - `FindByEmail` for login
  - `FindByID` checks Redis first, then Postgres fallback, then caches result for 5 minutes
- `WalletRepository`
  - Fetch wallet by user or wallet ID
  - Update wallet balance
- `TransactionRepository`
  - Insert transaction records
  - Fetch recent transactions for a wallet (`limit 50`, newest first)

### `service` layer

`internal/service/service.go` contains business logic.

- `AuthService`
  - Register:
    - Hash password with bcrypt
    - Create user and wallet in a single DB transaction
    - Generate JWT token
  - Login:
    - Lookup user by email
    - Compare bcrypt hash
    - Generate JWT token

- `WalletService`
  - `GetWallet`: fetch wallet for authenticated user
  - `Deposit`: add balance + create CREDIT transaction
  - `Transfer`:
    - prevent self-transfer
    - check sender balance
    - update sender and receiver balances
    - create TRANSFER transaction
    - all steps wrapped in one DB transaction for atomicity
  - `GetTransactions`: list latest transactions for user wallet

### `handlers` layer

`internal/handlers/handlers.go` is the API boundary.

- Binds request JSON into service DTOs (with Gin binding tags).
- Reads authenticated user id from context (set by middleware).
- Calls service methods.
- Returns HTTP status codes and JSON responses.

### `auth` + `middleware` layers

- `internal/auth/jwt.go`
  - Builds signed JWT tokens with expiry and issue timestamp.
  - Validates tokens and returns typed claims.

- `internal/middleware/auth.go`
  - Requires `Authorization: Bearer <token>`.
  - Validates token via `JWTService`.
  - Injects `userID` and `email` into Gin context for downstream handlers.

## 6) Request Lifecycle Example

Example: `POST /api/v1/wallet/transfer`

1. Request hits Gin router in `main.go`.
2. JWT middleware validates token and sets `userID` in context.
3. `WalletHandler.Transfer` binds and validates JSON payload.
4. `WalletService.Transfer` enforces business checks:
- cannot transfer to yourself
- sender wallet exists
- sender has enough balance
- receiver wallet exists
5. Service opens DB transaction and performs:
- sender debit
- receiver credit
- transaction row insert
6. Handler returns success response with transaction payload.

If any DB operation fails, the transaction is rolled back to avoid partial money movement.

## 7) API Endpoints

### Public

```text
GET  /health
POST /api/v1/auth/register
POST /api/v1/auth/login
```

### Protected (Bearer token required)

```text
GET  /api/v1/wallet
POST /api/v1/wallet/deposit
POST /api/v1/wallet/transfer
GET  /api/v1/wallet/transactions
```

## 8) Run Locally

### Option A: Docker Compose (recommended)

```bash
docker-compose up --build
```

Services started:
- API on `http://localhost:8080`
- PostgreSQL on `localhost:5432`
- Redis on `localhost:6379`
- pgAdmin on `http://localhost:5050`

### Option B: Run API directly

1. Start PostgreSQL and Redis locally.
2. Create `.env` from `.env.example`.
3. Run:

```bash
go run ./cmd/api
```

## 9) Environment Variables

Defined in `.env.example`:

```env
APP_PORT=8080

DB_HOST=postgres
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=postgres
DB_NAME=fintech_wallet

REDIS_HOST=redis
REDIS_PORT=6379

JWT_SECRET=super-secret-jwt-key-change-in-production
JWT_EXPIRY_HOURS=24
```

For production, always rotate `JWT_SECRET` and avoid committing real credentials.

## 10) Data Integrity and Safety Choices

- Passwords are stored as bcrypt hashes, never plaintext.
- JWT protects private wallet endpoints.
- UUID IDs reduce identifier predictability.
- Transfer flow is atomic through DB transactions.
- Redis cache is read-through style for user lookup by ID.

## 11) Current Limitations and Next Improvements

- Monetary values are stored as `float64`; consider integer minor units (paise/cents) or fixed decimal library for financial precision.
- Transfer currently updates balances without row-level locking; for high concurrency, add locking strategy (`SELECT ... FOR UPDATE`) to prevent race conditions.
- Error mapping can be standardized with typed/domain errors for cleaner HTTP responses.
- Add test suite (unit + integration) for critical money movement paths.
- Add structured logs, request IDs, and metrics for observability.

## 12) Example cURL Requests

### Register

```bash
curl -X POST http://localhost:8080/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{"name":"User One","email":"user1@example.com","password":"secret123"}'
```

### Login

```bash
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"user1@example.com","password":"secret123"}'
```

### Deposit

```bash
curl -X POST http://localhost:8080/api/v1/wallet/deposit \
  -H "Authorization: Bearer <TOKEN>" \
  -H "Content-Type: application/json" \
  -d '{"amount":1000,"description":"initial deposit"}'
```

### Transfer

```bash
curl -X POST http://localhost:8080/api/v1/wallet/transfer \
  -H "Authorization: Bearer <TOKEN>" \
  -H "Content-Type: application/json" \
  -d '{"receiver_user_id":"<RECEIVER_USER_UUID>","amount":500,"description":"rent split"}'
```
