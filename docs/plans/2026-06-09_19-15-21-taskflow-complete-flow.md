---
mode: plan
cwd: /Users/asaqelee/taskflow
task: 绘制 TaskFlow 完整流程流转图
complexity: medium
tool: mcp__sequential-thinking__sequentialthinking
total_thoughts: 6
created_at: 2026-06-09T19:15:21+08:00
---

# Plan: 绘制 TaskFlow 完整流程流转图

任务概述
本次任务的目标是基于当前仓库实现，梳理 TaskFlow 从启动装配、身份鉴权、任务动作、状态流转到留痕与删除清理的完整流程，并输出可维护的 Mermaid 图。非目标是不修改业务逻辑、不重构分层、不引入新的文档体系；只新增最小文档资产并附可审计证据。

执行计划
1. 勘察现有实现与测试，确认主入口、路由、状态机和异常分支。
   - 产物：关键事实清单与 `file:line` 引用。
   - 验收信号：能够回答“谁发起、谁处理、何时写 TaskRecord/AuditLog、何时结束”。
   - 影响面：只读源码与现有文档。
   - 回滚思路：无代码变更；若判断错误，重做事实清单即可。
2. 提炼核心数据结构、状态集合和动作权限边界。
   - 产物：一条主链、若干补充分支和异常路径摘要。
   - 验收信号：主流程可压缩为 `create -> assign -> start -> submit -> approve/reject -> close`，并能补充 `cancel / reactivate / delete`。
   - 影响面：仅文档分析层。
   - 回滚思路：回退到代码/测试重新核对状态转换。
3. 生成流程图文档，拆分为“系统处理流”与“状态流转图”。
   - 产物：`docs/taskflow-流程流转图.md`。
   - 验收信号：Mermaid 可读、节点命名与仓库术语一致、包含角色与留痕分支。
   - 影响面：新增文档文件。
   - 回滚思路：删除新增文档或回退本次提交。
4. 记录计划与证据，保证过程可审计。
   - 产物：本 Plan 文件与 `evidence/YY_MM_DD_HHMM-*.md`。
   - 验收信号：证据中包含输入来源、命令、结论、不确定性、风险与回滚点。
   - 影响面：新增文档文件与 `.gitignore` 一行。
   - 回滚思路：回退新增文档与 `.gitignore` 变更。
5. 运行最接近的验证命令，确认流程图与当前实现未脱节。
   - 产物：测试/构建结果与跳过说明。
   - 验收信号：`go test ./...` 成功，或明确记录失败/跳过原因及影响范围。
   - 影响面：只读验证，不改业务逻辑。
   - 回滚思路：验证不产生持久改动，无需回滚。

验证策略
- 构建/静态检查：运行 `go test ./...`，以覆盖编译与测试入口。
- 关键路径测试：依赖 `internal/handler/task_mongo_e2e_test.go` 的完整 15 步工作流与 `internal/service/task_service_test.go` 的状态机单测。
- 异常路径：核对 `UserAuth` 的未授权分支、`writeServiceError` 的 400/403/404/500 映射，以及状态不合法时的拒绝逻辑。
- 回归点：确认路由集合、动作名、状态名、TaskRecord/AuditLog 落点与 README 一致。

风险与注意事项
- 仓库当前工作区已有未提交源码改动，本次只新增文档与忽略规则，不触碰相关源码文件。
- Mongo E2E 依赖 `TASKFLOW_MONGO_TEST_URI`；若环境未提供，该测试会跳过，需在证据中明确说明。
- Mermaid 图若继续塞入 repository 细节会降低可读性，因此保留接口层、状态层与留痕层，不展开每个存储实现细节。

参考
- `README.md:12`
- `README.md:44`
- `internal/bootstrap/app.go:25`
- `internal/router/router.go:10`
- `internal/middleware/current_user.go:25`
- `internal/handler/task.go:39`
- `internal/service/task_service.go:70`
- `internal/service/task_service.go:188`
- `internal/service/task_service.go:298`
- `internal/service/task_service.go:367`
- `internal/service/task_service.go:436`
- `internal/service/task_service.go:505`
- `internal/service/task_service.go:557`
- `internal/handler/task_mongo_e2e_test.go:26`
