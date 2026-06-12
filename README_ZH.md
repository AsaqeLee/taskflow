<div align="center">

# TaskFlow

**遵循领域驱动设计 (DDD) 的 Go 语言工作流后端**

[![Architecture: State--Machine](https://img.shields.io/badge/architecture-state--machine-000000.svg?style=flat-square)](https://github.com/AsaqeLee/taskflow)
[![Standard: DDD--Compliant](https://img.shields.io/badge/standard-ddd--compliant-000000.svg?style=flat-square)](https://github.com/AsaqeLee/taskflow)
[![Persistence: Polyglot](https://img.shields.io/badge/persistence-polyglot-000000.svg?style=flat-square)](https://github.com/AsaqeLee/taskflow)

[English](./README.md) | 简体中文

</div>

---

## 项目简介

**TaskFlow** 是一个用于任务协作和生命周期管理的生产级蓝图。它优先考虑显式的状态转移、基于仓库模式 (Repository Pattern) 的持久化抽象以及低摩擦的开发恢复流程。通过将业务逻辑与基础设施解耦，它可以从简单的任务跟踪扩展到复杂的多角色协作工作流。

>[!IMPORTANT]
>本系统将任务生命周期视为形式化的状态机。每一个动作（分配、启动、提交、审批）都会根据当前状态和操作者权限进行验证，以确保零非法状态转移。

>[!NOTE]
>当前运行时基线已包含：基于密码的注册与 `POST /auth/login`、生产环境 JWT 鉴权、请求/链路 ID、结构化 JSON 日志、`/health` + `/livez` + `/readyz` + `/metrics`、Mongo 索引自举、软删除与审计保留、限流与幂等键回放。

---

## 核心架构

核心引擎强制执行严格的生命周期：`新建 -> 分配 -> 启动 -> 提交 -> 审批/驳回 -> 关闭`。

```mermaid
graph LR
    Create([新建]) --> Assign[分配]
    Assign --> Start[启动]
    Start --> Submit[提交]
    Submit --> Review{审批}
    Review -- 驳回 --> Start
    Review -- 通过 --> Close([关闭])
    
    style Review fill:none,stroke:#000,stroke-width:2px
```

---

## 技术规格

<details>
<summary><b>领域驱动设计 (DDD) 目录结构</b></summary>

```text
taskflow/
├── cmd/                # 入口点 (HTTP 服务)
├── internal/
│   ├── bootstrap/      # 依赖注入与应用组装
│   ├── service/        # 强约束状态机与业务规则
│   ├── repository/     # 持久化抽象 (MongoDB/内存)
│   └── domain/         # 核心实体与值对象
├── docs/               # 边界定义与项目目标
└── scripts/            # 部署与工具脚本
```
</details>

<details>
<summary><b>双持久化驱动协议 (Repository Pattern)</b></summary>

TaskFlow 通过仓库模式支持可插拔的持久化层：
- **内存驱动 (Memory Driver):** 针对极速本地迭代和 CI/CD 测试进行了优化。
- **Mongo 驱动 (Mongo Driver):** 用于生产级的持久化路径验证和水平扩展。
- **切换机制:** 通过 `TASK_REPOSITORY_DRIVER` 环境变量控制，无需修改业务逻辑。
</details>

<details>
<summary><b>企业级安装与使用</b></summary>

### 前置要求
- Go 1.21 或更高版本
- MongoDB (可选，用于生产驱动)

### 快速开始
```bash
# 克隆仓库
git clone https://github.com/AsaqeLee/taskflow.git
cd taskflow

# 完整性验证
go test ./...

# 本地开发模式启动（启用 seeded 测试用户与 X-User-ID / 旧 token fallback）
DEV_MODE=true TASK_REPOSITORY_DRIVER=memory JWT_SECRET=change-me go run ./cmd/server

# 注册带密码的新用户
curl -X POST http://localhost:8080/users \
  -H 'Content-Type: application/json' \
  -d '{"id":"u_demo","name":"Demo User","role":"human","password":"strong-pass-123"}'

# 或登录已有用户
curl -X POST http://localhost:8080/auth/login \
  -H 'Content-Type: application/json' \
  -d '{"id":"u_demo","password":"strong-pass-123"}'
```
</details>

---

## 战略边界

- **审计追踪:** 每一个状态变更都会触发 `AuditLog`，确保专业级的合规与追责。
- **轻量身份层:** 专注于生命周期完整性；完整的 OAuth/JWT 逻辑交由身份提供者处理。
- **高集成性:** 遵循高完整性的 Go 语言标准，最小化第三方依赖冗余。

---

<div align="center">

&copy; 2026 AsaqeLee. 为确定性工作流编排而生。

</div>
