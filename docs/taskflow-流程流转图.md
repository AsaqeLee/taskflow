# TaskFlow 流程流转图

本文基于当前仓库实现整理，目标是把 TaskFlow 从应用启动、用户进入、任务动作、状态流转，到 `TaskRecord` / `AuditLog` 留痕与删除清理的完整过程画成可维护的 Mermaid 图。事实来源以代码和测试为准，见文末引用。

## 1. 系统处理流

```mermaid
flowchart TD
    A[启动服务] --> B[加载配置]
    B --> C[bootstrap.NewApp]
    C --> D{RepositoryDriver}
    D -->|mongo| E[初始化 Mongo Client]
    D -->|memory| F[初始化 Memory Repositories]
    E --> G[组装 Task/User/Record/Audit Repositories]
    F --> G
    G --> H[seedDefaultUsers]
    H --> I[构建 TaskService]
    I --> J[构建 TaskHandler 与 IdentityHandler]
    J --> K[注册 Router]
    K --> L[启动 Gin Server]

    L --> M[客户端请求]
    M --> N{访问 POST /users?}
    N -->|是| O[IdentityHandler.Register]
    O --> P[生成用户 Token 并写入 UserRepository]
    P --> Q[返回 201 user]

    N -->|否| R[进入鉴权路由组]
    R --> S[UserAuth]
    S --> T{Bearer token 命中?}
    T -->|是| U[按 token 查 UserRepository]
    T -->|否| V[X-User-ID 查 UserRepository]
    U --> W{找到用户?}
    V --> W
    W -->|否| X[返回 401 unauthorized]
    W -->|是| Y[写入 currentUser]

    Y --> Z{任务动作}
    Z -->|create/list/get/patch| AA[TaskHandler]
    Z -->|assign/start/submit/reject/approve/close/cancel/reactivate/delete| AA
    Z -->|records/audit_logs| AA
    AA --> AB[参数绑定与 CurrentUser 读取]
    AB --> AC[TaskService]
    AC --> AD{dbClient 存在?}
    AD -->|是| AE[RunTransaction]
    AD -->|否| AF[直接执行 runOps]
    AE --> AG[读取并校验 Task]
    AF --> AG

    AG --> AH{动作是否合法?}
    AH -->|否| AI[返回业务错误]
    AI --> AJ[handler 映射 400/403/404/500]

    AH -->|是| AK[更新 Task 状态/字段]
    AK --> AL[写 TaskRepository]
    AL --> AM{是否需要 TaskRecord?}
    AM -->|submit/reject/approve/cancel/reactivate| AN[写 TaskRecordRepository]
    AM -->|其他动作| AO[跳过 TaskRecord]
    AN --> AP[写 AuditLogRepository]
    AO --> AP
    AP --> AQ[返回任务结果]
    AQ --> AR[HTTP JSON 响应]

    style X fill:#f8d7da,stroke:#b42318,color:#111
    style AI fill:#fdecc8,stroke:#b54708,color:#111
    style AR fill:#d1fadf,stroke:#027a48,color:#111
```

## 2. 任务状态流转图

```mermaid
stateDiagram-v2
    [*] --> Open: create

    Open --> Assigned: assign\n创建者指派执行者
    Assigned --> InProgress: start\n执行者开始处理
    InProgress --> Submitted: submit\n执行者提交结果
    Submitted --> Assigned: reject\n创建者驳回并退回
    Submitted --> Approved: approve\n创建者验收通过
    Approved --> Completed: close\n创建者正式关闭

    Open --> Cancelled: cancel
    Assigned --> Cancelled: cancel
    InProgress --> Cancelled: cancel
    Submitted --> Cancelled: cancel

    Cancelled --> Open: reactivate\n无 assignee
    Cancelled --> Assigned: reactivate\n保留 assignee
    Completed --> Open: reactivate\n无 assignee
    Completed --> Assigned: reactivate\n保留 assignee

    Open --> [*]: delete\n创建者硬删除 Task/Record/Audit
    Assigned --> [*]: delete
    InProgress --> [*]: delete
    Submitted --> [*]: delete
    Approved --> [*]: delete
    Completed --> [*]: delete
    Cancelled --> [*]: delete
```

## 3. 读图说明

- `POST /users` 是唯一公开写入口；其余任务接口都先走 `UserAuth`。
- `create / assign / start / close` 只写 `Task + AuditLog`；`submit / reject / approve / cancel / reactivate` 会同时写 `TaskRecord + AuditLog`。
- `reject` 会把任务从 `submitted` 退回到 `assigned`，而不是直接回到 `in_progress`，这样可以保留再次 `start` 的显式语义。
- `reactivate` 会根据 `assignee_id` 是否为空，恢复到 `open` 或 `assigned`。
- `delete` 是硬删除：`Task`、`TaskRecord`、`AuditLog` 都会级联清理。

## 4. 主要依据

- 主链与接口范围：`README.md:12`、`README.md:44`、`README.md:129`
- 应用装配与双驱动分支：`internal/bootstrap/app.go:25`
- 路由入口与公开/鉴权分组：`internal/router/router.go:10`
- 鉴权优先级与 401 分支：`internal/middleware/current_user.go:25`
- Handler 的请求解析与错误映射：`internal/handler/task.go:39`、`internal/handler/task.go:297`
- 状态机与留痕规则：`internal/service/task_service.go:70`、`internal/service/task_service.go:188`、`internal/service/task_service.go:246`、`internal/service/task_service.go:298`、`internal/service/task_service.go:367`、`internal/service/task_service.go:436`、`internal/service/task_service.go:505`、`internal/service/task_service.go:557`
- 全链路验证顺序：`internal/handler/task_mongo_e2e_test.go:26`
