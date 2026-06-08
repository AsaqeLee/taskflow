# TaskFlow

> **DDD-Compliant Go Workflow Backend** featuring a hardened **State Machine** and **Dual-Persistence** architecture (`memory` + `mongo`).

TaskFlow is a production-grade blueprint for task collaboration and lifecycle management. It prioritizes explicit state transitions, repository-based persistence abstractions, and low-friction developer recovery.

## 🏗 Key Architectural Decisions

### 1. Hardened State Machine
Unlike simple CRUD apps, TaskFlow treats task lifecycles as a formal state machine. Every action (`assign`, `start`, `submit`, `approve`) is validated against current state and actor permissions, ensuring zero illegal transitions.

### 2. DDD & Repository Pattern
Built with Domain-Driven Design principles. The core business logic is isolated from infrastructure, allowing the system to scale from simple task tracking to complex, multi-actor workflows without code rot.

### 3. Dual-Persistence Drivers
- **Memory Driver:** Optimized for lightning-fast local iteration and CI/CD testing.
- **Mongo Driver:** For production-grade persistence-path validation and horizontal scaling.
- *Switchable via environment variables without changing a single line of business logic.*

### 4. Audit & Collaboration Traceability
Every state change triggers an `AuditLog` and `TaskRecord`, providing a verifiable history of who did what, when, and why—essential for professional collaboration and AI-augmented workflows.

---

## 🚀 Quick Overview

### Tech Stack
`Go` · `Gin` · `MongoDB` · `REST API` · `Repository Pattern` · `State Machine`

### Current Lifecycle
`create -> assign -> start -> submit -> approve/reject -> close`

---

## 🛠 Local Recovery (5-Minute Path)
If you've been away for a while, use this sequence to verify the project is "alive":
1. `go test ./...`
2. `TASK_REPOSITORY_DRIVER=memory go run ./cmd/server`
3. Check `http://localhost:8080/health`

---

## 📂 Directory Structure
- `internal/service`: Hardened state machine and business rules.
- `internal/repository`: Persistence abstractions and driver implementations.
- `internal/bootstrap`: Application assembly and dependency injection.

---

## ⚖️ Boundaries (V0)
- **Included:** Hardened lifecycle, dual persistence, audit trails, minimal dynamic identity.
- **Deferred:** Full JWT/OAuth, sub-tasks, general-purpose workflow engine.

---

## 相关文档
- `docs/项目目标.md`
- `docs/V0 模块与职责边界.md`
