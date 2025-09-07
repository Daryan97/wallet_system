# Wallet System Backend

A wallet management system backend written in Go, using Gin, Gorm, MySQL, Redis, JWT, bcrypt, and logrus.

## Table of Contents

- [Features](#features)
- [Tech Stack](#tech-stack)
- [Getting Started](#getting-started)
  - [Prerequisites](#prerequisites)
  - [Setup](#setup)
- [API Endpoints](#api-endpoints)
- [API Input/Output Examples](#api-inputoutput-examples)
  - [Register User](#register-user)
  - [Login](#login)
  - [Deposit Funds](#deposit-funds)
  - [Transfer Funds](#transfer-funds)
  - [Get Wallet Info](#get-wallet-info)
  - [Get Transaction History](#get-transaction-history)
  - [Admin: List Users](#admin-list-users)
  - [Admin: List Transactions](#admin-list-transactions)
- [Logging & Monitoring](#logging--monitoring)
- [Caching](#caching)
- [Development](#development)

## Features

- User registration and login (JWT authentication)
- Secure password hashing (bcrypt)
- Wallet creation, deposit, and transfer
- Transaction history with pagination
- Admin endpoints for user and transaction management
- Role-based access control (admin/user)
- Redis caching for wallet and transaction data
- Logging and audit trail with logrus
- Configurable via `.env` file

## Tech Stack

- [Go](https://golang.org/)
- [Gin](https://github.com/gin-gonic/gin)
- [Gorm](https://gorm.io/)
- [MySQL](https://www.mysql.com/)
- [Redis](https://redis.io/)
- [JWT](https://github.com/golang-jwt/jwt)
- [logrus](https://github.com/sirupsen/logrus)
- [bcrypt](https://pkg.go.dev/golang.org/x/crypto/bcrypt)

## Getting Started

### Prerequisites

- Go 1.20+
- MySQL
- Redis

### Setup

1. Copy `.env.example` to `.env` and fill in your configuration.
2. Run database migration:

   ```sh
   go run ./cmd/migrate/main.go
   ```

3. Start the server:

   ```sh
   go run ./cmd/server/main.go
   ```

### API Endpoints

#### Auth

- `POST /user` - Register
- `GET /user` - Login

#### Wallet (JWT required)

- `POST /wallet` — Create wallet
- `GET /wallet` — Get wallet info
- `POST /wallet/deposit` — Deposit funds
- `POST /wallet/transfer` — Transfer funds
- `GET /wallet/transactions` — Transaction history

#### Admin (JWT + admin role required)

- `GET /admin/users` — List users
- `GET /admin/transactions` — List transactions

---

## API Input/Output Examples

Imagine the server is running at `http://localhost:8080`, here are some example requests and responses, replace `<JWT_TOKEN>` and `<ADMIN_JWT_TOKEN>` with actual tokens.

### Register User

**Request:**

```http
POST http://localhost:8080/user HTTP/1.1
Content-Type: application/json

{
  "username": "alice",
  "password": "password123"
}
```

**Success Response:**

```http
HTTP/1.1 200 OK
Content-Type: application/json

{
  "message": "User registered successfully"
}
```

**Error Response (username taken):**

```http
HTTP/1.1 400 Bad Request
Content-Type: application/json

{
  "error": "Username already exists"
}
```

### Login

**Request:**

```http
GET http://localhost:8080/user HTTP/1.1
Content-Type: application/json

{
  "username": "alice",
  "password": "password123"
}
```

**Success Response:**

```http
HTTP/1.1 200 OK
Content-Type: application/json

{
  "token": "<JWT_TOKEN>"
}
```

**Error Response (invalid credentials):**

```http
HTTP/1.1 401 Unauthorized
Content-Type: application/json

{
  "error": "Invalid username or password"
}
```

### Deposit Funds

**Request:**

```http
POST http://localhost:8080/wallet/deposit HTTP/1.1
Authorization: Bearer <JWT_TOKEN>
Content-Type: application/json

{
  "amount": 100.0
}
```

**Success Response:**

```http
HTTP/1.1 200 OK
Content-Type: application/json

{
  "message": "Deposit successful"
}
```

**Error Response (invalid amount):**

```http
HTTP/1.1 400 Bad Request
Content-Type: application/json

{
  "error": "Invalid amount"
}
```

### Transfer Funds

**Request:**

```http
POST http://localhost:8080/wallet/transfer HTTP/1.1
Authorization: Bearer <JWT_TOKEN>
Content-Type: application/json

{
  "to_username": "bob",
  "amount": 50.0
}
```

**Success Response:**

```http
HTTP/1.1 200 OK
Content-Type: application/json

{
  "message": "Transfer successful"
}
```

**Error Response (insufficient funds):**

```http
HTTP/1.1 400 Bad Request
Content-Type: application/json

{
  "error": "Insufficient funds"
}
```

### Get Wallet Info

**Request:**

```http
GET http://localhost:8080/wallet HTTP/1.1
Authorization: Bearer <JWT_TOKEN>
```

**Success Response:**

```http
HTTP/1.1 200 OK
Content-Type: application/json

{
  "wallet": {
    "id": 1,
    "user_id": 1,
    "balance": 50.0
  },
  "cached": false
}
```

**Error Response (not found):**

```http
HTTP/1.1 404 Not Found
Content-Type: application/json

{
  "error": "Wallet not found"
}
```

### Get Transaction History

**Request:**

```http
GET http://localhost:8080/wallet/transactions?page=1&page_size=10 HTTP/1.1
Authorization: Bearer <JWT_TOKEN>
```

**Success Response:**

```http
HTTP/1.1 200 OK
Content-Type: application/json

{
  "transactions": [
    {
      "id": 1,
      "from_wallet_id": 1,
      "to_wallet_id": 2,
      "amount": 50.0,
      "type": "transfer",
      "created_at": "2025-09-07T12:00:00Z"
    }
  ],
  "page": 1,
  "page_size": 10,
  "total": 1,
  "total_pages": 1,
  "cached": false
}
```

**Error Response (not found):**

```http
HTTP/1.1 404 Not Found
Content-Type: application/json

{
  "error": "Wallet not found"
}
```

### Admin: List Users

**Note:** Admin endpoints require a JWT from a user with *role=admin*. Promote a user in MySQL, then log in again to get a new token:

```sql
UPDATE users SET role = 'admin' WHERE username = 'alice';
```

**Request:**

```http
GET http://localhost:8080/admin/users?page=1&page_size=10 HTTP/1.1
Authorization: Bearer <ADMIN_JWT_TOKEN>
```

**Success Response:**

```http
HTTP/1.1 200 OK
Content-Type: application/json

{
  "users": [
    {
      "id": 1,
      "username": "alice",
      "role": "user",
      "wallet": {
        "id": 1,
        "user_id": 1,
        "balance": 50.0
      }
    }
  ],
  "page": 1,
  "page_size": 10,
  "total": 1,
  "total_pages": 1,
  "cached": false
}
```

**Error Response (unauthorized):**

```http
HTTP/1.1 401 Unauthorized
Content-Type: application/json

{
  "error": "Unauthorized"
}
```

### Admin: List Transactions

**Request:**

```http
GET http://localhost:8080/admin/transactions?page=1&page_size=10 HTTP/1.1
Authorization: Bearer <ADMIN_JWT_TOKEN>
```

**Success Response:**

```http
HTTP/1.1 200 OK
Content-Type: application/json

{
  "transactions": [
    {
      "id": 1,
      "from_wallet_id": 1,
      "to_wallet_id": 2,
      "amount": 50.0,
      "type": "transfer",
      "created_at": "2025-09-07T12:00:00Z"
    }
  ],
  "page": 1,
  "page_size": 10,
  "total": 1,
  "total_pages": 1,
  "cached": false
}
```

**Error Response (unauthorized):**

```http
HTTP/1.1 401 Unauthorized
Content-Type: application/json

{
  "error": "Unauthorized"
}
```

---

### Logging & Monitoring

- All errors and financial transactions are logged using logrus.
- Logs include user IDs, amounts, and timestamps for audit purposes.

### Caching

- Redis is used to cache wallet info and transaction history for performance.
- Cache is invalidated on data changes (deposit, transfer, etc).

## Development

- Code is organized in `internal/` by domain, API, middleware, config, and utils.
- Environment variables are loaded from `.env` (see `.env.example`).
- Use `IS_PROD=true` for production.
