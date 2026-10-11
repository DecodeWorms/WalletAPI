# High-Concurrency Distributed Wallet API

A production-grade, fault-tolerant financial wallet and transaction processing engine written in **Go (Golang)**. Designed to handle high-throughput balance operations while guaranteeing strict financial data consistency and race-condition prevention.

## 🚀 Core Architectural Features

- **ACID Transaction Management:** Implements strict database transaction isolation using PostgreSQL row-level locks (`SELECT ... FOR UPDATE`) to prevent concurrent race conditions, lost updates, and double-debit anomalies.
- **Distributed Idempotency Layer:** Features custom HTTP middleware leveraging Redis `SetNX` atomic distributed locks mapped to client-generated `X-Idempotency-Key` headers, guaranteeing safe retries over flaky network connections without duplicate charges.
- **Cache-Aside Read Scaling:** Employs the Cache-Aside pattern using Redis to serve high-volume wallet balance reads with sub-millisecond latency while maintaining clean cache invalidation on balance mutation.
- **API Rate Limiting:** Protects sensitive transfer endpoints against denial-of-service and bot traffic using a fixed-window counter algorithm in Redis.
- **Clean Architecture & Dependency Injection:** Decoupled storage, repository, service, and HTTP transport layers for modularity and testability.

## 🛠 Tech Stack

- **Language:** Go (Golang 1.22+)
- **Database:** PostgreSQL (ACID transactions, Row-level locking)
- **In-Memory Cache & Distributed Lock:** Redis (`go-redis/v9`)
- **Transport:** Standard Library `net/http` with custom middleware chaining

## 📡 Key Endpoints

- `POST /api/v1/transfer`: Execute atomic peer-to-peer wallet transfer (Protected by Rate Limiter & Idempotency Key).
- `GET /api/v1/wallet/{id}/balance`: Fetch cached wallet balance (Cache-Aside pattern).
