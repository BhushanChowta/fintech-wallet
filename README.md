# 💰 Fintech Wallet API

A production-ready REST API built with **Go (Golang)** simulating a digital wallet system — similar to PayPal or Paytm internals.

## Tech Stack

| Layer | Technology |
|-------|-----------|
| Language | Go 1.22 |
| Framework | Gin |
| Database | PostgreSQL + GORM |
| Cache | Redis |
| Auth | JWT |
| Infra | Docker + docker-compose |

## Architecture

```
cmd/api/          → main entrypoint
config/           → environment config loader
internal/
  auth/           → JWT token generation & validation
  handlers/       → HTTP request/response layer
  middleware/     → JWT auth middleware
  models/         → GORM models (User, Wallet, Transaction)
  repository/     → DB + Redis data access layer
  service/        → business logic layer
pkg/database/     → Postgres & Redis initializers
```

## Features

- ✅ User registration & login with bcrypt password hashing
- ✅ JWT-based authentication
- ✅ Wallet auto-created on registration
- ✅ Deposit funds into wallet
- ✅ Transfer funds between users (atomic DB transaction)
- ✅ Transaction history
- ✅ Redis caching for user profile lookups
- ✅ Docker + docker-compose for local dev

## Getting Started

```bash
# Clone the repo
git clone https://github.com/bhushanchowta/fintech-wallet
cd fintech-wallet

# Start everything with Docker
docker-compose up --build

# API is live at http://localhost:8080
```

## API Endpoints

### Public
```
POST /api/v1/auth/register    Register a new user
POST /api/v1/auth/login       Login and get JWT token
GET  /health                  Health check
```

### Protected (requires `Authorization: Bearer <token>`)
```
GET  /api/v1/wallet                  Get wallet balance
POST /api/v1/wallet/deposit          Deposit money
POST /api/v1/wallet/transfer         Transfer to another user
GET  /api/v1/wallet/transactions     Transaction history
```

## Example Requests

### Register
```bash
curl -X POST http://localhost:8080/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{"name":"Bhushan Chowta","email":"bhushan@example.com","password":"secret123"}'
```

### Login
```bash
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"bhushan@example.com","password":"secret123"}'
```

### Deposit
```bash
curl -X POST http://localhost:8080/api/v1/wallet/deposit \
  -H "Authorization: Bearer <your_token>" \
  -H "Content-Type: application/json" \
  -d '{"amount":1000,"description":"initial deposit"}'
```

### Transfer
```bash
curl -X POST http://localhost:8080/api/v1/wallet/transfer \
  -H "Authorization: Bearer <your_token>" \
  -H "Content-Type: application/json" \
  -d '{"receiver_user_id":"<uuid>","amount":500,"description":"splitting bill"}'
```

## Key Design Decisions

- **Atomic transfers**: Debit + credit happen inside a single DB transaction — no partial state
- **Redis caching**: User profiles cached for 5 minutes, invalidated on update
- **Layered architecture**: handlers → service → repository (clean separation of concerns)
- **UUID primary keys**: No sequential IDs exposed externally (security best practice)
- **bcrypt**: Industry-standard password hashing with cost factor

## Interview Talking Points

1. Why use DB transactions for transfer? → Atomicity, prevents double-spend
2. Why Redis cache? → Reduce DB load on high-traffic read-heavy endpoints
3. Why layered architecture? → Testability, single responsibility, easy to swap implementations
4. How would you scale this? → Horizontal scaling + DB connection pooling + message queue for async transfers
