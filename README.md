<div align="center">

# TaskFlow

**DDD-Compliant Go Workflow Backend with Hardened State Machine**

[![Architecture: State--Machine](https://img.shields.io/badge/architecture-state--machine-000000.svg?style=flat-square)](https://github.com/AsaqeLee/taskflow)
[![Standard: DDD--Compliant](https://img.shields.io/badge/standard-ddd--compliant-000000.svg?style=flat-square)](https://github.com/AsaqeLee/taskflow)
[![Persistence: Polyglot](https://img.shields.io/badge/persistence-polyglot-000000.svg?style=flat-square)](https://github.com/AsaqeLee/taskflow)

English | [简体中文](./README_ZH.md)

</div>

---

## Introduction

**TaskFlow** is a production-grade blueprint for task collaboration and lifecycle management. It prioritizes explicit state transitions, repository-based persistence abstractions, and low-friction developer recovery. By decoupling business logic from infrastructure, it scales from simple task tracking to complex, multi-actor workflows.

>[!IMPORTANT]
>This system treats task lifecycles as a formal state machine. Every action (assign, start, submit, approve) is validated against current state and actor permissions to ensure zero illegal transitions.

>[!NOTE]
>Current runtime baseline includes password-based registration and `POST /auth/login`, JWT-only production auth, request/trace IDs, structured JSON logs, `/health` + `/livez` + `/readyz` + `/metrics`, Mongo index bootstrap, soft delete with audit retention, rate limiting, and idempotency-key replay.

---

## Workflow Architecture

The core engine enforces a strict lifecycle: `create -> assign -> start -> submit -> approve/reject -> close`.

```mermaid
graph LR
    Create([Create]) --> Assign[Assign]
    Assign --> Start[Start]
    Start --> Submit[Submit]
    Submit --> Review{Review}
    Review -- Reject --> Start
    Review -- Approve --> Close([Close])
    
    style Review fill:none,stroke:#000,stroke-width:2px
```

---

## Technical Specifications

<details>
<summary><b>Domain-Driven Design (DDD) Structure</b></summary>

```text
taskflow/
├── cmd/                # Entry points (HTTP Server)
├── internal/
│   ├── bootstrap/      # Dependency Injection & App Assembly
│   ├── service/        # Hardened State Machine & Business Rules
│   ├── repository/     # Persistence Abstractions (Mongo/Memory)
│   └── domain/         # Core Entities & Value Objects
├── docs/               # Boundary definitions and targets
└── scripts/            # Deployment and utility scripts
```
</details>

<details>
<summary><b>Dual-Persistence Driver Protocol</b></summary>

TaskFlow supports pluggable persistence through the Repository Pattern:
- **Memory Driver:** Optimized for lightning-fast local iteration and CI/CD testing.
- **Mongo Driver:** For production-grade persistence-path validation and horizontal scaling.
- **Switching:** Controlled via `TASK_REPOSITORY_DRIVER` environment variable without business logic changes.
</details>

<details>
<summary><b>Enterprise Installation & Usage</b></summary>

### Prerequisites
- Go 1.21 or higher
- MongoDB (optional, for production driver)

### Quick Start
```bash
# Clone the repository
git clone https://github.com/AsaqeLee/taskflow.git
cd taskflow

# Verify integrity
go test ./...

# Run local dev mode (enables seeded dev users and X-User-ID / legacy token fallback)
DEV_MODE=true TASK_REPOSITORY_DRIVER=memory JWT_SECRET=change-me go run ./cmd/server

# Register a password-based user
curl -X POST http://localhost:8080/users \
  -H 'Content-Type: application/json' \
  -d '{"id":"u_demo","name":"Demo User","role":"human","password":"strong-pass-123"}'

# Or login with an existing user
curl -X POST http://localhost:8080/auth/login \
  -H 'Content-Type: application/json' \
  -d '{"id":"u_demo","password":"strong-pass-123"}'
```
</details>

---

## Strategic Boundaries

- **Audit Traceability:** Every state change triggers an `AuditLog` for professional accountability.
- **Minimalist Identity:** Focused on lifecycle integrity; full OAuth is deferred to identity providers.
- **Clean Code:** Adheres to high-integrity Go standards with minimal third-party dependency bloat.

---

<div align="center">

&copy; 2026 AsaqeLee. Built for deterministic workflow orchestration.

</div>
