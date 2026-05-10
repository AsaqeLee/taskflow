# TaskFlow

TaskFlow 是一个面向任务协作 / 工单管理场景的 Go 后端练习项目。

当前目标不是一次性做完整用户系统，而是先围绕 V0 范围搭出可运行骨架，并逐步打通任务闭环主链。

## Architecture

当前 TaskFlow 采用分层后端结构：

```mermaid
flowchart LR
    Client[Client / Curl / Frontend]
    Main[cmd/server/main.go]
    Config[internal/config]
    App[internal/bootstrap.App]
    Router[internal/router]
    Middleware[internal/middleware.FixedTestUser]
    Handlers[internal/handler<br/>Health / Me / Task]
    Service[internal/service.TaskService]
    Repo[internal/repository.TaskRepository]
    MemoryRepo[internal/repository.MemoryTaskRepository]
    Models[internal/model<br/>User / Task]
    DB[internal/database<br/>Mongo Client]

    Client --> Router
    Main --> Config
    Main --> App

    App --> Router
    App --> Service
    App --> Repo
    App --> DB

    Router --> Middleware
    Router --> Handlers

    Middleware --> Models
    Handlers --> Service
    Handlers --> Models

    Service --> Repo
    Service --> Models

    Repo --> MemoryRepo
```

### Layer responsibilities

- `cmd/server/main.go`
  - 程序启动入口
  - 加载配置并启动应用

- `internal/config`
  - 加载运行配置，例如端口和 MongoDB 设置

- `internal/bootstrap`
  - 负责应用装配
  - 构建 router、service、repository 和 database client

- `internal/router`
  - 注册 Gin 路由

- `internal/middleware`
  - 将固定测试用户注入请求上下文

- `internal/handler`
  - 处理 HTTP 请求与响应
  - 解析输入，并将 service 错误映射成 HTTP 状态码

- `internal/service`
  - 承载业务规则与校验逻辑

- `internal/repository`
  - 定义数据访问抽象
  - 当前实现仍使用内存存储

- `internal/model`
  - 定义核心数据结构，例如 `User` 和 `Task`

- `internal/database`
  - 初始化 MongoDB 客户端基础设施

### Current status

- MongoDB client bootstrap 已接入
- Task 数据当前仍通过 `MemoryTaskRepository` 管理
- 当前可运行接口：
  - `GET /health`
  - `GET /me`
  - `POST /tasks`
  - `GET /tasks`
  - `GET /tasks/:id`
  - `PATCH /tasks/:id`
  - `POST /tasks/:id/assign`
  - `POST /tasks/:id/start`
  - `POST /tasks/:id/submit`
  - `POST /tasks/:id/reject`
  - `POST /tasks/:id/approve`
  - `POST /tasks/:id/close`
- `assign` 已完成主链手测，当前已确认它作为第一个动作型接口可以跑通 `create -> assign` 这条最小链路
- `start` 已完成最小闭环：当前仅允许执行者将任务从 `assigned` 推进到 `in_progress`
- `submit` 已完成最小闭环：当前仅允许执行者将任务从 `in_progress` 推进到 `submitted`
- `reject` 已完成最小闭环：当前仅允许创建者将任务从 `submitted` 驳回到 `assigned`
- `approve` 已完成最小闭环：当前仅允许创建者将任务从 `submitted` 推进到 `approved`
- `close` 已完成最小闭环：当前仅允许创建者将任务从 `approved` 推进到 `completed`

## 当前进度

当前已完成的能力已经不再只是最小启动骨架，还包括：
- Gin 作为 Web 框架
- `cmd/server/main.go` 作为启动入口
- `internal/bootstrap` 负责应用装配
- `internal/config` 负责最小配置加载
- `internal/router` 负责路由注册
- `internal/handler` 提供 `/health`、`/me` 与 Task HTTP 接口
- `internal/database` 提供 MongoDB 初始化骨架
- `internal/middleware` 提供固定测试用户注入
- `internal/model` 提供最小 `User` 与 `Task` 结构
- `internal/service` 已承载 Task 基础校验与 `assign` / `start` / `submit` / `reject` / `approve` / `close` 动作规则
- `internal/repository` 已提供内存版 TaskRepository

当前可运行接口：
- `GET /health`
- `GET /me`
- `POST /tasks`
- `GET /tasks`
- `GET /tasks/:id`
- `PATCH /tasks/:id`
- `POST /tasks/:id/assign`
- `POST /tasks/:id/start`
- `POST /tasks/:id/submit`
- `POST /tasks/:id/reject`
- `POST /tasks/:id/approve`
- `POST /tasks/:id/close`

当前已明确落地的业务规则：
- `title` 不能为空
- `title` 长度至少 3 个字符
- `assign` 需要最小请求体字段 `assignee_id`
- 当前仅允许创建者执行 `assign`
- 当前仅允许 `open -> assigned`
- 当前版本不支持重新分配
- `start` 当前不需要请求体
- 当前仅允许 assignee 执行 `start`
- 当前仅允许 `assigned -> in_progress`
- `submit` 当前不需要请求体
- 当前仅允许 assignee 执行 `submit`
- 当前仅允许 `in_progress -> submitted`
- `reject` 当前不需要请求体
- 当前仅允许创建者执行 `reject`
- 当前仅允许 `submitted -> assigned`
- `approve` 当前不需要请求体
- 当前仅允许创建者执行 `approve`
- 当前仅允许 `submitted -> approved`
- `close` 当前不需要请求体
- 当前仅允许创建者执行 `close`
- 当前仅允许 `approved -> completed`

返回示例：

```json
{
  "status": "ok"
}
```

```json
{
  "user": {
    "id": "u_test_001",
    "name": "Test User",
    "role": "owner"
  }
}
```

## 本地启动

### 1. 安装依赖

```bash
go mod tidy
```

### 2. 准备环境变量

```bash
cp .env.example .env
```

当前默认环境变量：
- `PORT=8080`
- `MONGODB_URI=mongodb://localhost:27017`
- `MONGODB_DATABASE=taskflow`

### 3. 启动服务

```bash
go run ./cmd/server
```

默认监听端口：`8080`

也可以通过环境变量覆盖：

```bash
PORT=8081 MONGODB_URI=mongodb://localhost:27017 MONGODB_DATABASE=taskflow go run ./cmd/server
```

### 4. 检查服务

```bash
curl http://localhost:8080/health
curl http://localhost:8080/me
```

## 当前 V0 边界

Week 2 当前优先级：
1. 项目骨架
2. 配置 / 路由 / 数据库接线
3. 最小身份能力
4. Task 基础能力
5. 动作型接口：`assign / start / submit / approve / reject / close`

当前明确先不做：
- 完整注册 / 登录系统
- JWT 完整方案
- 子任务
- 多协作者关系表
- Redis
- 异步任务
- 通用工作流引擎

## 下一步

下一步应优先进入动作链收尾与配套能力阶段：
- 继续收紧 `assign / start / submit / reject / approve / close` 的错误边界与返回语义
- 为动作型接口补统一的操作记录抽象
- 在动作链逐渐稳定后，再补 `TaskRecord` / `AuditLog` 与更真实的持久化实现
